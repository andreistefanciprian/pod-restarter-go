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
		{
			name: "existing_pod",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "namespace1",
					},
				},
			},
			targetNamespace: "namespace1",
			targetPod:       "pod1",
			expectSuccess:   true,
		},
		{
			name:            "not_existing_pod",
			pods:            []runtime.Object{},
			targetNamespace: "namespace1",
			targetPod:       "pod1",
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
			if err != nil && !test.expectSuccess {
				fmt.Print(err.Error())
			}
		})
	}
}
