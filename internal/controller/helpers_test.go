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

	orchestrationciscocomv1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

var _ = Describe("Helper Functions", func() {
	Context("updatePackageBundlePhase", func() {
		const testResourceName = "test-pb-helper"
		var testPackageBundle *orchestrationciscocomv1alpha1.PackageBundle
		ctx := context.Background()

		BeforeEach(func() {
			testPackageBundle = &orchestrationciscocomv1alpha1.PackageBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
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
			err := k8sClient.Create(ctx, testPackageBundle)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			resource := &orchestrationciscocomv1alpha1.PackageBundle{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: testResourceName, Namespace: "default"}, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should update PackageBundle phase successfully", func() {
			By("Updating the phase to Downloading")
			err := updatePackageBundlePhase(ctx, k8sClient, testPackageBundle,
				orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading,
				"Download in progress", "test-job")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the phase was updated")
			updatedPB := &orchestrationciscocomv1alpha1.PackageBundle{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testResourceName, Namespace: "default"}, updatedPB)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPB.Status.Phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading))
			Expect(updatedPB.Status.Message).To(Equal("Download in progress"))
			Expect(updatedPB.Status.JobName).To(Equal("test-job"))
			Expect(updatedPB.Status.LastTransitionTime).NotTo(BeNil())
		})

		It("should not update if phase is the same", func() {
			By("Setting initial phase")
			err := updatePackageBundlePhase(ctx, k8sClient, testPackageBundle,
				orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading,
				"Download in progress", "test-job")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the last transition time")
			updatedPB := &orchestrationciscocomv1alpha1.PackageBundle{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testResourceName, Namespace: "default"}, updatedPB)
			Expect(err).NotTo(HaveOccurred())
			originalTime := updatedPB.Status.LastTransitionTime

			By("Trying to update to the same phase")
			time.Sleep(time.Millisecond * 100) // Ensure time difference would be visible
			err = updatePackageBundlePhase(ctx, k8sClient, testPackageBundle,
				orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading,
				"Still downloading", "test-job")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying LastTransitionTime was not updated")
			finalPB := &orchestrationciscocomv1alpha1.PackageBundle{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testResourceName, Namespace: "default"}, finalPB)
			Expect(err).NotTo(HaveOccurred())
			Expect(finalPB.Status.LastTransitionTime).To(Equal(originalTime))
		})

		It("should handle non-existent PackageBundle gracefully", func() {
			nonExistentPB := &orchestrationciscocomv1alpha1.PackageBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent",
					Namespace: "default",
				},
			}

			err := updatePackageBundlePhase(ctx, k8sClient, nonExistentPB,
				orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading,
				"Download in progress", "test-job")
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("getJobStatus", func() {
		ctx := context.Background()

		AfterEach(func() {
			// Clean up any test jobs
			jobList := &batchv1.JobList{}
			err := k8sClient.List(ctx, jobList)
			if err == nil {
				for _, job := range jobList.Items {
					if job.Name == "test-job-complete" || job.Name == "test-job-failed" || job.Name == "test-job-running" {
						_ = k8sClient.Delete(ctx, &job)
					}
				}
			}
		})

		It("should return Pending for non-existent job", func() {
			phase, message, err := getJobStatus(ctx, k8sClient, "non-existent-job", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhasePending))
			Expect(message).To(Equal("Job not found"))
		})

		It("should return Downloaded for completed job", func() {
			By("Creating a completed job")
			completedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-complete",
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
			err := k8sClient.Create(ctx, completedJob)
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

			By("Getting job status")
			phase, message, err := getJobStatus(ctx, k8sClient, "test-job-complete", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseDownloaded))
			Expect(message).To(Equal("Package download completed successfully"))
		})

		It("should return FailedToDownload for failed job", func() {
			By("Creating a failed job")
			failedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-failed",
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
						Message: "Pod failed",
					},
				},
				StartTime: &now,
				Failed:    1,
			}
			err = k8sClient.Status().Update(ctx, failedJob)
			Expect(err).NotTo(HaveOccurred())

			By("Getting job status")
			phase, message, err := getJobStatus(ctx, k8sClient, "test-job-failed", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseFailedToDownload))
			Expect(message).To(Equal("Pod failed"))
		})

		It("should return Downloading for running job", func() {
			By("Creating a running job")
			runningJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-running",
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
			err := k8sClient.Create(ctx, runningJob)
			Expect(err).NotTo(HaveOccurred())

			By("Updating job status to running")
			runningJob.Status = batchv1.JobStatus{
				Active: 1,
			}
			err = k8sClient.Status().Update(ctx, runningJob)
			Expect(err).NotTo(HaveOccurred())

			By("Getting job status")
			phase, message, err := getJobStatus(ctx, k8sClient, "test-job-running", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseDownloading))
			Expect(message).To(Equal("Package download in progress"))
		})

		It("should return ContainerCreating for pending job", func() {
			By("Creating a pending job")
			pendingJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-pending",
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
					Active:    0,
					Succeeded: 0,
					Failed:    0,
				},
			}
			err := k8sClient.Create(ctx, pendingJob)
			Expect(err).NotTo(HaveOccurred())

			By("Getting job status")
			phase, message, err := getJobStatus(ctx, k8sClient, "test-job-pending", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(phase).To(Equal(orchestrationciscocomv1alpha1.PackageBundlePhaseContainerCreating))
			Expect(message).To(Equal("Job is creating containers"))

			By("Cleaning up")
			err = k8sClient.Delete(ctx, pendingJob)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("ensureObjectExists", func() {
		ctx := context.Background()

		It("should create object if it doesn't exist", func() {
			testPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc-helper",
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			requeue, err := ensureObjectExists(ctx, k8sClient, testPVC)
			Expect(err).NotTo(HaveOccurred())
			Expect(requeue).To(BeTrue())

			// Verify the PVC was created
			createdPVC := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pvc-helper", Namespace: "default"}, createdPVC)
			Expect(err).NotTo(HaveOccurred())

			// Clean up
			err = k8sClient.Delete(ctx, createdPVC)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return false for existing object", func() {
			testPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc-existing",
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			// Create the PVC first
			err := k8sClient.Create(ctx, testPVC)
			Expect(err).NotTo(HaveOccurred())

			// Now test ensureObjectExists
			requeue, err := ensureObjectExists(ctx, k8sClient, testPVC)
			Expect(err).NotTo(HaveOccurred())
			Expect(requeue).To(BeFalse())

			// Clean up
			err = k8sClient.Delete(ctx, testPVC)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
