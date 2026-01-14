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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orchestrationciscocomv1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

var _ = Describe("PackageBundle Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		packagebundle := &orchestrationciscocomv1alpha1.PackageBundle{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PackageBundle")
			err := k8sClient.Get(ctx, typeNamespacedName, packagebundle)
			if err != nil && errors.IsNotFound(err) {
				resource := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
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
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &orchestrationciscocomv1alpha1.PackageBundle{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PackageBundle")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource (adds finalizer)")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling again to set status")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PackageBundle status was set to Pending")
			updatedPackageBundle := &orchestrationciscocomv1alpha1.PackageBundle{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedPackageBundle)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPackageBundle.Status.Phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhasePending))
		})

		It("should create PVC and Job resources", func() {
			// Create a unique PackageBundle for this test to avoid conflicts
			uniquePB := &orchestrationciscocomv1alpha1.PackageBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource-pvc-job",
					Namespace: "default",
				},
				Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
					TargetName:  "test-nso",
					StorageSize: "1Gi",
					Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
					Source: orchestrationciscocomv1alpha1.PackageSource{
						Url:  "https://github.com/example/test-repo.git",
						Path: "packages",
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

			err := k8sClient.Create(ctx, uniquePB)
			Expect(err).NotTo(HaveOccurred())

			uniqueNamespacedName := types.NamespacedName{
				Name:      uniquePB.Name,
				Namespace: uniquePB.Namespace,
			}

			By("Reconciling multiple times to create resources")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Verifying PVC was created")
			pvcName := uniquePB.Name + "-test-nso"
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				// Keep reconciling to progress through states
				_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: uniqueNamespacedName,
				})
				return k8sClient.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: "default"}, pvc)
			}, time.Second*10, time.Millisecond*500).Should(Succeed())
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))

			By("Verifying Job was created")
			jobName := "download-" + uniquePB.Name
			job := &batchv1.Job{}
			Eventually(func() error {
				// Keep reconciling to progress through states
				_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: uniqueNamespacedName,
				})
				return k8sClient.Get(ctx, types.NamespacedName{Name: jobName, Namespace: "default"}, job)
			}, time.Second*10, time.Millisecond*500).Should(Succeed())
			Expect(job.Spec.Template.Spec.InitContainers[0].Image).To(Equal("alpine/git"))
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("carlosgrillet/cisco-nso:6.1.19-build"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, uniquePB)).To(Succeed())
		})

		It("should update status based on Job completion", func() {
			// Create a unique PackageBundle for this test
			uniquePB := &orchestrationciscocomv1alpha1.PackageBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource-job-complete",
					Namespace: "default",
				},
				Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
					TargetName:  "test-nso",
					StorageSize: "1Gi",
					Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
					Source: orchestrationciscocomv1alpha1.PackageSource{
						Url:  "https://github.com/example/test-repo.git",
						Path: "packages",
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

			err := k8sClient.Create(ctx, uniquePB)
			Expect(err).NotTo(HaveOccurred())

			uniqueNamespacedName := types.NamespacedName{
				Name:      uniquePB.Name,
				Namespace: uniquePB.Namespace,
			}

			By("Creating a completed Job manually to simulate completion")
			jobName := "download-" + uniquePB.Name
			completedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: "default",
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{{
								Name:  "test",
								Image: "alpine",
							}},
						},
					},
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					}},
					Succeeded: 1,
				},
			}

			err = k8sClient.Create(ctx, completedJob)
			Expect(err).NotTo(HaveOccurred())

			By("Updating job status to completed")
			now := metav1.Now()
			completedJob.Status = batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobSuccessCriteriaMet,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
				},
				StartTime:      &now,
				CompletionTime: &now,
				Succeeded:      1,
			}
			err = k8sClient.Status().Update(ctx, completedJob)
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling to check Job status (adds finalizer first)")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: uniqueNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling again to process job status")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: uniqueNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PackageBundle status was updated to Downloaded")
			Eventually(func() orchestrationciscocomv1alpha1.PackageBundlePhase {
				updatedPackageBundle := &orchestrationciscocomv1alpha1.PackageBundle{}
				err := k8sClient.Get(ctx, uniqueNamespacedName, updatedPackageBundle)
				if err != nil {
					return ""
				}
				// Keep reconciling to progress
				_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: uniqueNamespacedName,
				})
				return updatedPackageBundle.Status.Phase
			}, time.Second*10, time.Millisecond*100).Should(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseDownloaded))

			// Cleanup
			Expect(k8sClient.Delete(ctx, uniquePB)).To(Succeed())
			Expect(k8sClient.Delete(ctx, completedJob)).To(Succeed())
		})

		It("should handle Job failure correctly", func() {
			By("Creating a failed Job manually to simulate failure")
			jobName := "download-" + resourceName + "-failed"
			failedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: "default",
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{{
								Name:  "test",
								Image: "alpine",
							}},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, failedJob)
			Expect(err).NotTo(HaveOccurred())

			By("Updating job status to failed")
			now := metav1.Now()
			failedJob.Status = batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailureTarget,
						Status: corev1.ConditionTrue,
					},
					{
						Type:    batchv1.JobFailed,
						Status:  corev1.ConditionTrue,
						Message: "Job failed due to test error",
					},
				},
				StartTime: &now,
				Failed:    1,
			}
			err = k8sClient.Status().Update(ctx, failedJob)
			Expect(err).NotTo(HaveOccurred())

			By("Testing getJobStatus function with failed job")
			phase, message, err := getJobStatus(ctx, k8sClient, jobName, "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseFailedToDownload))
			Expect(message).To(Equal("Job failed due to test error"))
		})
	})

	Context("PackageBundle Finalizer and Cleanup", func() {
		var pbReconciler *PackageBundleReconciler
		ctx := context.Background()

		BeforeEach(func() {
			pbReconciler = &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		Describe("Finalizer Management", func() {
			It("should add finalizer to new PackageBundle", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-finalizer",
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

				By("Creating PackageBundle without finalizer")
				err := k8sClient.Create(ctx, testPB)
				Expect(err).NotTo(HaveOccurred())

				By("Reconciling to add finalizer")
				_, err = pbReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      testPB.Name,
						Namespace: testPB.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				By("Checking finalizer was added")
				updatedPB := &orchestrationciscocomv1alpha1.PackageBundle{}
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      testPB.Name,
					Namespace: testPB.Namespace,
				}, updatedPB)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedPB.Finalizers).To(ContainElement("packagebundle.orchestration.cisco.com/finalizer"))

				// Cleanup
				err = k8sClient.Delete(ctx, testPB)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("unmountPVCFromNSO", func() {
			It("should remove PVC volume from NSO when PackageBundle is deleted", func() {
				// Create test NSO with mounted PVC
				testNSO := &orchestrationciscocomv1alpha1.NSO{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nso-cleanup",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.NSOSpec{
						Image:       "nso:latest",
						ServiceName: "test-nso-svc",
						LabelSelector: map[string]string{
							"app": "nso-test-cleanup",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
						},
						AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
							Username:          "admin",
							PasswordSecretRef: "admin-secret",
						},
						Volumes: []corev1.Volume{
							{
								Name: "package-test-cleanup-pb",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "test-pvc",
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "package-test-cleanup-pb",
								MountPath: "/nso/run/packages",
							},
						},
					},
				}

				err := k8sClient.Create(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())

				// Create PackageBundle
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cleanup-pb",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso-cleanup",
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

				By("Calling unmountPVCFromNSO")
				err = pbReconciler.unmountPVCFromNSO(ctx, testPB)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying volume was removed from NSO")
				updatedNSO := &orchestrationciscocomv1alpha1.NSO{}
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      testNSO.Name,
					Namespace: testNSO.Namespace,
				}, updatedNSO)
				Expect(err).NotTo(HaveOccurred())

				// Check volume was removed
				volumeFound := false
				for _, vol := range updatedNSO.Spec.Volumes {
					if vol.Name == "package-test-cleanup-pb" {
						volumeFound = true
						break
					}
				}
				Expect(volumeFound).To(BeFalse(), "Volume should be removed from NSO")

				// Check volume mount was removed
				mountFound := false
				for _, mount := range updatedNSO.Spec.VolumeMounts {
					if mount.Name == "package-test-cleanup-pb" {
						mountFound = true
						break
					}
				}
				Expect(mountFound).To(BeFalse(), "Volume mount should be removed from NSO")

				// Cleanup
				err = k8sClient.Delete(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle gracefully when NSO does not exist", func() {
				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-no-nso",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "nonexistent-nso",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url: "https://github.com/example/packages.git",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{Image: "alpine/git"},
							Build:    orchestrationciscocomv1alpha1.ContainerParams{Image: "nso:latest"},
						},
					},
				}

				By("Calling unmountPVCFromNSO with nonexistent NSO")
				err := pbReconciler.unmountPVCFromNSO(ctx, testPB)
				Expect(err).NotTo(HaveOccurred(), "Should not error when NSO doesn't exist")
			})

			It("should handle gracefully when volume is not mounted", func() {
				// Create NSO without the package volume
				testNSO := &orchestrationciscocomv1alpha1.NSO{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nso-no-volume",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.NSOSpec{
						Image:       "nso:latest",
						ServiceName: "test-nso-svc",
						LabelSelector: map[string]string{
							"app": "nso-test",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
						},
						AdminCredentials: orchestrationciscocomv1alpha1.Credentials{
							Username:          "admin",
							PasswordSecretRef: "admin-secret",
						},
					},
				}

				err := k8sClient.Create(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())

				testPB := &orchestrationciscocomv1alpha1.PackageBundle{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pb-no-mount",
						Namespace: "default",
					},
					Spec: orchestrationciscocomv1alpha1.PackageBundleSpec{
						TargetName:  "test-nso-no-volume",
						StorageSize: "1Gi",
						Origin:      orchestrationciscocomv1alpha1.OriginTypeSCM,
						Source: orchestrationciscocomv1alpha1.PackageSource{
							Url: "https://github.com/example/packages.git",
						},
						Config: orchestrationciscocomv1alpha1.PackageConfig{
							Download: orchestrationciscocomv1alpha1.ContainerParams{Image: "alpine/git"},
							Build:    orchestrationciscocomv1alpha1.ContainerParams{Image: "nso:latest"},
						},
					},
				}

				By("Calling unmountPVCFromNSO when volume is not mounted")
				err = pbReconciler.unmountPVCFromNSO(ctx, testPB)
				Expect(err).NotTo(HaveOccurred(), "Should not error when volume is not mounted")

				// Cleanup
				err = k8sClient.Delete(ctx, testNSO)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
