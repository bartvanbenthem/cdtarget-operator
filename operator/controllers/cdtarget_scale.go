package controllers

import (
	cnadv1alpha1 "github.com/bartvanbenthem/cdtarget-operator/api/v1alpha1"
	kedav2 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *CDTargetReconciler) scaledObjectForCDTarget(t *cnadv1alpha1.CDTarget) *kedav2.ScaledObject {
	name := "cdtarget-config"

	triggerMeta := map[string]string{
		"poolName":               t.Spec.Config.PoolName,
		"organizationURLFromEnv": "AZP_URL",
	}

	so := &kedav2.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace,
			Labels:    t.Spec.AdditionalSelector,
		},
		Spec: kedav2.ScaledObjectSpec{
			ScaleTargetRef: &kedav2.ScaleTarget{
				Name: t.Name,
			},
			MinReplicaCount: t.Spec.MinReplicaCount,
			MaxReplicaCount: t.Spec.MaxReplicaCount,
			Triggers: []kedav2.ScaleTriggers{{
				Type: "azure-pipelines",
				AuthenticationRef: &kedav2.ScaledObjectAuthRef{
					Name: "pipeline-trigger-auth",
				},
				Metadata: triggerMeta,
			}},
		},
	}

	return so
}

func (r *CDTargetReconciler) triggerAuthenticationForCDTarget(t *cnadv1alpha1.CDTarget) *kedav2.TriggerAuthentication {

	name := "pipeline-trigger-auth"

	ta := &kedav2.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace,
			Labels:    t.Spec.AdditionalSelector,
		},
		Spec: kedav2.TriggerAuthenticationSpec{
			SecretTargetRef: []kedav2.AuthSecretTargetRef{{
				Parameter: "personalAccessToken",
				Name:      t.Spec.TokenRef,
				Key:       "AZP_TOKEN",
			}},
		},
	}

	return ta
}
