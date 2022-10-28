package assets

import (
	"embed"

	v1 "k8s.io/api/core/v1"
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
	if err := v1.AddToScheme(appsScheme); err != nil {
		panic(err)
	}
}

func GetConfigMapFromFile(name string) *v1.ConfigMap {
	configMapBytes, err := manifests.ReadFile(name)
	if err != nil {
		panic(err)
	}

	configMapObject, err := runtime.Decode(appsCodecs.UniversalDecoder(v1.SchemeGroupVersion), configMapBytes)
	if err != nil {
		panic(err)
	}

	return configMapObject.(*v1.ConfigMap)
}
