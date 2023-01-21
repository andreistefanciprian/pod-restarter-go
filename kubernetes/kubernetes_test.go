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
		testName      string
		mockedPods    []runtime.Object
		podNamespace  string
		podName       string
		expectSuccess bool
	}{
		// delete a Pod that exists
		{
			testName: "Delete existing Pod",
			mockedPods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod_exists",
						Namespace: "default",
					},
				},
			},
			podNamespace:  "default",
			podName:       "pod_exists",
			expectSuccess: true,
		},
		// delete a Pod that does not exist
		{
			testName:      "Delete Pod that does not exist",
			mockedPods:    []runtime.Object{},
			podNamespace:  "default",
			podName:       "pod_does_not_exist",
			expectSuccess: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.mockedPods...)
			err := clt.DeletePod(
				ctx,
				test.podName,
				test.podNamespace,
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
		testName              string
		mockedEvents          []runtime.Object
		eventNamespace        string
		eventReason           string
		eventMessage          string
		expectedPodsWithError int
	}{
		// This test is looking for Events that match Reason and Message in a namespace
		// 1 Event will match Reason and Message
		{
			testName: "Get Pod Events that match Reason and Message in a namespace",
			mockedEvents: []runtime.Object{
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
			eventNamespace:        "default",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 1,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 2 Events will match Reason and Message
		{
			testName: "Get Events that match Reason and Message across all namespaces",
			mockedEvents: []runtime.Object{
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
			eventNamespace:        "",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 2,
		},
		// This test is looking for Events that match Reason and Message in a namespace
		// 0 Events will match Reason and Message
		{
			testName:              "Get no matching Events from namespace",
			mockedEvents:          []runtime.Object{},
			eventNamespace:        "default",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 0,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 0 Events will match Reason and Message across all namespaces
		{
			testName:              "Get no matching Events across all namespaces",
			mockedEvents:          []runtime.Object{},
			eventNamespace:        "",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedPodsWithError: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.mockedEvents...)
			podEvents, err := clt.GetEvents(
				ctx,
				test.eventNamespace,
				test.eventReason,
				test.eventMessage,
			)
			if err != nil {
				t.Fatalf("Unexpected error getting Pod Events: %s", err.Error())
			} else if test.expectedPodsWithError != len(podEvents) {
				t.Fatalf("Unexpected Number of Events with Reason and Error Message found. Expected: %d. Found: %d", test.expectedPodsWithError, len(podEvents))
			}
		})
	}
}

func TestGetPodDetails(t *testing.T) {
	testCases := []struct {
		testName      string
		mockedPods    []runtime.Object
		podNamespace  string
		podName       string
		expectSuccess bool
	}{
		// Get the details of an existing Pod
		{
			testName: "Pod Exists",
			mockedPods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "default",
					},
				},
			},
			podNamespace:  "default",
			podName:       "foo",
			expectSuccess: true,
		},
		// Get the details of a Pod that does not exist
		{
			testName:      "Pod does not exist",
			mockedPods:    []runtime.Object{},
			podNamespace:  "default",
			podName:       "foo",
			expectSuccess: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.mockedPods...)
			_, err := clt.GetPodDetails(
				ctx,
				test.podName,
				test.podNamespace,
			)
			if err != nil && test.expectSuccess {
				t.Fatalf("Unexpected error geting existing Pod: %s", err.Error())
			} else if err == nil && !test.expectSuccess {
				t.Fatalf("We we're expecting an Error for getting details of a Pod that does not exist!")
			}
		})
	}
}
