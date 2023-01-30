package kubernetes

import (
	"fmt"
	"math/rand"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func makePod(name, namespace string, rv int, phase v1.PodPhase, UID types.UID) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:               UID,
			Name:              name,
			Namespace:         namespace,
			ResourceVersion:   fmt.Sprintf("%d", rv),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			DeletionTimestamp: nil,
		},
		Status: v1.PodStatus{
			Phase: phase, // v1.PodRunning v1.PodPending v1.PodFailing
		},
	}
}

func makeEvent(name, namespace, eventReason, eventMessage, eventType string,
	rv int, UID types.UID) *v1.Event {
	eventTime := metav1.Now()
	rand.Seed(time.Now().UnixNano())

	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%v.%d", name, rand.Intn(10000)),
		},
		Reason:  eventReason,
		Message: eventMessage,
		InvolvedObject: v1.ObjectReference{
			Kind:            "Pod",
			Namespace:       namespace,
			Name:            name,
			UID:             UID, // eg: "62f2e232-542f-40b6-9495-97ab3e443c1d"
			APIVersion:      "v1",
			ResourceVersion: fmt.Sprintf("%d", rv),
			FieldPath:       "spec.containers{mycontainer}",
		},
		Source: v1.EventSource{
			Component: "kubelet",
			Host:      "kublet.node1",
		},
		Count:          1,
		FirstTimestamp: eventTime,
		LastTimestamp:  eventTime,
		Type:           eventType, // v1.EventTypeNormal, v1.EventTypeWarning
	}
}
