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
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nsov1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

// PackageBundleReconciler reconciles a PackageBundle object
type PackageBundleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=orchestration.cisco.com,resources=packagebundles/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

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

	// Fetch the PackageBundle instance
	packageBundle := &nsov1alpha1.PackageBundle{}
	err := r.Get(ctx, req.NamespacedName, packageBundle)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("PackageBundle resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PackageBundle")
		return ctrl.Result{}, err
	}

	// Set initial status if not set
	if packageBundle.Status.Phase == "" {
		if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, nsov1alpha1.PackageBundlePhasePending, "PackageBundle created", ""); err != nil {
			log.Error(err, "Failed to set initial status")
			return ctrl.Result{}, err
		}
	}

	// If already downloaded, skip job creation
	if packageBundle.Status.Phase == nsov1alpha1.PackageBundlePhaseDownloaded {
		return ctrl.Result{}, nil
	}

	// Create PVC
	pvc := r.newPersistenVolumeClaim(ctx, packageBundle)
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
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Update PackageBundle status based on Job status
	if err := updatePackageBundlePhase(ctx, r.Client, packageBundle, phase, message, jobName); err != nil {
		log.Error(err, "Failed to update PackageBundle status", "phase", phase)
		return ctrl.Result{}, err
	}

	// Requeue if the job is still running or pending
	if phase == nsov1alpha1.PackageBundlePhaseContainerCreating || phase == nsov1alpha1.PackageBundlePhaseDownloading {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
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
