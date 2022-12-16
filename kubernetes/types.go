package kubernetes

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// kubeClient holds K8s parameters
type kubeClient struct {
	clientSet *kubernetes.Clientset
}

// PodDetails holds data associated with a Pod
type PodDetails struct {
	UID               types.UID
	PodName           string
	PodNamespace      string
	ResourceVersion   string
	OwnerReferences   []metav1.OwnerReference
	Phase             v1.PodPhase
	ContainerStatuses []v1.ContainerStatus
	CreationTimestamp time.Time
	DeletionTimestamp *metav1.Time
}

// PodEvent holds events data associated with a Pod
type PodEvent struct {
	UID             types.UID
	PodName         string
	PodNamespace    string
	ResourceVersion string
	EventType       string
	Reason          string
	Message         string
	FirstTimestamp  time.Time
	LastTimestamp   time.Time
}
