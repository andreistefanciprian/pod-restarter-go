package kubernetes

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	e "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// discover if kubeconfig creds are inside a Pod or outside the cluster
func (p *PodRestarter) K8sClient() (*kubernetes.Clientset, error) {
	// read and parse kubeconfig
	config, err := rest.InClusterConfig() // creates the in-cluster config
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *p.Kubeconfig) // creates the out-cluster config
		if err != nil {
			msg := fmt.Sprintf("The kubeconfig cannot be loaded: %v\n", err)
			return nil, errors.New(msg)
		}
		p.Logger.Println("Running from OUTSIDE the cluster")
	} else {
		p.Logger.Println("Running from INSIDE the cluster")
	}

	// create the clientset for in-cluster/out-cluster config
	p.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		msg := fmt.Sprintf("The clientset cannot be created: %v\n", err)
		return nil, errors.New(msg)
	}
	return p.Clientset, nil
}

// returns a map with Pending Pods (podName:podNamespace)
func (p *PodRestarter) GetPendingPods(namespace string) ([]PodDetails, error) {
	api := p.Clientset.CoreV1()
	var podData PodDetails
	var podsData []PodDetails

	// list all Pods in Pending state
	pods, err := api.Pods(namespace).List(
		p.Ctx,
		metav1.ListOptions{
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
			FieldSelector: "status.phase=Pending",
		},
	)
	if err != nil {
		msg := fmt.Sprintf("Could not get a list of Pending Pods: \n%v", err)
		return podsData, errors.New(msg)
	}

	for _, pod := range pods.Items {
		podData = PodDetails{
			UID:               pod.ObjectMeta.UID,
			PodName:           pod.ObjectMeta.Name,
			PodNamespace:      pod.ObjectMeta.Namespace,
			ResourceVersion:   pod.ObjectMeta.ResourceVersion,
			Phase:             pod.Status.Phase,
			OwnerData:         pod.ObjectMeta.OwnerReferences,
			CreationTimestamp: pod.ObjectMeta.CreationTimestamp.Time,
			DeletionTimestamp: pod.ObjectMeta.DeletionTimestamp,
		}

		// check if Pod has owner/controller
		if len(pod.ObjectMeta.OwnerReferences) > 0 {
			podData.HasOwner = true
		}

		podsData = append(podsData, podData)
	}
	p.Logger.Printf("There is a TOTAL of %d Pods in Pending state in the cluster\n", len(podsData))
	return podsData, nil
}

// returns Pod Events
func (p *PodRestarter) GetPodEvents(pod, namespace string) ([]PodEvent, error) {

	api := p.Clientset.CoreV1()

	var podEvents []PodEvent
	// get Pod events
	eventsStruct, err := api.Events(namespace).List(
		p.Ctx,
		metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})

	if err != nil {
		msg := fmt.Sprintf("Could not go through Pod's Events: %s/%s\n%s", namespace, pod, err)
		return podEvents, errors.New(msg)
	}

	for _, item := range eventsStruct.Items {
		podEventData := PodEvent{
			UID:             item.InvolvedObject.UID,
			PodName:         item.InvolvedObject.Name,
			PodNamespace:    item.InvolvedObject.Namespace,
			ResourceVersion: item.InvolvedObject.ResourceVersion,
			Reason:          item.Reason,
			EventType:       item.Type,
			Message:         item.Message,
			FirstTimestamp:  item.FirstTimestamp.Time,
			LastTimestamp:   item.LastTimestamp.Time,
		}
		podEvents = append(podEvents, podEventData)
	}

	if len(podEvents) == 0 {
		msg := fmt.Sprintf(
			"Pod has 0 Events. Probably it does not exist or it does not have any events in the last hour: %s/%s",
			namespace, pod,
		)
		return podEvents, errors.New(msg)
	}
	return podEvents, nil
}

// returns Pod details
func (p *PodRestarter) GetPodDetails(pod, namespace string) (*PodDetails, error) {
	api := p.Clientset.CoreV1()
	var podRawData *v1.Pod
	var podData PodDetails
	var err error

	podRawData, err = api.Pods(namespace).Get(
		p.Ctx,
		pod,
		metav1.GetOptions{},
	)
	if e.IsNotFound(err) {
		msg := fmt.Sprintf("Pod %s/%s does not exist anymore", namespace, pod)
		return &podData, errors.New(msg)
	} else if statusError, isStatus := err.(*e.StatusError); isStatus {
		msg := fmt.Sprintf("Error getting pod %s/%s: %v",
			namespace, pod, statusError.ErrStatus.Message)
		return &podData, errors.New(msg)
	} else if err != nil {
		msg := fmt.Sprintf("Pod %s/%s has a problem: %v", namespace, pod, err)
		return &podData, errors.New(msg)
	}
	podData = PodDetails{
		PodName:           podRawData.ObjectMeta.Name,
		PodNamespace:      podRawData.ObjectMeta.Namespace,
		Phase:             podRawData.Status.Phase,
		OwnerData:         podRawData.ObjectMeta.OwnerReferences,
		CreationTimestamp: podRawData.ObjectMeta.CreationTimestamp.Time,
	}

	if len(podRawData.ObjectMeta.OwnerReferences) > 0 {
		podData.HasOwner = true
	}
	return &podData, nil
}

// deletes a Pod
func (p *PodRestarter) DeletePod(pod, namespace string) error {
	api := p.Clientset.CoreV1()

	err := api.Pods(namespace).Delete(
		p.Ctx,
		pod,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}
	p.Logger.Printf("DELETED Pod %s/%s", namespace, pod)
	return nil
}
