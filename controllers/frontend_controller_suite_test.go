package controllers

import (
	"context"
	"time"

	crd "github.com/RedHatInsights/frontend-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Frontend controller", func() {
	var inputJSON apiextensions.JSON

	inputJSON.UnmarshalJSON([]byte(`{
		"title": "Test",
		"groupID": "",
		"navItems": {},
		"appId": "",
		"href": "/test/href"
	}`))

	const (
		FrontendName      = "test-frontend"
		FrontendNamespace = "default"
		FrontendEnvName   = "test-env"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a Frontend Resource", func() {
		It("Should create a deployment with the correct items", func() {
			By("By creating a new Frontend")
			ctx := context.Background()

			frontend := &crd.Frontend{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cloud.redhat.com/v1",
					Kind:       "Frontend",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      FrontendName,
					Namespace: FrontendNamespace,
				},
				Spec: crd.FrontendSpec{
					EnvName:        FrontendEnvName,
					Title:          "",
					DeploymentRepo: "",
					API: crd.ApiInfo{
						Versions: []string{"v1"},
					},
					Frontend: crd.FrontendInfo{
						Paths: []string{"/things/test"},
					},
					Image: "my-image:version",
					Extensions: []crd.Extension{{
						Type:       "cloud.redhat.com/frontend",
						Properties: inputJSON,
					}},
					Module: crd.FedModule{
						ManifestLocation: "/apps/inventory/fed-mods.json",
						Modules: []crd.Module{{
							Id:     "test",
							Module: "./RootApp",
							Routes: []crd.Routes{{
								Pathname: "/test/href",
							}},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, frontend)).Should(Succeed())

			frontendEnvironment := crd.FrontendEnvironment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cloud.redhat.com/v1",
					Kind:       "FrontendEnvironment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      FrontendEnvName,
					Namespace: FrontendNamespace,
				},
				Spec: crd.FrontendEnvironmentSpec{
					SSO:      "https://something-auth",
					Hostname: "something",
				},
			}
			Expect(k8sClient.Create(ctx, &frontendEnvironment)).Should(Succeed())

			bundle := crd.Bundle{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cloud.redhat.com/v1",
					Kind:       "Bundle",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      FrontendEnvName,
					Namespace: FrontendNamespace,
				},
				Spec: crd.BundleSpec{
					ID:      "test-bundle",
					Title:   "",
					AppList: []string{FrontendName},
					EnvName: FrontendEnvName,
				},
			}
			Expect(k8sClient.Create(ctx, &bundle)).Should(Succeed())

			deploymentLookupKey := types.NamespacedName{Name: frontend.Name, Namespace: FrontendNamespace}
			ingressLookupKey := types.NamespacedName{Name: frontend.Name, Namespace: FrontendNamespace}
			configMapLookupKey := types.NamespacedName{Name: frontendEnvironment.Name, Namespace: FrontendNamespace}

			createdDeployment := &apps.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdDeployment.Name).Should(Equal(FrontendName))

			createdIngress := &networking.Ingress{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ingressLookupKey, createdIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdIngress.Name).Should(Equal(FrontendName))

			createdConfigMap := &v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, configMapLookupKey, createdConfigMap)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdConfigMap.Name).Should(Equal(FrontendEnvName))
			Expect(createdConfigMap.Data).Should(Equal(map[string]string{
				"fed-modules.json": "{\"test-frontend\":{\"manifestLocation\":\"/apps/inventory/fed-mods.json\",\"modules\":[{\"id\":\"test\",\"module\":\"./RootApp\",\"routes\":[{\"pathname\":\"/test/href\"}]}]}}",
				"test-env.json":    "{\"id\":\"test-bundle\",\"title\":\"\",\"navItems\":[{\"title\":\"Test\",\"href\":\"/test/href\"}]}",
			}))
		})
	})
})
