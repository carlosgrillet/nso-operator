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

	orchestrationciscocomv1alpha1 "wwwin-github.cisco.com/cgrillet/nso-operator/api/v1alpha1"
)

var _ = Describe("PackageBundle Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
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
			By("Reconciling the created resource")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
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
			By("Reconciling the resource")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PVC was created")
			pvcName := resourceName + "-test-nso"
			pvc := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: "default"}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))

			By("Verifying Job was created")
			jobName := "download-" + resourceName
			job := &batchv1.Job{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: jobName, Namespace: "default"}, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("alpine/git"))
		})

		It("should update status based on Job completion", func() {
			By("Creating a completed Job manually to simulate completion")
			jobName := "download-" + resourceName
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

			err := k8sClient.Create(ctx, completedJob)
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

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

			By("Reconciling again to check Job status")
			controllerReconciler := &PackageBundleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PackageBundle status was updated to Downloaded")
			Eventually(func() orchestrationciscocomv1alpha1.PackageBundlePhase {
				updatedPackageBundle := &orchestrationciscocomv1alpha1.PackageBundle{}
				err := k8sClient.Get(ctx, typeNamespacedName, updatedPackageBundle)
				if err != nil {
					return ""
				}
				return updatedPackageBundle.Status.Phase
			}, time.Second*10, time.Millisecond*100).Should(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseDownloaded))
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
})
