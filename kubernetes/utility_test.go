package kubernetes

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVerifyPodStatus(t *testing.T) {
	type Inputs struct {
		pod PodDetails
	}

	type Expected struct {
		err error
	}

	tests := map[string]struct {
		inputs   Inputs
		expected Expected
	}{
		"Verify Pod is in Pending Phase": {
			inputs: Inputs{
				pod: PodDetails{
					UID:             "1",
					PodName:         "foo",
					PodNamespace:    "default",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       "foo",
							UID:        "1",
						},
					},
					Phase: v1.PodPending,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "nginx",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason:  "ImagePullBackOff",
									Message: "Back-off pulling image ...",
								},
								Running:    nil,
								Terminated: nil,
							},
							LastTerminationState: v1.ContainerState{
								Waiting:    nil,
								Running:    nil,
								Terminated: nil,
							},
							Ready:        false,
							RestartCount: 0,
						},
					},
					CreationTimestamp: time.Now(),
					DeletionTimestamp: nil,
				},
			},
			expected: Expected{err: fmt.Errorf("Pod is in a Pending state: default/foo")},
		},
		"Verify Pod is in Running Phase with failed container": {
			inputs: Inputs{
				pod: PodDetails{
					UID:             "1",
					PodName:         "foo",
					PodNamespace:    "default",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       "foo",
							UID:        "1",
						},
					},
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "failing_container",
							State: v1.ContainerState{
								Waiting: nil,
								Running: nil,
								Terminated: &v1.ContainerStateTerminated{
									ExitCode: 2,
								},
							},
							LastTerminationState: v1.ContainerState{
								Waiting:    nil,
								Running:    nil,
								Terminated: nil,
							},
							Ready:        true,
							RestartCount: 0,
						},
						{
							Name: "good_container",
							State: v1.ContainerState{
								Waiting:    nil,
								Running:    &v1.ContainerStateRunning{},
								Terminated: nil,
							},
							LastTerminationState: v1.ContainerState{
								Waiting:    nil,
								Running:    nil,
								Terminated: nil,
							},
							Ready:        false,
							RestartCount: 0,
						},
					},
					CreationTimestamp: time.Now(),
					DeletionTimestamp: nil,
				},
			},
			expected: Expected{err: fmt.Errorf("Pod is in a Running state and has issues: default/foo")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			err := tc.inputs.pod.verifyPodStatus()

			if tc.expected.err != nil {
				require.Error(tc.expected.err)
				assert.EqualError(err, tc.expected.err.Error(), "Expected error: %v Got: %v", tc.expected.err, err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestVerifyPodHasOwner(t *testing.T) {
	type Inputs struct {
		pod PodDetails
	}

	type Expected struct {
		err error
	}

	tests := map[string]struct {
		inputs   Inputs
		expected Expected
	}{
		"Verify no error is thrown when pod has owner": {
			inputs: Inputs{
				pod: PodDetails{
					PodName:      "foo",
					PodNamespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       "foo",
							UID:        "1",
						},
					},
				},
			},
			expected: Expected{err: nil},
		},
		"Verify error is thrown if pod has no owner": {
			inputs: Inputs{
				pod: PodDetails{
					PodName:         "foo",
					PodNamespace:    "default",
					OwnerReferences: []metav1.OwnerReference{},
				},
			},
			expected: Expected{err: fmt.Errorf("Pod does not have owner/controller: default/foo")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			err := tc.inputs.pod.verifyPodHasOwner()

			if tc.expected.err != nil {
				require.Error(tc.expected.err)
				assert.EqualError(err, tc.expected.err.Error(), "Expected error: %v Got: %v", tc.expected.err, err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestVerifyPodScheduledToBeDeleted(t *testing.T) {
	deletionTimestamp := &metav1.Time{Time: time.Now()}
	creationTimestamp := time.Now().Add(-time.Second * 10)

	type Inputs struct {
		pod PodDetails
	}

	type Expected struct {
		err error
	}

	tests := map[string]struct {
		inputs   Inputs
		expected Expected
	}{
		"Verify pod without deletion schedule": {
			inputs: Inputs{
				pod: PodDetails{
					PodName:           "foo",
					PodNamespace:      "default",
					CreationTimestamp: creationTimestamp,
					DeletionTimestamp: nil,
				},
			},
			expected: Expected{err: nil},
		},
		"Verify pod with deletion schedule": {
			inputs: Inputs{
				pod: PodDetails{
					PodName:           "foo",
					PodNamespace:      "default",
					CreationTimestamp: creationTimestamp,
					DeletionTimestamp: deletionTimestamp,
				},
			},
			expected: Expected{err: fmt.Errorf("Pod has already been scheduled to be deleted: default/foo\n%v", deletionTimestamp)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			err := tc.inputs.pod.verifyPodScheduledToBeDeleted()

			if tc.expected.err != nil {
				require.Error(tc.expected.err)
				assert.EqualError(err, tc.expected.err.Error(), "Expected error: %v Got: %v", tc.expected.err, err)
			} else {
				require.NoError(err)
			}
		})
	}
}
