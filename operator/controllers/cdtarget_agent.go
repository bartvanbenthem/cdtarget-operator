package controllers

import (
	cnadv1alpha1 "github.com/bartvanbenthem/cdtarget-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *CDTargetReconciler) configMapForCDTarget(t *cnadv1alpha1.CDTarget) *v1.ConfigMap {
	ls := t.Spec.PodSelector
	name := "cdtarget-config"

	data := map[string]string{}
	data["AZP_POOL"] = string(t.Spec.Config.PoolName)
	data["AZP_URL"] = string(t.Spec.Config.URL)
	data["AZP_WORK"] = string(t.Spec.Config.WorkDir)
	data["AZP_AGENT_NAME"] = string(t.Spec.Config.AgentName)
	data["AGENT_MTU_VALUE"] = string(t.Spec.Config.MTUValue)

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace,
			Labels:    ls,
		},
		Data: data,
	}

	return cm
}

func (r *CDTargetReconciler) proxySecretForCDTarget(t *cnadv1alpha1.CDTarget) *corev1.Secret {
	ls := t.Spec.PodSelector
	name := t.Spec.ProxyRef

	secdata := map[string]string{}
	secdata["HTTP_PROXY"] = string("")
	secdata["HTTPS_PROXY"] = string("")
	secdata["FTP_PROXY"] = string("")
	secdata["PROXY_URL"] = string("")
	secdata["PROXY_USER"] = string("")
	secdata["PROXY_PW"] = string("")
	secdata["NO_PROXY"] = string("")

	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    ls,
			Name:      name,
			Namespace: t.Namespace,
		},
		StringData: secdata,
	}

	return &sec
}

func (r *CDTargetReconciler) tokenSecretForCDTarget(t *cnadv1alpha1.CDTarget) *corev1.Secret {
	ls := t.Spec.PodSelector
	name := t.Spec.TokenRef

	secdata := map[string]string{}
	secdata["AZP_TOKEN"] = string("")

	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    ls,
			Name:      name,
			Namespace: t.Namespace,
		},
		StringData: secdata,
	}

	return &sec
}

func (r *CDTargetReconciler) deploymentForCDTarget(t *cnadv1alpha1.CDTarget) *appsv1.Deployment {
	ls := t.Spec.PodSelector
	replicas := t.Spec.MinReplicaCount

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: t.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: t.Spec.AgentImage,
						Name:  "agent",
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
										Key: "AZP_WORK",
									},
								},
							},
							{
								Name: "AZP_AGENT_NAME",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Key: "AZP_AGENT_NAME",
									},
								},
							},
							{
								Name: "AGENT_MTU_VALUE",
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "cdtarget-config"},
										Key: "AGENT_MTU_VALUE",
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
										Key: "HTTP_PROXY",
									},
								},
							},
							{
								Name: "HTTPS_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "HTTPS_PROXY",
									},
								},
							},
							{
								Name: "PROXY_USER",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "PROXY_USER",
									},
								},
							},
							{
								Name: "PROXY_PW",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "PROXY_PW",
									},
								},
							},
							{
								Name: "PROXY_URL",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "PROXY_URL",
									},
								},
							},
							{
								Name: "FTP_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "FTP_PROXY",
									},
								},
							},
							{
								Name: "NO_PROXY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: t.Spec.ProxyRef},
										Key: "NO_PROXY",
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	return dep
}
