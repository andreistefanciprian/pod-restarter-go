package kubernetes

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Logger interface {
	Print(v ...any)
	Println(v ...any)
	Printf(format string, v ...any)
}

// podRestarter holds K8s parameters
type PodRestarter struct {
	Logger     Logger
	Kubeconfig *string
	Ctx        context.Context
	Clientset  *kubernetes.Clientset
}

// podDetails holds data associated with a Pod
type PodDetails struct {
	UID               types.UID
	PodName           string
	PodNamespace      string
	ResourceVersion   string
	HasOwner          bool
	OwnerData         interface{}
	Phase             v1.PodPhase
	CreationTimestamp time.Time
	DeletionTimestamp *metav1.Time
}

// podEvent holds events data associated with a Pod
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
