package kubernetes

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeletePod(t *testing.T) {
	testCases := []struct {
		name            string
		pods            []runtime.Object
		targetNamespace string
		targetPod       string
		expectSuccess   bool
	}{
		// delete a Pod that exists
		{
			name: "pod_exists",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod_exists",
						Namespace: "default",
					},
				},
			},
			targetNamespace: "default",
			targetPod:       "pod_exists",
			expectSuccess:   true,
		},
		// delete a Pod that does not exist
		{
			name:            "pod_does_not_exist",
			pods:            []runtime.Object{},
			targetNamespace: "default",
			targetPod:       "pod_does_not_exist",
			expectSuccess:   false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.pods...)
			err := clt.DeletePod(
				ctx,
				test.targetPod,
				test.targetNamespace,
			)

			if err != nil && test.expectSuccess {
				t.Fatalf("Unexpected error deleting existing Pod: %s", err.Error())
			} else if err == nil && !test.expectSuccess {
				t.Fatalf("We we're expecting an error for deleting a Pod that does not exist: %s", err.Error())
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	testCases := []struct {
		name          string
		events        []runtime.Object
		namespace     string
		eventReason   string
		errorMessage  string
		expectSuccess bool
	}{
		{
			name: "events_with_error",
			events: []runtime.Object{
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "nginx-7c979dff44-2d4bh.1",
					},
					Reason:  "Scheduled",
					Message: "Successfully assigned test/nginx-7c979dff44-2d4bh to docker-desktop",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "nginx-7c979dff44-2d4bh",
						UID:             "62f2e232-542f-40b6-9495-97ab3e443c1d",
						APIVersion:      "v1",
						ResourceVersion: "3757",
						FieldPath:       "spec.containers{mycontainer}",
					},
					Source: corev1.EventSource{
						Component: "kubelet",
						Host:      "kublet.node1",
					},
					Count:          1,
					FirstTimestamp: metav1.Now(),
					LastTimestamp:  metav1.Now(),
					Type:           corev1.EventTypeNormal,
				},
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "nginx-7c979dff44-2d4bh.2",
					},
					Reason:  "FailedCreatePodSandBox",
					Message: "container veth name provided (eth0) already exists ....",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "nginx-7c979dff44-2d4bh.2",
						UID:             "62f2e232-542f-40b6-9495-97ab3e443c1d",
						APIVersion:      "v1",
						ResourceVersion: "3768",
						FieldPath:       "spec.containers{mycontainer}",
					},
					Source: corev1.EventSource{
						Component: "kubelet",
						Host:      "kublet.node1",
					},
					Count:          1,
					FirstTimestamp: metav1.Now(),
					LastTimestamp:  metav1.Now(),
					Type:           corev1.EventTypeWarning,
				},
			},
			namespace:     "default",
			eventReason:   "FailedCreatePodSandBox",
			errorMessage:  "container veth name provided (eth0) already exists",
			expectSuccess: true,
		},
		// no events
		{
			name:          "no_events",
			events:        []runtime.Object{},
			namespace:     "default",
			eventReason:   "FailedCreatePodSandBox",
			errorMessage:  "container veth name provided (eth0) already exists",
			expectSuccess: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.events...)
			podEvents, err := clt.GetEvents(
				ctx,
				test.namespace,
				test.eventReason,
				test.errorMessage,
			)
			fmt.Printf("Pod Events: %+v", podEvents) // DEBUG
			if err != nil && test.expectSuccess {
				t.Fatalf("Unexpected error getting Events: %s", err.Error())
			}
		})
	}
}
