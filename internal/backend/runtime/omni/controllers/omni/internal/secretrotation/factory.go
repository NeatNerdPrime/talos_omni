// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package secretrotation

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesClient interface {
	Clientset() *kubernetes.Clientset
	Close()
}

type KubernetesClientFactory interface {
	NewClient(config *rest.Config) (KubernetesClient, error)
}
