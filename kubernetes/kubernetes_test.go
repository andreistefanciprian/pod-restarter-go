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
