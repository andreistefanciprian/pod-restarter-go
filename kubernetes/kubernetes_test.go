package kubernetes

import (
	"context"
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
		name                  string
		events                []runtime.Object
		namespace             string
		eventReason           string
		errorMessage          string
		expectedPodsWithError int
	}{
		// This test is looking for Events that match Reason and Message in a namespace
		// 1 Event will match Reason and Message
		{
			name: "Events that match Reason and Message in a namespace",
			events: []runtime.Object{
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "foo.1",
					},
					Reason:  "Scheduled",
					Message: "Successfully assigned default/foo to kublet.node1",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "foo",
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
						Name:      "foo.2",
					},
					Reason:  "FailedCreatePodSandBox",
					Message: "container veth name provided (eth0) already exists ....",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "foo.2",
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
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "bar.1",
					},
					Reason:  "FailedCreatePodSandBox",
					Message: "container veth name provided (eth0) already exists ....",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "test",
						Name:            "bar.1",
						UID:             "62f2e232-542f-40b6-9495-2",
						APIVersion:      "v1",
						ResourceVersion: "3777",
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
			namespace:             "default",
			eventReason:           "FailedCreatePodSandBox",
			errorMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 1,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 2 Events will match Reason and Message
		{
			name: "Events that match Reason and Message in a namespace",
			events: []runtime.Object{
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "foo.1",
					},
					Reason:  "Scheduled",
					Message: "Successfully assigned default/foo to kublet.node1",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "foo",
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
						Name:      "foo.2",
					},
					Reason:  "FailedCreatePodSandBox",
					Message: "container veth name provided (eth0) already exists ....",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "default",
						Name:            "foo.2",
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
				&corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "bar.1",
					},
					Reason:  "FailedCreatePodSandBox",
					Message: "container veth name provided (eth0) already exists ....",
					InvolvedObject: corev1.ObjectReference{
						Kind:            "Pod",
						Namespace:       "test",
						Name:            "bar.1",
						UID:             "62f2e232-542f-40b6-9495-2",
						APIVersion:      "v1",
						ResourceVersion: "3777",
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
			namespace:             "",
			eventReason:           "FailedCreatePodSandBox",
			errorMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 2,
		},
		// This test is looking for Events that match Reason and Message in a namespace
		// 0 Events will match Reason and Message
		{
			name:                  "No Events in a namespace",
			events:                []runtime.Object{},
			namespace:             "default",
			eventReason:           "FailedCreatePodSandBox",
			errorMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 0,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 0 Events will match Reason and Message across all namespaces
		{
			name:                  "No Events across all namespaces",
			events:                []runtime.Object{},
			namespace:             "",
			eventReason:           "FailedCreatePodSandBox",
			errorMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 0,
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
			if err != nil {
				t.Fatalf("Unexpected error getting Pod Events: %s", err.Error())
			} else if test.expectedPodsWithError != len(podEvents) {
				t.Fatalf("Unexpected Number of Events with Reason and Error Message found. Expected: %d. Found: %d", test.expectedPodsWithError, len(podEvents))
			}
		})
	}
}
