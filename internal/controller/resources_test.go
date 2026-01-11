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
					Config: orchestrationciscocomv1alpha1.PackageConfig{
						Download: orchestrationciscocomv1alpha1.ContainerParams{
							Image: "alpine/git",
						},
						Build: orchestrationciscocomv1alpha1.ContainerParams{
							Image: "carlosgrillet/cisco-nso:6.1.19-build",
						},
					},
				},
			}
		})

		Describe("newPersistentVolumeClaim", func() {
			It("should create a PVC with correct specifications", func() {
				// Create PackageBundle for PVC test
				err := k8sClient.Create(ctx, testPackageBundle)
				Expect(err).NotTo(HaveOccurred())

				pvc := packageBundleReconciler.newPersistentVolumeClaim(ctx, testPackageBundle)

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
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "carlosgrillet/cisco-nso:6.1.19-build",
							},
						},
					},
				}
				err := k8sClient.Create(ctx, testPB2)
				Expect(err).NotTo(HaveOccurred())

				pvc := packageBundleReconciler.newPersistentVolumeClaim(ctx, testPB2)

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
				Expect(initContainer.Name).To(Equal("test-pb-resources-downloader"))
				Expect(initContainer.Image).To(Equal("alpine/git"))
				Expect(initContainer.Command).To(Equal([]string{"git", "clone", "--depth", "1", "--branch", "main", "https://github.com/example/test-repo.git", "/repo"}))

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
				Expect(container.Name).To(Equal("test-pb-resources-builder"))
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
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "carlosgrillet/cisco-nso:6.1.19-build",
							},
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
				Expect(container.Name).To(Equal("test-pb-different-builder"))
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

	Context("PackageBundle Job Creation with Git Features", func() {
		var pbReconciler *PackageBundleReconciler
		ctx := context.Background()

		BeforeEach(func() {
			pbReconciler = &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		Describe("newJob with Git Branch Support", func() {
			It("should include branch in git clone command when branch is specified", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-branch",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:    "https://github.com/example/packages.git",
							Branch: "develop",
							Path:   "packages",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "nso-builder:latest",
							},
						},
					},
				}

				job := pbReconciler.newJob(ctx, testPB)

				// Check that git clone command includes --branch flag
				initContainer := job.Spec.Template.Spec.InitContainers[0]
				Expect(initContainer.Command).To(ContainElement("--branch"))
				Expect(initContainer.Command).To(ContainElement("develop"))

				// Verify command structure: git clone --depth 1 --branch develop <url> <path>
				cmdIndex := 0
				for i, arg := range initContainer.Command {
					if arg == "git" {
						cmdIndex = i
						break
					}
				}
				Expect(initContainer.Command[cmdIndex]).To(Equal("git"))
				Expect(initContainer.Command[cmdIndex+1]).To(Equal("clone"))
				Expect(initContainer.Command).To(ContainElement("--depth"))
				Expect(initContainer.Command).To(ContainElement("1"))
			})

			It("should not include branch flag when branch is empty", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-no-branch",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:  "https://github.com/example/packages.git",
							Path: "packages",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "nso-builder:latest",
							},
						},
					},
				}

				job := pbReconciler.newJob(ctx, testPB)

				// Check that git clone command does NOT include --branch flag
				initContainer := job.Spec.Template.Spec.InitContainers[0]
				Expect(initContainer.Command).NotTo(ContainElement("--branch"))
			})
		})

		Describe("newJob with SSH Authentication", func() {
			It("should mount SSH key and configure environment for git@ URLs", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-ssh",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Credentials: orchestrationciscocomv1alpha1.AccessCredentials{
							SshKeySecretRef: "ssh-key-secret",
						},
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:  "git@github.com:example/packages.git",
							Path: "packages",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "nso-builder:latest",
							},
						},
					},
				}

				job := pbReconciler.newJob(ctx, testPB)

				// Check SSH key volume is mounted
				volumes := job.Spec.Template.Spec.Volumes
				sshVolumeFound := false
				for _, vol := range volumes {
					if vol.Name == sshVolumeName {
						sshVolumeFound = true
						Expect(vol.Secret).NotTo(BeNil())
						Expect(vol.Secret.SecretName).To(Equal("ssh-key-secret"))
						break
					}
				}
				Expect(sshVolumeFound).To(BeTrue(), "SSH key volume should be present")

				// Check SSH key volume mount in init container
				initContainer := job.Spec.Template.Spec.InitContainers[0]
				sshMountFound := false
				for _, mount := range initContainer.VolumeMounts {
					if mount.Name == sshVolumeName {
						sshMountFound = true
						Expect(mount.MountPath).To(Equal("/.ssh"))
						Expect(mount.ReadOnly).To(BeTrue())
						break
					}
				}
				Expect(sshMountFound).To(BeTrue(), "SSH key mount should be present in init container")

				// Check environment variables for SSH
				homeEnvFound := false
				gitSSHEnvFound := false
				for _, env := range initContainer.Env {
					if env.Name == "HOME" {
						homeEnvFound = true
						Expect(env.Value).To(Equal("/tmp"))
					}
					if env.Name == "GIT_SSH_COMMAND" {
						gitSSHEnvFound = true
						Expect(env.Value).To(ContainSubstring("ssh -i /.ssh/id_rsa"))
						Expect(env.Value).To(ContainSubstring("StrictHostKeyChecking=no"))
					}
				}
				Expect(homeEnvFound).To(BeTrue(), "HOME environment variable should be set")
				Expect(gitSSHEnvFound).To(BeTrue(), "GIT_SSH_COMMAND environment variable should be set")
			})

			It("should NOT mount SSH key for HTTPS URLs even if sshKeySecretRef is set", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-https-no-ssh",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Credentials: orchestrationciscocomv1alpha1.AccessCredentials{
							SshKeySecretRef: "ssh-key-secret",
						},
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:  "https://github.com/example/packages.git",
							Path: "packages",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "nso-builder:latest",
							},
						},
					},
				}

				job := pbReconciler.newJob(ctx, testPB)

				// Check SSH key volume is NOT mounted
				volumes := job.Spec.Template.Spec.Volumes
				for _, vol := range volumes {
					Expect(vol.Name).NotTo(Equal(sshVolumeName), "SSH key volume should NOT be present for HTTPS URLs")
				}

				// Check init container has no SSH environment variables
				initContainer := job.Spec.Template.Spec.InitContainers[0]
				for _, env := range initContainer.Env {
					Expect(env.Name).NotTo(Equal("GIT_SSH_COMMAND"), "GIT_SSH_COMMAND should NOT be set for HTTPS URLs")
				}

				// Check security context is standard (RunAsUser=1000)
				Expect(initContainer.SecurityContext).NotTo(BeNil())
				Expect(initContainer.SecurityContext.RunAsUser).NotTo(BeNil())
				Expect(*initContainer.SecurityContext.RunAsUser).To(Equal(int64(1000)))
			})

			It("should use different security context for SSH vs HTTPS", func() {
				// SSH URL test
				sshPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ssh", Namespace: "default"},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Credentials: orchestrationciscocomv1alpha1.AccessCredentials{
							SshKeySecretRef: "ssh-key",
						},
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url: "git@github.com:example/repo.git",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{Image: "alpine/git"},
							Build:    orchestrationciscocomv1alpha1.ContainerParams{Image: "nso:latest"},
						},
					},
				}

				sshJob := pbReconciler.newJob(ctx, sshPB)
				sshInitContainer := sshJob.Spec.Template.Spec.InitContainers[0]

				// SSH should NOT have RunAsUser set (runs as default container user)
				if sshInitContainer.SecurityContext != nil {
					Expect(sshInitContainer.SecurityContext.RunAsUser).To(BeNil())
				}

				// HTTPS URL test
				httpsPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{Name: "test-https", Namespace: "default"},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url: "https://github.com/example/repo.git",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{Image: "alpine/git"},
							Build:    orchestrationciscocomv1alpha1.ContainerParams{Image: "nso:latest"},
						},
					},
				}

				httpsJob := pbReconciler.newJob(ctx, httpsPB)
				httpsInitContainer := httpsJob.Spec.Template.Spec.InitContainers[0]

				// HTTPS should have RunAsUser=1000
				Expect(httpsInitContainer.SecurityContext).NotTo(BeNil())
				Expect(httpsInitContainer.SecurityContext.RunAsUser).NotTo(BeNil())
				Expect(*httpsInitContainer.SecurityContext.RunAsUser).To(Equal(int64(1000)))
			})
		})

		Describe("newJob with combined SSH and Branch", func() {
			It("should support both SSH authentication and branch selection", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-ssh-branch",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Credentials: orchestrationciscocomv1alpha1.AccessCredentials{
							SshKeySecretRef: "ssh-key-secret",
						},
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url:    "git@github.com:example/packages.git",
							Branch: "feature/new-packages",
							Path:   "packages",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "alpine/git",
							},
							Build: orchestrationciscocomv1alpha1.ContainerParams{
								Image: "nso-builder:latest",
							},
						},
					},
				}

				job := pbReconciler.newJob(ctx, testPB)

				// Check both SSH and branch are configured
				initContainer := job.Spec.Template.Spec.InitContainers[0]

				// Branch in command
				Expect(initContainer.Command).To(ContainElement("--branch"))
				Expect(initContainer.Command).To(ContainElement("feature/new-packages"))

				// SSH environment variables
				gitSSHEnvFound := false
				for _, env := range initContainer.Env {
					if env.Name == "GIT_SSH_COMMAND" {
						gitSSHEnvFound = true
						break
					}
				}
				Expect(gitSSHEnvFound).To(BeTrue())

				// SSH volume mount
				sshMountFound := false
				for _, mount := range initContainer.VolumeMounts {
					if mount.Name == sshVolumeName && mount.MountPath == "/.ssh" {
						sshMountFound = true
						break
					}
				}
				Expect(sshMountFound).To(BeTrue())
			})
		})
	})
})
