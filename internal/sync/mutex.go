package sync

import "sync"

var (
	// ConfigurationMutex is used to handle concurrent access to common ArgoCD
	// configuration stored in the `argocd-cm` ConfigMap resource.
	ConfigurationMutex = &sync.RWMutex{}

	// GPGKeysMutex is used to handle concurrent access to ArgoCD GPG keys which are
	// stored in the `argocd-gpg-keys-cm` ConfigMap resource
	GPGKeysMutex = &sync.RWMutex{}

	// SecretsMutex is used to handle concurrent access to ArgoCD secrets which
	// are stored in the `argocd-secret` Secret resource
	SecretsMutex = &sync.RWMutex{}
)
