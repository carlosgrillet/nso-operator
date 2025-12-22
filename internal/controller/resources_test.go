/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orchestrationciscocomv1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

var _ = Describe("Resource Creation Functions", func() {
	Context("NSO Resources", func() {
		var nsoReconciler *NSOReconciler
		var testNSO *orchestrationciscocomv1alpha1.NSO
		ctx := context.Background()

		BeforeEach(func() {
			nsoReconciler = &NSOReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			testNSO = &orchestrationciscocomv1alpha1.NSO{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nso-resources",
					Namespace: "default",
				},
				Spec: orchestrationciscocomv1alpha1.NSOSpec{
					Image:       "test-nso:latest",
					ServiceName: "test-nso-service",
					Replicas:    2,
					LabelSelector: map[string]string{
						"app": "nso-test",
					},
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 8080,
						},
						{
							Name: "https",
							Port: 8888,
						},
					},
					NsoConfigRef: "test-nso-config",
					AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
						Username:          "admin",
						PasswordSecretRef: "test-admin-secret",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "TEST_ENV",
							Value: "test-value",
						},
					},
				},
			}
		})

		Describe("newService", func() {
			It("should create a headless service with correct specifications", func() {
				// First create the NSO so controller reference can be set
				err := k8sClient.Create(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())

				service := nsoReconciler.newService(ctx, testNSO)

				Expect(service.Name).To(Equal("test-nso-service"))
				Expect(service.Namespace).To(Equal("default"))
				Expect(service.Labels).To(Equal(testNSO.Spec.LabelSelector))
				Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
				Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))
				Expect(service.Spec.Selector).To(Equal(testNSO.Spec.LabelSelector))
				Expect(service.Spec.Ports).To(Equal(testNSO.Spec.Ports))

				// Clean up
				err = k8sClient.Delete(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("newStatefulSet", func() {
			It("should create a statefulset with correct specifications", func() {
				// Create a separate NSO for this test
				testNSO2 := &orchestrationciscocomv1alpha1.NSO{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nso-statefulset",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.NSOSpec{
						Image:       "test-nso:latest",
						ServiceName: "test-nso-service",
						Replicas:    2,
						LabelSelector: map[string]string{
							"app": "nso-test",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
							{
								Name: "https",
								Port: 8888,
							},
						},
						NsoConfigRef: "test-nso-config",
						AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
							Username:          "admin",
							PasswordSecretRef: "test-admin-secret",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "test-value",
							},
						},
					},
				}
				err := k8sClient.Create(ctx, testNSO2)
				Expect(err).NotTo(HaveOccurred())

				statefulSet := nsoReconciler.newStatefulSet(ctx, testNSO2)

				Expect(statefulSet.Name).To(Equal("test-nso-statefulset"))
				Expect(statefulSet.Namespace).To(Equal("default"))
				Expect(statefulSet.Spec.ServiceName).To(Equal("test-nso-service"))
				Expect(*statefulSet.Spec.Replicas).To(Equal(int32(2)))
				Expect(statefulSet.Spec.Selector.MatchLabels).To(Equal(testNSO2.Spec.LabelSelector))

				// Check pod template
				podTemplate := statefulSet.Spec.Template
				Expect(podTemplate.Labels).To(Equal(testNSO2.Spec.LabelSelector))

				// Check container specifications
				containers := podTemplate.Spec.Containers
				Expect(containers).To(HaveLen(1))

				container := containers[0]
				Expect(container.Name).To(Equal("ncs"))
				Expect(container.Image).To(Equal("test-nso:latest"))
				Expect(container.Ports).To(HaveLen(2))
				Expect(container.Ports[0].ContainerPort).To(Equal(int32(8080)))
				Expect(container.Ports[1].ContainerPort).To(Equal(int32(8888)))

				// Check environment variables
				envVars := container.Env
				Expect(envVars).To(HaveLen(3)) // ADMIN_USERNAME, ADMIN_PASSWORD, TEST_ENV

				// Check admin username env var
				adminUsernameVar := envVars[0]
				Expect(adminUsernameVar.Name).To(Equal("ADMIN_USERNAME"))
				Expect(adminUsernameVar.Value).To(Equal("admin"))

				// Check admin password env var (from secret)
				adminPasswordVar := envVars[1]
				Expect(adminPasswordVar.Name).To(Equal("ADMIN_PASSWORD"))
				Expect(adminPasswordVar.ValueFrom).NotTo(BeNil())
				Expect(adminPasswordVar.ValueFrom.SecretKeyRef.Name).To(Equal("test-admin-secret"))
				Expect(adminPasswordVar.ValueFrom.SecretKeyRef.Key).To(Equal("password"))

				// Check custom env var
				customEnvVar := envVars[2]
				Expect(customEnvVar.Name).To(Equal("TEST_ENV"))
				Expect(customEnvVar.Value).To(Equal("test-value"))

				// Check volume mounts
				volumeMounts := container.VolumeMounts
				Expect(volumeMounts).To(HaveLen(2))
				Expect(volumeMounts[0].Name).To(Equal("ncs-config"))
				Expect(volumeMounts[0].MountPath).To(Equal("/etc/ncs/ncs.conf"))
				Expect(volumeMounts[0].SubPath).To(Equal("ncs.conf"))
				Expect(volumeMounts[1].Name).To(Equal("cdb-storage"))
				Expect(volumeMounts[1].MountPath).To(Equal("/nso/run/cdb"))

				// Check volumes
				volumes := podTemplate.Spec.Volumes
				Expect(volumes).To(HaveLen(2))
				volume := volumes[0]
				Expect(volume.Name).To(Equal("ncs-config"))
				Expect(volume.ConfigMap).NotTo(BeNil())
				Expect(volume.ConfigMap.Name).To(Equal("ncs-config"))
				Expect(volume.ConfigMap.Items).To(HaveLen(1))
				Expect(volume.ConfigMap.Items[0].Key).To(Equal("ncs.conf"))
				Expect(volume.ConfigMap.Items[0].Path).To(Equal("ncs.conf"))
				Expect(*volume.ConfigMap.Items[0].Mode).To(Equal(int32(0600)))

				cdbVolume := volumes[1]
				Expect(cdbVolume.Name).To(Equal("cdb-storage"))
				Expect(cdbVolume.PersistentVolumeClaim).NotTo(BeNil())
				Expect(cdbVolume.PersistentVolumeClaim.ClaimName).To(Equal("test-nso-statefulset-cdb"))

				// Clean up
				err = k8sClient.Delete(ctx, testNSO2)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("newCDBPersistentVolumeClaim", func() {
			It("should create CDB PVC with default size", func() {
				// Create a separate NSO for this test
				testNSO3 := &orchestrationciscocomv1alpha1.NSO{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nso-cdb",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.NSOSpec{
						Image:       "test-nso:latest",
						ServiceName: "test-nso-service",
						Replicas:    1,
						LabelSelector: map[string]string{
							"app": "nso-test",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
							{
								Name: "https",
								Port: 8888,
							},
						},
						NsoConfigRef: "test-nso-config",
						AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
							Username:          "admin",
							PasswordSecretRef: "test-admin-secret",
						},
					},
				}

				err := k8sClient.Create(ctx, testNSO3)
				Expect(err).NotTo(HaveOccurred())

				pvc := nsoReconciler.newCDBPersistentVolumeClaim(ctx, testNSO3)

				Expect(pvc.Name).To(Equal("test-nso-cdb-cdb"))
				Expect(pvc.Namespace).To(Equal("default"))
				Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("5Gi")))
				Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))

				// Clean up
				err = k8sClient.Delete(ctx, testNSO3)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should create CDB PVC with custom size", func() {
				// Create NSO with custom CDB size
				testNSO4 := &orchestrationciscocomv1alpha1.NSO{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nso-cdb-custom",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.NSOSpec{
						Image:          "test-nso:latest",
						ServiceName:    "test-nso-service",
						Replicas:       1,
						CDBStorageSize: "10Gi",
						LabelSelector: map[string]string{
							"app": "nso-test",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
							{
								Name: "https",
								Port: 8888,
							},
						},
						NsoConfigRef: "test-nso-config",
						AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
							Username:          "admin",
							PasswordSecretRef: "test-admin-secret",
						},
					},
				}

				err := k8sClient.Create(ctx, testNSO4)
				Expect(err).NotTo(HaveOccurred())

				pvc := nsoReconciler.newCDBPersistentVolumeClaim(ctx, testNSO4)

				Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("10Gi")))

				// Clean up
				err = k8sClient.Delete(ctx, testNSO4)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("PackageBundle Resources", func() {
		var packageBundleReconciler *PackageBundleReconciler
		var testPackageBundle *orchestrationciscocomv1alpha1.PackageBundle
		ctx := context.Background()

		BeforeEach(func() {
			packageBundleReconciler = &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			testPackageBundle = &orchestrationciscocomv1alpha1.PackageBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pb-resources",
					Namespace: "default",
				},
				Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
					TargetName:  "test-nso",
					StorageSize: "2Gi",
					Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
					Source: orchestrationciscocomv1alpha1.PackageSource{
						Url:    "https://github.com/example/test-repo.git",
						Branch: "main",
						Path:   "packages",
					},
				},
			}
		})

		Describe("newPersistenVolumeClaim", func() {
			It("should create a PVC with correct specifications", func() {
				// Create PackageBundle for PVC test
				err := k8sClient.Create(ctx, testPackageBundle)
				Expect(err).NotTo(HaveOccurred())

				pvc := packageBundleReconciler.newPersistenVolumeClaim(ctx, testPackageBundle)

				expectedName := "test-pb-resources-test-nso"
				Expect(pvc.Name).To(Equal(expectedName))
				Expect(pvc.Namespace).To(Equal("default"))
				Expect(pvc.Spec.AccessModes).To(HaveLen(1))
				Expect(pvc.Spec.AccessModes[0]).To(Equal(corev1.ReadWriteOnce))

				expectedSize := resource.MustParse("2Gi")
				Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(expectedSize))

				// Clean up
				err = k8sClient.Delete(ctx, testPackageBundle)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use default storage size when not specified", func() {
				// Create a separate PackageBundle without StorageSize
				testPB2 := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-default-size",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName: "test-nso",
						Origin:     orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url: "https://github.com/example/test-repo.git",
						},
					},
				}
				err := k8sClient.Create(ctx, testPB2)
				Expect(err).NotTo(HaveOccurred())

				pvc := packageBundleReconciler.newPersistenVolumeClaim(ctx, testPB2)

				expectedSize := resource.MustParse("1Gi")
				Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(expectedSize))

				// Clean up
				err = k8sClient.Delete(ctx, testPB2)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("newJob", func() {
			It("should create a Job with correct specifications", func() {
				// First create the PackageBundle so controller reference can be set
				err := k8sClient.Create(ctx, testPackageBundle)
				Expect(err).NotTo(HaveOccurred())

				job := packageBundleReconciler.newJob(ctx, testPackageBundle)

				expectedJobName := "download-test-pb-resources"
				expectedPVCName := "test-pb-resources-test-nso"

				Expect(job.Name).To(Equal(expectedJobName))
				Expect(job.Namespace).To(Equal("default"))

				// Check job specifications
				Expect(*job.Spec.TTLSecondsAfterFinished).To(Equal(int32(300)))
				Expect(*job.Spec.BackoffLimit).To(Equal(int32(3)))

				// Check pod template
				podTemplate := job.Spec.Template
				Expect(podTemplate.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyNever))

				// Check init containers
				initContainers := podTemplate.Spec.InitContainers
				Expect(initContainers).To(HaveLen(1))

				initContainer := initContainers[0]
				Expect(initContainer.Name).To(Equal("downloader"))
				Expect(initContainer.Image).To(Equal("alpine/git"))
				Expect(initContainer.Command).To(Equal([]string{"git", "clone", "--depth", "1", "https://github.com/example/test-repo.git", "/repo"}))

				// Check init container volume mounts
				initVolumeMounts := initContainer.VolumeMounts
				Expect(initVolumeMounts).To(HaveLen(1))
				Expect(initVolumeMounts[0].Name).To(Equal("package-storage"))
				Expect(initVolumeMounts[0].MountPath).To(Equal("/repo"))

				// Check security context
				Expect(initContainer.SecurityContext).NotTo(BeNil())
				Expect(*initContainer.SecurityContext.RunAsNonRoot).To(BeTrue())
				Expect(*initContainer.SecurityContext.RunAsUser).To(Equal(int64(1000)))
				Expect(*initContainer.SecurityContext.AllowPrivilegeEscalation).To(BeFalse())

				// Check main containers
				containers := podTemplate.Spec.Containers
				Expect(containers).To(HaveLen(1))

				container := containers[0]
				Expect(container.Name).To(Equal("builder"))
				Expect(container.Image).To(Equal("carlosgrillet/cisco-nso:6.1.19-build"))
				Expect(container.Command).To(Equal([]string{"/bin/sh", "-c"}))
				Expect(container.Args).To(HaveLen(1))
				Expect(container.Args[0]).To(ContainSubstring("for dir in packages/*/src"))
				Expect(container.WorkingDir).To(Equal("/repo"))

				// Check main container volume mounts
				volumeMounts := container.VolumeMounts
				Expect(volumeMounts).To(HaveLen(1))
				Expect(volumeMounts[0].Name).To(Equal("package-storage"))
				Expect(volumeMounts[0].MountPath).To(Equal("/repo"))

				// Check volumes
				volumes := podTemplate.Spec.Volumes
				Expect(volumes).To(HaveLen(1))
				volume := volumes[0]
				Expect(volume.Name).To(Equal("package-storage"))
				Expect(volume.PersistentVolumeClaim).NotTo(BeNil())
				Expect(volume.PersistentVolumeClaim.ClaimName).To(Equal(expectedPVCName))

				// Clean up
				err = k8sClient.Delete(ctx, testPackageBundle)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle different source configurations", func() {
				// Create a separate PackageBundle for this test
				differentPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-different",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName: "test-nso",
						Origin:     orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:  "https://github.com/another/repo.git",
							Path: "/my-packages",
						},
					},
				}
				err := k8sClient.Create(ctx, differentPB)
				Expect(err).NotTo(HaveOccurred())

				job := packageBundleReconciler.newJob(ctx, differentPB)

				// Check init container has correct command with the different URL
				Expect(job.Spec.Template.Spec.InitContainers).To(HaveLen(1))
				initContainer := job.Spec.Template.Spec.InitContainers[0]
				Expect(initContainer.Command).To(Equal([]string{"git", "clone", "--depth", "1", "https://github.com/another/repo.git", "/repo"}))

				// Check main container exists and uses the correct path
				Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := job.Spec.Template.Spec.Containers[0]
				Expect(container.Name).To(Equal("builder"))
				Expect(container.Args[0]).To(ContainSubstring("my-packages/*/src"))

				// Clean up
				err = k8sClient.Delete(ctx, differentPB)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("Helper Functions", func() {
		Describe("normalizePathString", func() {
			It("should trim leading and trailing slashes", func() {
				result := normalizePathString("/packages/")
				Expect(result).To(Equal("packages"))
			})

			It("should trim whitespace", func() {
				result := normalizePathString("  /packages/  ")
				Expect(result).To(Equal("packages"))
			})

			It("should handle path without slashes", func() {
				result := normalizePathString("packages")
				Expect(result).To(Equal("packages"))
			})

			It("should handle nested paths", func() {
				result := normalizePathString("/path/to/packages/")
				Expect(result).To(Equal("path/to/packages"))
			})

			It("should handle empty string", func() {
				result := normalizePathString("")
				Expect(result).To(Equal(""))
			})

			It("should handle just slashes", func() {
				result := normalizePathString("///")
				Expect(result).To(Equal(""))
			})
		})
	})
})
