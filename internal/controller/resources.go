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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nsov1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

// Create a new Headless Service for NSO StatefulSet
func (r *NSOReconciler) newService(ctx context.Context, nso *nsov1alpha1.NSO) *corev1.Service {
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

// Create a new StatefulSet for NSO
func (r *NSOReconciler) newStatefulSet(ctx context.Context, nso *nsov1alpha1.NSO) *appsv1.StatefulSet {
	log := logf.FromContext(ctx)
	statefulSetName := nso.Name
	ncsConfigFileMode := int32(0600)
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
						VolumeMounts: append([]corev1.VolumeMount{{
							Name:      "ncs-config",
							MountPath: "/etc/ncs/ncs.conf",
							SubPath:   "ncs.conf",
						}}, nso.Spec.VolumeMounts...),
					}},
					Volumes: append([]corev1.Volume{{
						Name: "ncs-config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ncs-config",
								},
								Items: []corev1.KeyToPath{{
									Key:  "ncs.conf",
									Path: "ncs.conf",
									Mode: &ncsConfigFileMode,
								}},
							},
						},
					}}, nso.Spec.Volumes...),
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

// Create a new PVC to store the downloaded packages
func (r *PackageBundleReconciler) newPersistenVolumeClaim(ctx context.Context, pb *nsov1alpha1.PackageBundle) *corev1.PersistentVolumeClaim {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-%s", pb.Name, pb.Spec.TargetName)
	size := resource.MustParse("1Gi")
	if pb.Spec.StorageSize != "" {
		size = resource.MustParse(pb.Spec.StorageSize)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pb.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(pb, pvc, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for PVC")
		return &corev1.PersistentVolumeClaim{}
	}
	return pvc
}

// Create a new Job to download the NSO packages
func (r *PackageBundleReconciler) newJob(ctx context.Context, pb *nsov1alpha1.PackageBundle) *batchv1.Job {
	log := logf.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-%s", pb.Name, pb.Spec.TargetName)
	jobName := fmt.Sprintf("download-%s", pb.Name)
	volumeName := "package-storage"
	var ttlSecondsAfterFinished int32 = 300
	var backoffLimit int32 = 3
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pb.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			BackoffLimit:            &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "downloader",
						Image:   "alpine/git",
						Command: []string{"/bin/sh"},
						Args: []string{
							"-c",
							fmt.Sprintf("cd /packages && git clone %s", pb.Spec.Source.Url),
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      volumeName,
							MountPath: "/packages",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: volumeName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					}},
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(pb, job, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for Job")
		return &batchv1.Job{}
	}
	return job
}
