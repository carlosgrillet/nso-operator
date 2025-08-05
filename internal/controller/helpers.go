package controller

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nsov1alpha1 "wwwin-github.cisco.com/cgrillet/nso-operator/api/v1alpha1"
)

// Function to safely verify if the resource is created or not before reconcile
func ensureObjectExists(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	log := logf.FromContext(ctx)

	err := c.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new resource")
		err = c.Create(ctx, obj)
		if err != nil {
			log.Error(err, "Failed to create new resource")
			return false, err
		}
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get resource")
		return false, err
	}

	log.V(1).Info("Skip reconcile: resource already exists")
	return false, nil
}

// Maps ConfigMap and Secrets changes to NSO reconcile requests
func (r *NSOReconciler) watchForResourceChange(ctx context.Context, resource client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)
	attachedNSOList := &nsov1alpha1.NSOList{}
	resourceName := resource.GetName()
	resourceKind := resource.GetObjectKind().GroupVersionKind().Kind

	// List all NSO resources in the same namespace
	listOptions := &client.ListOptions{
		Namespace: resource.GetNamespace(),
	}

	// Verify if there are NSO instances in the namespace
	err := r.List(ctx, attachedNSOList, listOptions)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0)
	for _, nso := range attachedNSOList.Items {

		nsoConfigMapName := nso.Spec.NsoConfigRef
		nsoSecretName := nso.Spec.AdminCredentials.PasswordSecretRef

		shouldReconcile := (resourceKind == "Secret" && nsoSecretName == resourceName) ||
			(resourceKind == "ConfigMap" && nsoConfigMapName == resourceName)

		if shouldReconcile {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      nso.GetName(),
					Namespace: nso.GetNamespace(),
				},
			})

			log.Info("Resource change detected. Reconciling NSO",
				"nsoInstace", nso.GetName(),
				"resourceChanged", "kind", resourceKind, "name", resourceName)
		}
	}

	return requests
}

// Updates the PackageBundle status phase based on the Job status
func updatePackageBundlePhase(ctx context.Context, c client.Client, packageBundle *nsov1alpha1.PackageBundle, newPhase nsov1alpha1.PackageBundlePhase, message string, jobName string) error {
	log := logf.FromContext(ctx)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the PackageBundle to avoid conflicts
		latest := &nsov1alpha1.PackageBundle{}
		if err := c.Get(ctx, types.NamespacedName{Name: packageBundle.Name, Namespace: packageBundle.Namespace}, latest); err != nil {
			return err
		}

		// Only update if phase has changed
		if latest.Status.Phase == newPhase {
			return nil
		}

		// Update status fields
		now := metav1.NewTime(time.Now())
		latest.Status.Phase = newPhase
		latest.Status.Message = message
		latest.Status.JobName = jobName
		latest.Status.LastTransitionTime = &now

		// Update the status subresource
		if err := c.Status().Update(ctx, latest); err != nil {
			log.Error(err, "Failed to update PackageBundle status", "phase", newPhase, "message", message)
			return err
		}

		log.Info("Updated PackageBundle phase", "name", packageBundle.Name, "phase", newPhase, "message", message)
		return nil
	})
}

// Checks the Job status and returns the corresponding PackageBundle phase and message
func getJobStatus(ctx context.Context, c client.Client, jobName, namespace string) (nsov1alpha1.PackageBundlePhase, string, error) {
	log := logf.FromContext(ctx)

	job := &batchv1.Job{}
	err := c.Get(ctx, types.NamespacedName{Name: jobName, Namespace: namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			return nsov1alpha1.PackageBundlePhasePending, "Job not found", nil
		}
		return nsov1alpha1.PackageBundlePhasePending, "Failed to get Job status", err
	}

	// Check Job conditions
	for _, condition := range job.Status.Conditions {
		switch condition.Type {
		case batchv1.JobComplete:
			if condition.Status == "True" {
				log.Info("Job completed successfully", "job", jobName)
				return nsov1alpha1.PackageBundlePhaseDownloaded, "Package download completed successfully", nil
			}
		case batchv1.JobFailed:
			if condition.Status == "True" {
				message := "Package download failed"
				if condition.Message != "" {
					message = condition.Message
				}
				log.Info("Job failed", "job", jobName, "message", message)
				return nsov1alpha1.PackageBundlePhaseFailedToDownload, message, nil
			}
		}
	}

	// Check if Job is actively running
	if job.Status.Active > 0 {
		log.Info("Job is running", "job", jobName, "activePods", job.Status.Active)
		return nsov1alpha1.PackageBundlePhaseDownloading, "Package download in progress", nil
	}

	// Job exists but no active pods yet - could be container creating
	if job.Status.Active == 0 && job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		log.Info("Job is pending", "job", jobName)
		return nsov1alpha1.PackageBundlePhaseContainerCreating, "Job is creating containers", nil
	}

	// Default case
	return nsov1alpha1.PackageBundlePhasePending, "Job status unknown", nil
}
