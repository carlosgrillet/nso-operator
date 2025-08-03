package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
		log.Info("Creating a new resource", "kind:", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
		err = c.Create(ctx, obj)
		if err != nil {
			log.Error(err, "Failed to create new resource", "kind:", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
			return false, err
		}
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get resource", "kind:", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
		return false, err
	}

	log.Info("Skip reconcile: resource already exists", "kind:", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
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
	// err := c.List(ctx, attachedNSOList, listOptions)
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
