package controller

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nsov1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

// PodExecutor handles command execution in pods
type PodExecutor struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewPodExecutor creates an executor from the in-cluster config
func newPodExecutor(config *rest.Config) (*PodExecutor, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &PodExecutor{clientset: clientset, config: config}, nil
}

// ExecInPod runs a command in a container and returns stdout/stderr
func (e *PodExecutor) ExecCommandInPod(ctx context.Context, namespace, podName, container string, command []string, stdin string) (string, string, error) {
	podExecOptions := &corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}

	req := e.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(podExecOptions, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(e.config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  strings.NewReader(stdin),
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return stdout.String(), stderr.String(), err
}

// Function to safely verify if the resource is created or not before reconcile
func ensureObjectExists(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	log := logf.FromContext(ctx)

	// Create a copy to check if resource exists
	existing := obj.DeepCopyObject().(client.Object)
	err := c.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)

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

	// Resource exists - Check if it's a Job or PVC (which have immutable specs)
	if _, isJob := obj.(*batchv1.Job); isJob {
		log.V(1).Info("Job already exists, skipping update (Jobs are immutable)")
		return false, nil
	}

	if _, isPVC := obj.(*corev1.PersistentVolumeClaim); isPVC {
		log.V(1).Info("PersistentVolumeClaim already exists, skipping update (PVC specs are immutable)")
		return false, nil
	}

	// Resource exists - update it with the NEW desired state using retry logic
	log.Info("Resource exists, updating to match desired state")
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Re-fetch the latest version to avoid conflicts
		latest := obj.DeepCopyObject().(client.Object)
		if err := c.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, latest); err != nil {
			return err
		}

		// Update the resource version and other metadata from the latest version
		obj.SetResourceVersion(latest.GetResourceVersion())
		obj.SetUID(latest.GetUID())

		// Attempt the update
		return c.Update(ctx, obj)
	})

	if err != nil {
		log.Error(err, "Failed to update resource after retries")
		return false, err
	}

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

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the PackageBundle to avoid conflicts
		latest := &nsov1alpha1.PackageBundle{}
		if err := c.Get(ctx, types.NamespacedName{Name: packageBundle.Name, Namespace: packageBundle.Namespace}, latest); err != nil {
			return err
		}

		// Only update if status has actually changed
		// Check phase, message, and jobName to avoid unnecessary updates
		if latest.Status.Phase == newPhase &&
			latest.Status.Message == message &&
			latest.Status.JobName == jobName {
			log.V(1).Info("Status unchanged, skipping update", "name", packageBundle.Name, "phase", newPhase)
			return nil
		}

		// Check if phase is actually changing (for LastTransitionTime)
		phaseChanged := latest.Status.Phase != newPhase

		// Update status fields
		latest.Status.Phase = newPhase
		latest.Status.Message = message
		latest.Status.JobName = jobName

		if phaseChanged {
			now := metav1.NewTime(time.Now())
			latest.Status.LastTransitionTime = &now
		}

		// Update the status subresource
		return c.Status().Update(ctx, latest)
	})

	if err != nil {
		log.Error(err, "Failed to update PackageBundle status after retries", "phase", newPhase, "message", message)
		return err
	}

	log.Info("Updated PackageBundle phase", "name", packageBundle.Name, "phase", newPhase, "message", message)
	return nil
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

func normalizePathString(path string) string {
	path = strings.TrimSpace(path)
	return strings.Trim(path, "/")
}

// findReadyPodByLabels finds a ready pod matching the given label selector
// Returns the pod name if found and ready, empty string if not found or not ready, and error for API failures
func findReadyPodByLabels(ctx context.Context, c client.Client, namespace string, labels map[string]string) (string, error) {
	log := logf.FromContext(ctx)

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	}

	if err := c.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "namespace", namespace, "labels", labels)
		return "", err
	}

	if len(podList.Items) == 0 {
		log.Info("No pods found matching labels", "namespace", namespace, "labels", labels)
		return "", nil
	}

	// Find first pod that is Running and Ready
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Check if all containers are ready
		allReady := true
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				if condition.Status != corev1.ConditionTrue {
					allReady = false
				}
				break
			}
		}

		if allReady {
			log.Info("Found ready pod", "podName", pod.Name, "namespace", namespace)
			return pod.Name, nil
		}
	}

	log.Info("Found pods but none are ready yet", "namespace", namespace, "labels", labels, "podCount", len(podList.Items))
	return "", nil
}

// generatePVCName creates a consistent PVC name for a PackageBundle
// Format: {bundleName}-{targetName}
func generatePVCName(bundleName, targetName string) string {
	return fmt.Sprintf("%s-%s", bundleName, targetName)
}

// generatePackageVolumeName creates a consistent volume name for a PackageBundle
// Format: package-{bundleName}
func generatePackageVolumeName(bundleName string) string {
	return fmt.Sprintf(packageVolumeNameFmt, bundleName)
}

// generateDownloadJobName creates a consistent job name for a PackageBundle
// Format: download-{bundleName}
func generateDownloadJobName(bundleName string) string {
	return fmt.Sprintf(downloadJobNameFmt, bundleName)
}
