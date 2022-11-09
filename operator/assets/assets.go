package assets

import (
	"embed"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	//go:embed manifests/*
	manifests embed.FS

	appsScheme = runtime.NewScheme()
	appsCodecs = serializer.NewCodecFactory(appsScheme)
)

func init() {
	if err := corev1.AddToScheme(appsScheme); err != nil {
		panic(err)
	}

	if err := netv1.AddToScheme(appsScheme); err != nil {
		panic(err)
	}
}

func GetConfigMapFromFile(name string) *corev1.ConfigMap {
	configMapBytes, err := manifests.ReadFile(name)
	if err != nil {
		panic(err)
	}

	configMapObject, err := runtime.Decode(appsCodecs.UniversalDecoder(corev1.SchemeGroupVersion), configMapBytes)
	if err != nil {
		panic(err)
	}

	return configMapObject.(*corev1.ConfigMap)
}

func GetNetworkPolicyFromFile(name string) *netv1.NetworkPolicy {
	networkPolicyBytes, err := manifests.ReadFile(name)
	if err != nil {
		panic(err)
	}

	networkPolicyObject, err := runtime.Decode(appsCodecs.UniversalDecoder(netv1.SchemeGroupVersion), networkPolicyBytes)
	if err != nil {
		panic(err)
	}

	return networkPolicyObject.(*netv1.NetworkPolicy)
}
