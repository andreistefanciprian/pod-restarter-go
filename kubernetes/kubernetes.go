package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	e "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient interface {
	DeletePod(ctx context.Context, pod, namespace string) error
	GenerateToBeDeletedPodList(ctx context.Context, namespace, eventReason, errorMessage string, counter, pollingInterval int) (map[string]string, error)
	PodChecks(ctx context.Context, podName, podNamespace string) error
}

// NewK8sClient discover if kubeconfig creds are inside a Pod or outside the cluster and return a clientSet
func NewK8sClient(kubeconfig string) (*kubeClient, error) {
	// read and parse kubeconfig
	config, err := rest.InClusterConfig() // creates the in-cluster config
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig) // creates the out-cluster config
		if err != nil {
			msg := fmt.Sprintf("The kubeconfig cannot be loaded: %v\n", err)
			return nil, errors.New(msg)
		}
		log.Println("Running from OUTSIDE the cluster")
	} else {
		log.Println("Running from INSIDE the cluster")
	}

	// create the clientset for in-cluster/out-cluster config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		msg := fmt.Sprintf("The clientset cannot be created: %v\n", err)
		return nil, errors.New(msg)
	}

	return &kubeClient{
		clientSet: clientset,
	}, nil
}

// listPods returns a list with all the Pods in the Cluster
func (c *kubeClient) listPods(ctx context.Context, namespace string) (*[]PodDetails, error) {
	api := c.clientSet.CoreV1()
	var podData PodDetails
	var podsData []PodDetails

	// list all Pods in Pending state
	pods, err := api.Pods(namespace).List(
		ctx,
		metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			// FieldSelector: "status.phase=Pending",
		},
	)
	if err != nil {
		msg := fmt.Sprintf("Could not get a list of Pods: \n%v", err)
		return &podsData, errors.New(msg)
	}

	for _, pod := range pods.Items {
		podData = PodDetails{
			UID:               pod.ObjectMeta.UID,
			PodName:           pod.ObjectMeta.Name,
			PodNamespace:      pod.ObjectMeta.Namespace,
			ResourceVersion:   pod.ObjectMeta.ResourceVersion,
			Phase:             pod.Status.Phase,
			ContainerStatuses: pod.Status.ContainerStatuses,
			OwnerReferences:   pod.ObjectMeta.OwnerReferences,
			CreationTimestamp: pod.ObjectMeta.CreationTimestamp.Time,
			DeletionTimestamp: pod.ObjectMeta.DeletionTimestamp,
		}
		podsData = append(podsData, podData)
	}
	log.Printf("There is a TOTAL of %d Pods in the cluster\n", len(podsData))
	return &podsData, nil
}

// GetEvents returns a list of namespaced Events that match Reason
func (c *kubeClient) GetEvents(ctx context.Context, namespace, eventReason, errorMessage string) ([]PodEvent, error) {
	api := c.clientSet.CoreV1()
	var podEvents []PodEvent

	eventList, err := api.Events(namespace).List(
		ctx,
		metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{Kind: "Pod"},
			// ResourceVersion: "46641835",
		})

	if err != nil {
		msg := fmt.Sprintf("Could not get Events in namespace: %s\n%s", namespace, err)
		return podEvents, errors.New(msg)
	}

	// keep only Events that match event Reason (eg: FailedCreatePodSandBox)
	// keep only Events that have errorMessage
	// TO ADD filter out Events older than polling interval
	for _, item := range eventList.Items {
		if item.Reason == eventReason && strings.Contains(item.Message, errorMessage) {
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
	}
	return podEvents, nil
}

// getPodEvents returns Pod Events
func (c *kubeClient) getPodEvents(ctx context.Context, pod, namespace string) ([]PodEvent, error) {

	api := c.clientSet.CoreV1()

	var podEvents []PodEvent
	// get Pod events
	eventsStruct, err := api.Events(namespace).List(
		ctx,
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

// GetPodDetails returns Pod details
func (c *kubeClient) GetPodDetails(ctx context.Context, pod, namespace string) (*PodDetails, error) {

	api := c.clientSet.CoreV1()
	var item *v1.Pod
	var podData PodDetails
	var err error

	item, err = api.Pods(namespace).Get(
		ctx,
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
		UID:               item.ObjectMeta.UID,
		PodName:           item.ObjectMeta.Name,
		PodNamespace:      item.ObjectMeta.Namespace,
		ResourceVersion:   item.ObjectMeta.ResourceVersion,
		Phase:             item.Status.Phase,
		ContainerStatuses: item.Status.ContainerStatuses,
		OwnerReferences:   item.ObjectMeta.OwnerReferences,
		CreationTimestamp: item.ObjectMeta.CreationTimestamp.Time,
		DeletionTimestamp: item.ObjectMeta.DeletionTimestamp,
	}
	return &podData, nil
}

// DeletePod deletes a Pod
func (c *kubeClient) DeletePod(ctx context.Context, pod, namespace string) error {
	api := c.clientSet.CoreV1()

	err := api.Pods(namespace).Delete(
		ctx,
		pod,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}
	log.Printf("DELETED Pod %s/%s", namespace, pod)
	return nil
}

// GenerateToBeDeletedPodList generates a map of Pods that match Event Reason and Error Message
func (c *kubeClient) GenerateToBeDeletedPodList(ctx context.Context, namespace, eventReason, errorMessage string, counter, pollingInterval int) (map[string]string, error) {

	var uniquePodList = make(map[string]string)

	// get a list of Events that match Reason
	eventList, err := c.GetEvents(ctx, namespace, eventReason, errorMessage)
	if err != nil {
		return uniquePodList, err
	}

	// Filter out Events that are older than polling interval
	eventMaxAge := time.Now().Add(-time.Duration(pollingInterval) * time.Second)
	if counter > 0 {
		eventList = removeOlderEvents(eventList, eventMaxAge)
	}

	log.Printf("There is a total of %d Events with Reason: %s", len(eventList), eventReason) // DEBUG

	// generate a unique list of Pods that match Event Reason
	// we do this because a Pod might have multiple Events with the same Reason
	uniquePodList = getUniqueListOfPods(eventList)

	log.Printf("There is a total of %d Pods with Reason: %s", len(uniquePodList), eventReason) // DEBUG

	return uniquePodList, nil
}
