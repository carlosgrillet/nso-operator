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
	"errors"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nsov1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

const (
	// Reconciliation requeue intervals
	requeueIntervalMountRetry = 15 * time.Second
	requeueIntervalJobStatus  = 30 * time.Second

	// NSO container and command configuration
	nsoContainerName        = "ncs"
	nsoCliCommand           = "ncs_cli"
	nsoCliUserFlag          = "-Cu"
	nsoPackageReloadCommand = "packages reload"
	nsoPackagesMountPath    = "/nso/run/packages"

	// Volume and storage configuration
	defaultStorageSize   = "1Gi"
	packageVolumeNameFmt = "package-%s"
	downloadJobNameFmt   = "download-%s"
	jobVolumeName        = "package-storage"
	jobVolumeMountPath   = "/repo"

	// Finalizer for cleanup
	packageBundleFinalizer = "packagebundle.orchestration.cisco.com/finalizer"
)

// Sentinel errors for control flow
var (
	ErrNSOPodRestarting = fmt.Errorf("NSO pod is restarting after volume mount")
	ErrNSOPodNotReady   = fmt.Errorf("NSO pod not ready yet")
)

// PackageBundleReconciler reconciles a PackageBundle object
type PackageBundleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *rest.Config
}

// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles/finalizers,verbs=update
// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=nsoes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PackageBundle object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PackageBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	packageBundle := &nsov1alpha1.PackageBundle{}
	err := r.Get(ctx, req.NamespacedName, packageBundle)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("PackageBundle resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PackageBundle")
		return ctrl.Result{}, err
	}

	if packageBundle.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(packageBundle, packageBundleFinalizer) {

			log.Info("PackageBundle is being deleted, cleaning up")

			if err := r.unmountPVCFromNSO(ctx, packageBundle); err != nil {
				log.Error(err, "Failed to unmount PVC during deletion")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(packageBundle, packageBundleFinalizer)
			if err := r.Update(ctx, packageBundle); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Finalizer removed, PackageBundle can now be deleted")
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(packageBundle, packageBundleFinalizer) {
		controllerutil.AddFinalizer(packageBundle, packageBundleFinalizer)
		if err := r.Update(ctx, packageBundle); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Finalizer added to PackageBundle")
		return ctrl.Result{Requeue: true}, nil
	}

	if packageBundle.Status.Phase == "" {
		if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhasePending, "PackageBundle created", ""); err != nil {
			log.Error(err, "Failed to set initial status")
			return ctrl.Result{}, err
		}
	}

	// If already loaded, nothing to do
	if packageBundle.Status.Phase == nsov1alpha1.PackageBundlePhaseLoaded {
		return ctrl.Result{}, nil
	}

	if packageBundle.Status.Phase == nsov1alpha1.PackageBundlePhaseDownloaded ||
		packageBundle.Status.Phase == nsov1alpha1.PackageBundlePhaseLoading {
		if err := r.mountPVCToNSO(ctx, packageBundle); err != nil {
			// Check for sentinel errors that indicate expected retry conditions
			if errors.Is(err, ErrNSOPodRestarting) || errors.Is(err, ErrNSOPodNotReady) {
				log.Info("NSO pod not ready, will retry", "error", err)
				return ctrl.Result{RequeueAfter: requeueIntervalMountRetry}, nil
			}
			log.Error(err, "Failed to mount PVC or reload packages in NSO instance")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create PVC
	pvc := r.newPersistentVolumeClaim(ctx, packageBundle)
	requeue, err := ensureObjectExists(ctx, r.Client, pvc)
	if err != nil {
		if updateErr := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseFailedToDownload, fmt.Sprintf("Failed to create PVC: %v", err), ""); updateErr != nil {
			log.Error(updateErr, "Failed to update status after PVC creation failure")
		}
		return ctrl.Result{}, err
	}
	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// Create Job
	job := r.newJob(ctx, packageBundle)
	jobName := job.Name
	requeue, err = ensureObjectExists(ctx, r.Client, job)
	if err != nil {
		if updateErr := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseFailedToDownload, fmt.Sprintf("Failed to create Job: %v", err), jobName); updateErr != nil {
			log.Error(updateErr, "Failed to update status after Job creation failure")
		}
		return ctrl.Result{}, err
	}
	if requeue {
		// Update status to indicate job is being created
		if updateErr := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseContainerCreating, "Job created, waiting for containers", jobName); updateErr != nil {
			log.Error(updateErr, "Failed to update status after Job creation")
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check Job status and update PackageBundle phase accordingly
	phase, message, err := getJobStatus(ctx, r.Client, jobName, packageBundle.Namespace)
	if err != nil {
		log.Error(err, "Failed to get Job status", "job", jobName)
		return ctrl.Result{RequeueAfter: requeueIntervalJobStatus}, err
	}

	// Update PackageBundle status based on Job status
	if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, phase, message, jobName); err != nil {
		log.Error(err, "Failed to update PackageBundle status", "phase", phase)
		return ctrl.Result{}, err
	}

	// Requeue if the job is still running or pending
	if phase == nsov1alpha1.PackageBundlePhaseContainerCreating || phase == nsov1alpha1.PackageBundlePhaseDownloading {
		return ctrl.Result{RequeueAfter: requeueIntervalJobStatus}, nil
	}

	// If job succeeded, the phase will be Downloaded and on next reconcile the PVC will be mounted
	return ctrl.Result{}, nil
}

func (r *PackageBundleReconciler) unmountPVCFromNSO(ctx context.Context, packageBundle *nsov1alpha1.PackageBundle) error {
	log := logf.FromContext(ctx)

	nso := &nsov1alpha1.NSO{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      packageBundle.Spec.TargetName,
		Namespace: packageBundle.Namespace,
	}, nso)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("NSO instance not found, nothing to unmount", "targetName", packageBundle.Spec.TargetName)
			return nil
		}
		return err
	}

	volumeName := generatePackageVolumeName(packageBundle.Name)

	volumeExists := false
	for _, vol := range nso.Spec.Volumes {
		if vol.Name == volumeName {
			volumeExists = true
			break
		}
	}

	if !volumeExists {
		log.Info("Volume not mounted, nothing to unmount", "volume", volumeName)
		return nil
	}

	log.Info("Removing PVC volume from NSO instance", "nso", nso.Name, "volume", volumeName)

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latestNSO := &nsov1alpha1.NSO{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      packageBundle.Spec.TargetName,
			Namespace: packageBundle.Namespace,
		}, latestNSO); err != nil {
			return err
		}

		newVolumes := []corev1.Volume{}
		for _, vol := range latestNSO.Spec.Volumes {
			if vol.Name != volumeName {
				newVolumes = append(newVolumes, vol)
			}
		}
		latestNSO.Spec.Volumes = newVolumes

		newVolumeMounts := []corev1.VolumeMount{}
		for _, vm := range latestNSO.Spec.VolumeMounts {
			if vm.Name != volumeName {
				newVolumeMounts = append(newVolumeMounts, vm)
			}
		}
		latestNSO.Spec.VolumeMounts = newVolumeMounts

		// Update the NSO instance
		return r.Update(ctx, latestNSO)
	})

	if err != nil {
		log.Error(err, "Failed to update NSO instance to remove PVC mount")
		return err
	}

	log.Info("Successfully unmounted PVC from NSO instance", "nso", nso.Name)
	return nil
}

// mountPVCToNSO mounts the PVC to the NSO instance specified in targetName
func (r *PackageBundleReconciler) mountPVCToNSO(ctx context.Context, packageBundle *nsov1alpha1.PackageBundle) error {
	log := logf.FromContext(ctx)

	// Get the NSO instance
	nso := &nsov1alpha1.NSO{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      packageBundle.Spec.TargetName,
		Namespace: packageBundle.Namespace,
	}, nso)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "NSO instance not found", "targetName", packageBundle.Spec.TargetName)
			return fmt.Errorf("NSO instance %s not found: %w", packageBundle.Spec.TargetName, err)
		}
		return err
	}

	// Prepare the PVC name and volume name
	pvcName := generatePVCName(packageBundle.Name, packageBundle.Spec.TargetName)
	volumeName := generatePackageVolumeName(packageBundle.Name)
	mountPath := nsoPackagesMountPath
	packagesFolder := normalizePathString(packageBundle.Spec.Source.Path)

	// Check if volume is already mounted
	volumeExists := false
	for _, vol := range nso.Spec.Volumes {
		if vol.Name == volumeName {
			volumeExists = true
			break
		}
	}

	// If volume doesn't exist, add it
	if !volumeExists {
		log.Info("Adding PVC volume to NSO instance", "nso", nso.Name, "pvc", pvcName)

		// Prepare the volume and volume mount to add
		newVolume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		}

		newVolumeMount := corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			SubPath:   packagesFolder,
		}

		// Update the NSO instance with retry logic - THIS TRIGGERS POD RESTART
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Re-fetch the latest NSO to avoid conflicts
			latestNSO := &nsov1alpha1.NSO{}
			if err := r.Get(ctx, client.ObjectKey{
				Name:      packageBundle.Spec.TargetName,
				Namespace: packageBundle.Namespace,
			}, latestNSO); err != nil {
				return err
			}

			// Check again if volume already exists (it might have been added by another reconcile)
			for _, vol := range latestNSO.Spec.Volumes {
				if vol.Name == volumeName {
					log.Info("Volume already exists in latest NSO, skipping update")
					return nil
				}
			}

			// Add the volume and volume mount
			latestNSO.Spec.Volumes = append(latestNSO.Spec.Volumes, newVolume)
			latestNSO.Spec.VolumeMounts = append(latestNSO.Spec.VolumeMounts, newVolumeMount)

			// Update the NSO instance
			return r.Update(ctx, latestNSO)
		})

		if err != nil {
			log.Error(err, "Failed to update NSO instance with PVC mount after retries")
			return err
		}

		log.Info("Successfully mounted PVC to NSO instance - pod will restart", "nso", nso.Name, "pvc", pvcName)

		// Update status to Loading - volume is mounted, now wait for pod restart
		if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseLoading,
			fmt.Sprintf("Volume mounted to NSO instance %s, waiting for pod restart", nso.Name),
			packageBundle.Status.JobName); err != nil {
			log.Error(err, "Failed to update PackageBundle status to Loading")
			return err
		}

		// Return sentinel error to trigger requeue
		return ErrNSOPodRestarting
	}

	// Volume already exists - pod should be ready, proceed to reload packages
	log.Info("Volume already mounted, proceeding to reload packages", "nso", nso.Name)

	// Find the ready NSO pod
	podName, err := findReadyPodByLabels(ctx, r.Client, packageBundle.Namespace, nso.Spec.LabelSelector)
	if err != nil {
		log.Error(err, "Failed to find NSO pod", "labels", nso.Spec.LabelSelector)
		return err
	}

	if podName == "" {
		log.Info("NSO pod not ready yet, will retry", "labels", nso.Spec.LabelSelector)
		// Update status to Loading if not already
		if packageBundle.Status.Phase != nsov1alpha1.PackageBundlePhaseLoading {
			if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseLoading,
				"Waiting for NSO pod to be ready",
				packageBundle.Status.JobName); err != nil {
				log.Error(err, "Failed to update PackageBundle status to Loading")
			}
		}
		return ErrNSOPodNotReady
	}

	// Execute packages reload command
	log.Info("Executing package reload command", "pod", podName, "namespace", packageBundle.Namespace)

	executor, err := newPodExecutor(r.Config)
	if err != nil {
		log.Error(err, "Failed to create pod executor")
		return err
	}

	adminUsername := nso.Spec.AdminCredentials.Username
	command := []string{nsoCliCommand, nsoCliUserFlag, adminUsername}
	stdin := nsoPackageReloadCommand
	containerName := nsoContainerName

	stdout, stderr, err := executor.ExecCommandInPod(
		ctx,
		packageBundle.Namespace,
		podName,
		containerName,
		command,
		stdin,
	)

	// Log output regardless of success/failure
	log.Info("Package reload command output",
		"pod", podName,
		"stdout", stdout,
		"stderr", stderr,
		"error", err)

	if err != nil {
		// Update status to indicate failure
		failureMsg := fmt.Sprintf("Failed to reload packages in NSO pod %s: %v. Stderr: %s", podName, err, stderr)
		if updateErr := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseLoading,
			failureMsg,
			packageBundle.Status.JobName); updateErr != nil {
			log.Error(updateErr, "Failed to update PackageBundle status after command failure")
		}
		return fmt.Errorf("%s", failureMsg)
	}

	// Success! Update status to Loaded
	if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhaseLoaded,
		fmt.Sprintf("Package successfully loaded to NSO instance %s. Output: %s", nso.Name, stdout),
		packageBundle.Status.JobName); err != nil {
		log.Error(err, "Failed to update PackageBundle status to Loaded")
		return err
	}

	log.Info("Successfully reloaded packages in NSO", "pod", podName)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PackageBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nsov1alpha1.PackageBundle{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("packagebundle").
		Complete(r)
}
