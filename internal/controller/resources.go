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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nsov1alpha1 "github.com/carlosgrillet/nso-operator/api/v1alpha1"
)

const (
	sshKeyFileMode  = int32(0400)
	sshVolumeName   = "ssh-key"
	sshKeyMountPath = "/.ssh"
)

func (r *NSOReconciler) newCDBPersistentVolumeClaim(ctx context.Context, nso *nsov1alpha1.NSO) *corev1.PersistentVolumeClaim {
	log := logf.FromContext(ctx)
	defaultSize := "5Gi"
	if nso.Spec.CDBStorageSize != "" {
		defaultSize = nso.Spec.CDBStorageSize
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cdb", nso.Name),
			Namespace: nso.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(defaultSize),
				},
			},
		},
	}
	err := controllerutil.SetControllerReference(nso, pvc, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for CDB PVC")
		return &corev1.PersistentVolumeClaim{}
	}
	return pvc
}

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
						}, {
							Name:      "cdb-storage",
							MountPath: "/nso/run/cdb",
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
					}, {
						Name: "cdb-storage",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: fmt.Sprintf("%s-cdb", nso.Name),
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
func (r *PackageBundleReconciler) newPersistentVolumeClaim(ctx context.Context, pb *nsov1alpha1.PackageBundle) *corev1.PersistentVolumeClaim {
	log := logf.FromContext(ctx)
	pvcName := generatePVCName(pb.Name, pb.Spec.TargetName)
	size := resource.MustParse(defaultStorageSize)
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
	pvcName := generatePVCName(pb.Name, pb.Spec.TargetName)
	jobName := generateDownloadJobName(pb.Name)
	volumeName := jobVolumeName
	volumeMountPath := jobVolumeMountPath
	packagesPath := normalizePathString(pb.Spec.Source.Path)
	var ttlSecondsAfterFinished int32 = 1800
	var backoffLimit int32 = 3

	// Determine if we're using SSH (git@ URL) vs HTTPS
	isSSH := strings.HasPrefix(pb.Spec.Source.Url, "git@")

	securityContext := &corev1.SecurityContext{
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To(int64(1000)),
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	if isSSH {
		securityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		}
	}

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	gitCloneCmd := []string{"git", "clone", "--depth", "1"}
	if pb.Spec.Source.Branch != "" {
		gitCloneCmd = append(gitCloneCmd, "--branch", pb.Spec.Source.Branch)
	}
	gitCloneCmd = append(gitCloneCmd, pb.Spec.Source.Url, volumeMountPath)

	initVolumeMounts := []corev1.VolumeMount{{
		Name:      volumeName,
		MountPath: volumeMountPath,
	}}

	volumes := []corev1.Volume{{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}}

	var initEnv []corev1.EnvVar
	var podSecurityContext *corev1.PodSecurityContext

	// Only mount SSH key if using SSH protocol (git@)
	if isSSH && pb.Spec.Credentials.SshKeySecretRef != "" {
		volumes = append(volumes, corev1.Volume{
			Name: sshVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  pb.Spec.Credentials.SshKeySecretRef,
					DefaultMode: ptr.To(sshKeyFileMode),
					Items: []corev1.KeyToPath{{
						Key:  "id_rsa",
						Path: "id_rsa",
						Mode: ptr.To(sshKeyFileMode),
					}},
				},
			},
		})
		initVolumeMounts = append(initVolumeMounts, corev1.VolumeMount{
			Name:      sshVolumeName,
			MountPath: sshKeyMountPath,
			ReadOnly:  true,
		})

		initEnv = []corev1.EnvVar{
			{
				Name:  "HOME",
				Value: "/tmp",
			},
			{
				Name:  "GIT_SSH_COMMAND",
				Value: fmt.Sprintf("ssh -i %s/id_rsa -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyMountPath),
			},
		}

		podSecurityContext = nil
	}

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
					RestartPolicy:   corev1.RestartPolicyNever,
					SecurityContext: podSecurityContext,
					InitContainers: []corev1.Container{{
						Name:            fmt.Sprintf("%s-downloader", pb.Name),
						Image:           pb.Spec.Config.Download.Image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         gitCloneCmd,
						Env:             initEnv,
						Resources:       resources,
						SecurityContext: securityContext,
						VolumeMounts:    initVolumeMounts,
					}},
					Containers: []corev1.Container{{
						Name:       fmt.Sprintf("%s-builder", pb.Name),
						Image:      pb.Spec.Config.Build.Image,
						Command:    []string{"/bin/sh", "-c"},
						Args:       []string{fmt.Sprintf("for dir in %s/*/src; do echo \"$dir\"; cd \"$dir\" && make clean all; done", packagesPath)},
						WorkingDir: volumeMountPath,
						Resources:  resources,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      volumeName,
							MountPath: volumeMountPath,
						}},
					}},
					Volumes: volumes,
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
