package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
				makePod("foo", "default", 1, corev1.PodRunning, "abc1"),
			},
			podNamespace:  "default",
			podName:       "foo",
			expectSuccess: true,
		},
		// delete a Pod that does not exist
		{
			testName:      "Delete Pod that does not exist",
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
				makeEvent("foo", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid1"),
				makeEvent("foo", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid1"),
				makeEvent("foo", "test", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid2"),
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
				makeEvent("foo", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid1"),
				makeEvent("foo", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid1"),
				makeEvent("foo", "test", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid2"),
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
				makePod("foo", "default", 1, corev1.PodRunning, "abc1"),
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

func TestGenerateToBeDeletedPodList(t *testing.T) {
	testCases := []struct {
		testName              string
		mockedEvents          []runtime.Object
		eventNamespace        string
		eventReason           string
		eventMessage          string
		counter               int
		pollingInterval       int
		expectedUniquePodList int
	}{
		// This test is looking for Events that match Reason and Message in a namespace
		// 1 Event will match Reason and Message
		{
			testName: "Get Pod Events that match Reason and Message in default namespace",
			mockedEvents: []runtime.Object{
				makeEvent("pod_1", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid1"),
				makeEvent("pod_1", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid1"),
				makeEvent("pod_1", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 3, "uid1"),
				makeEvent("pod_2", "test", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid2"),
				makeEvent("pod_3", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid3"),
				makeEvent("pod_3", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid3"),
				makeEvent("pod_3", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 3, "uid3"),
				makeEvent("pod_4", "test2", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 1, "uid4"),
			},
			eventNamespace:        "default",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedUniquePodList: 2,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 2 Events will match Reason and Message
		{
			testName: "Get Events that match Reason and Message across all namespaces",
			mockedEvents: []runtime.Object{
				makeEvent("pod_1", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid1"),
				makeEvent("pod_1", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid1"),
				makeEvent("pod_1", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 3, "uid1"),
				makeEvent("pod_2", "test", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid2"),
				makeEvent("pod_3", "default", "Scheduled", "Successfully assigned pod to kublet.node1", "Normal", 1, "uid3"),
				makeEvent("pod_3", "default", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid3"),
				makeEvent("pod_4", "test2", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 1, "uid4"),
				makeEvent("pod_4", "test2", "FailedCreatePodSandBox", "container veth name provided (eth0) already exists ....", "Warning", 2, "uid4"),
			},
			eventNamespace:        "",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedUniquePodList: 4,
		},
		// This test is looking for Events that match Reason and Message in a namespace
		// 0 Events will match Reason and Message
		{
			testName:              "Get no matching Events from namespace",
			mockedEvents:          []runtime.Object{},
			eventNamespace:        "default",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedUniquePodList: 0,
		},
		// This test is looking for Events that match Reason and Message across all namespaces
		// 0 Events will match Reason and Message across all namespaces
		{
			testName:              "Get no matching Events across all namespaces",
			mockedEvents:          []runtime.Object{},
			eventNamespace:        "",
			eventReason:           "FailedCreatePodSandBox",
			eventMessage:          "container veth name provided (eth0) already exists",
			expectedUniquePodList: 0,
		},
		// // Test when getting an error while getting Events
		// {
		// 	testName:              "Get error while getting Events",
		// 	mockedEvents:          []runtime.Object{},
		// 	eventNamespace:        "",
		// 	eventReason:           "FailedCreatePodSandBox",
		// 	eventMessage:          "container veth name provided (eth0) already exists",
		// 	expectedUniquePodList: 0,
		// },
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			var clt kubeClient
			var ctx = context.TODO()
			clt.clientSet = fake.NewSimpleClientset(test.mockedEvents...)
			uniquePodList, err := clt.GenerateToBeDeletedPodList(
				ctx,
				test.eventNamespace,
				test.eventReason,
				test.eventMessage,
				0,
				10,
			)

			if err != nil {
				assert.NotNil(t, err)
				assert.Equal(t, fmt.Sprintf("Could not get Events in namespace: %s\n%s", test.eventNamespace, err), err.Error())
			} else if test.expectedUniquePodList != len(uniquePodList) {
				require.NoError(t, err)
				assert.Nil(t, err)
				assert.Equal(t, test.expectedUniquePodList, len(uniquePodList))
			}
		})
	}
}
