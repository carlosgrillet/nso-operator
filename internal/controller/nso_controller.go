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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	orchestrationciscocomv1alpha1 "wwwin-github.cisco.com/cgrillet/nso-operator/api/v1alpha1"
)

// NSOReconciler reconciles a NSO object
type NSOReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=orchestration.cisco.com.cisco.com,resources=nsos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orchestration.cisco.com.cisco.com,resources=nsos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=orchestration.cisco.com.cisco.com,resources=nsos/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NSO object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *NSOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the NSO instance
	nso := &orchestrationciscocomv1alpha1.NSO{}
	err := r.Get(ctx, req.NamespacedName, nso)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("NSO resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NSO")
		return ctrl.Result{}, err
	}

	// Objects to create - Service must be created first for StatefulSet

	service := r.serviceForNSO(nso, ctx)
	requeue, err := r.ensureObjectExists(ctx, service)
	if err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, nil
	}

	statefulSet := r.statefulSetForNSO(nso, ctx)
	requeue, err = r.ensureObjectExists(ctx, statefulSet)
	if err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, nil
	}

	return ctrl.Result{}, nil
}

// Function to safely verify if the resource is created or not before reconcile
func (r *NSOReconciler) ensureObjectExists(ctx context.Context, obj client.Object) (bool, error) {
	log := logf.FromContext(ctx)

	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new resource", "kind:", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
		err = r.Create(ctx, obj)
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

func (r *NSOReconciler) statefulSetForNSO(nso *orchestrationciscocomv1alpha1.NSO, ctx context.Context) *appsv1.StatefulSet {
	log := logf.FromContext(ctx)
	statefulSetName := nso.Name
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: nso.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: nso.Spec.ServiceName,
			Replicas:    &nso.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: nso.Spec.LabelSelector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: nso.Spec.LabelSelector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "ncs",
						Image: nso.Spec.Image,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "http",
						}, {
							ContainerPort: 8888,
							Name:          "https",
						}},
						Env: append([]corev1.EnvVar{{
							Name:  "ADMIN_USERNAME",
							Value: nso.Spec.AdminCredentials.Username,
						}, {
							Name: "ADMIN_PASSWORD",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: nso.Spec.AdminCredentials.PasswordSecretRef,
									},
									Key: "password",
								},
							},
						}}, nso.Spec.Env...),
					}},
				},
			},
		},
	}
	err := controllerutil.SetControllerReference(nso, statefulSet, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for StatefulSet")
		return &appsv1.StatefulSet{}
	}
	return statefulSet
}

func (r *NSOReconciler) serviceForNSO(nso *orchestrationciscocomv1alpha1.NSO, ctx context.Context) *corev1.Service {
	log := logf.FromContext(ctx)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nso.Spec.ServiceName,
			Namespace: nso.Namespace,
			Labels:    nso.Spec.LabelSelector,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			Selector:  nso.Spec.LabelSelector,
			Ports:     nso.Spec.Ports,
			ClusterIP: corev1.ClusterIPNone,
		},
	}
	err := controllerutil.SetControllerReference(nso, service, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for Service")
		return &corev1.Service{}
	}
	return service
}

// SetupWithManager sets up the controller with the Manager.
func (r *NSOReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&orchestrationciscocomv1alpha1.NSO{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Named("nso").
		Complete(r)
}
