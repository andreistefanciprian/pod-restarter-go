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
