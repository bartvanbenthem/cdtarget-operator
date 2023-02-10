package controllers

import (
	cnadv1alpha1 "github.com/bartvanbenthem/cdtarget-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *CDTargetReconciler) configMapForCDTarget(t *cnadv1alpha1.CDTarget) *corev1.ConfigMap {

	name := "cdtarget-config"

	data := map[string]string{}
	data["AZP_POOL"] = string(t.Spec.Config.PoolName)
	data["AZP_URL"] = string(t.Spec.Config.URL)
	data["AZP_WORK"] = string(t.Spec.Config.WorkDir)
	data["AZP_AGENT_NAME"] = string(t.Spec.Config.AgentName)
	data["AGENT_MTU_VALUE"] = string(t.Spec.Config.MTUValue)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace,
			Labels:    t.Spec.AdditionalSelector,
		},
		Data: data,
	}

	return cm
}

func (r *CDTargetReconciler) tokenSecretForCDTarget(t *cnadv1alpha1.CDTarget) *corev1.Secret {

	name := t.Spec.TokenRef

	secdata := map[string]string{}
	secdata["AZP_TOKEN"] = string("")

	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    t.Spec.AdditionalSelector,
			Name:      name,
			Namespace: t.Namespace,
		},
		StringData: secdata,
	}

	return &sec
}

func boolPointer(b bool) *bool {
	return &b
}

func (r *CDTargetReconciler) deploymentForCDTarget(t *cnadv1alpha1.CDTarget) *appsv1.Deployment {

	cmd := []string{"/bin/sh", "-c", "update-ca-certificates"}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: t.Namespace,
			Labels:    t.Spec.AdditionalSelector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: t.Spec.MinReplicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: t.Spec.AdditionalSelector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: t.Spec.AdditionalSelector,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: t.Spec.CACertRef,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: t.Spec.CACertRef,
								Optional:   boolPointer(true),
							},
						},
					}},
					ImagePullSecrets: t.Spec.ImagePullSecrets,
					Containers: []corev1.Container{{
						Image: t.Spec.AgentImage,
						Name:  "agent",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      t.Spec.CACertRef,
							MountPath: "/usr/local/share/ca-certificates",
							ReadOnly:  true,
						}},
						Lifecycle: &corev1.Lifecycle{
							PostStart: &corev1.LifecycleHandler{
								Exec: &corev1.ExecAction{
									Command: cmd,
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: t.Spec.AgentResources.Requests,
							Limits:   t.Spec.AgentResources.Limits,
						},
						Env: []corev1.EnvVar{
							{
								Name: "AZP_URL",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Key: "AZP_URL",
									},
								},
							},
							{
								Name: "AZP_POOL",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Key: "AZP_POOL",
									},
								},
							},
							{
								Name: "AZP_WORK",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Optional: boolPointer(true),
										Key:      "AZP_WORK",
									},
								},
							},
							{
								Name: "AZP_AGENT_NAME",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Optional: boolPointer(true),
										Key:      "AZP_AGENT_NAME",
									},
								},
							},
							{
								Name: "AGENT_MTU_VALUE",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Optional: boolPointer(true),
										Key:      "AGENT_MTU_VALUE",
									},
								},
							},
							{
								Name: "AZP_TOKEN",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.TokenRef},
										Key: "AZP_TOKEN",
									},
								},
							},
							{
								Name: "HTTP_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "HTTP_PROXY",
									},
								},
							},
							{
								Name: "HTTPS_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "HTTPS_PROXY",
									},
								},
							},
							{
								Name: "PROXY_USER",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "PROXY_USER",
									},
								},
							},
							{
								Name: "PROXY_PW",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "PROXY_PW",
									},
								},
							},
							{
								Name: "PROXY_URL",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "PROXY_URL",
									},
								},
							},
							{
								Name: "FTP_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "FTP_PROXY",
									},
								},
							},
							{
								Name: "NO_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Optional: boolPointer(true),
										Key:      "NO_PROXY",
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	if dep.Spec.Template.Spec.Containers[0].Name == "agent" {
		for _, env := range t.Spec.Env {
			dep.Spec.Template.Spec.Containers[0].Env =
				append(dep.Spec.Template.Spec.Containers[0].Env, env)
		}
	}

	return dep
}
