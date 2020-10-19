package builder

import (
	corev1api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceBuilder builds Namespace objects.
type NamespaceBuilder struct {
	object *corev1api.Namespace
}

func ForNamespace(name string) *NamespaceBuilder {
	return &NamespaceBuilder{
		object: &corev1api.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1api.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

// Result returns the built Namespace.
func (b *NamespaceBuilder) Result() *corev1api.Namespace {
	return b.object
}

// ObjectMeta applies functional options to the Namespace's ObjectMeta.
func (b *NamespaceBuilder) ObjectMeta(opts ...ObjectMetaOpt) *NamespaceBuilder {
	for _, opt := range opts {
		opt(b.object)
	}

	return b
}

// Phase sets the namespace's phase
func (b *NamespaceBuilder) Phase(val corev1api.NamespacePhase) *NamespaceBuilder {
	b.object.Status.Phase = val
	return b
}

//https://github.com/vmware-tanzu/velero/blob/f42c63af1b9af445e38f78a7256b1c48ef79c10e/pkg/builder/namespace_builder.go

//Golang pattern (Functional Options)
//https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md
