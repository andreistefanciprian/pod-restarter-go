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

// discover if kubeconfig creds are inside a Pod or outside the cluster and return a clientSet
func NewK8sClient(kubeconfig string) (*KubeClient, error) {
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

	fmt.Println(config.CurrentContext)
	// create the clientset for in-cluster/out-cluster config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		msg := fmt.Sprintf("The clientset cannot be created: %v\n", err)
		return nil, errors.New(msg)
	}
	return &KubeClient{
		Clientset: clientset,
	}, nil
}

// returns a list with all the Pods in the Cluster
func (c *KubeClient) ListPods(ctx context.Context, namespace string) (*[]PodDetails, error) {
	api := c.Clientset.CoreV1()
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
func (c *KubeClient) GetEvents(ctx context.Context, namespace, eventReason, errorMessage string) ([]PodEvent, error) {
	// defer timeTrack(time.Now(), "GetEvents") // calculates the time it takes to execute this method

	api := c.Clientset.CoreV1()
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

// returns Pod Events
func (c *KubeClient) GetPodEvents(ctx context.Context, pod, namespace string) ([]PodEvent, error) {

	api := c.Clientset.CoreV1()

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

// returns Pod details
func (c *KubeClient) GetPodDetails(ctx context.Context, pod, namespace string) (*PodDetails, error) {
	// defer timeTrack(time.Now(), "GetPodDetails") // calculates the time it takes to execute this method

	api := c.Clientset.CoreV1()
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
	}
	return &podData, nil
}

// deletes a Pod
func (c *KubeClient) DeletePod(ctx context.Context, pod, namespace string) error {
	api := c.Clientset.CoreV1()

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

// generates a map of Pods that match Event Reason and Error Message
func (c *KubeClient) GenerateToBeDeletedPodList(ctx context.Context, namespace, eventReason, errorMessage string, counter, pollingInterval int) (map[string]string, error) {

	var uniquePodList = make(map[string]string)

	// get a list of Events that match Reason
	eventList, err := c.GetEvents(ctx, namespace, eventReason, errorMessage)
	if err != nil {
		return uniquePodList, err
	}

	// Filter out Events that are older than polling interval
	eventMaxAge := time.Now().Add(-time.Duration(pollingInterval) * time.Second)
	if counter > 0 {
		eventList = RemoveOlderEvents(eventList, eventMaxAge)
	}

	log.Printf("There is a total of %d Events with Reason: %s", len(eventList), eventReason) // DEBUG

	// generate a unique list of Pods that match Event Reason
	// we do this because a Pod might have multiple Events with the same Reason
	uniquePodList = GetUniqueListOfPods(eventList)

	log.Printf("There is a total of %d Pods with Reason: %s", len(uniquePodList), eventReason) // DEBUG

	return uniquePodList, nil
}

// returns nil if Pod
// 1. exists
// 2. has Owner
// 3. has not been scheduled to be deleted
// 4. and is not in a Healthy state (eg: Pending, Failed or Running with unhealthy containers)
func (c *KubeClient) PodChecks(ctx context.Context, podName, podNamespace string) error {
	// verify if Pod exists
	podInfo, err := c.GetPodDetails(ctx, podName, podNamespace)
	if err != nil {
		return err
	}

	// verify Pod has owner
	err = podInfo.VerifyPodHasOwner()
	if err != nil {
		return err
	}

	// verify Pod is scheduled to be deleted
	err = podInfo.VerifyPodScheduledToBeDeleted()
	if err != nil {
		return err
	}

	// verify Pod is in an Unhealthy state
	err = podInfo.VerifyPodStatus()
	if err != nil {
		return nil
	} else {
		msg := fmt.Sprintf("Pod is in a Healthy State: %s/%s", podNamespace, podName)
		return errors.New(msg)
	}
}

// returns error if Pod is in a Pending, Failed or Running (with unhealthy containers) state
func (p *PodDetails) VerifyPodStatus() error {
	// defer timeTrack(time.Now(), "VerifyPodStatus") // calculates the time it takes to execute this method

	switch p.Phase {

	case "Pending":
		msg := fmt.Sprintf(
			"Pod is in a %s state: %s/%s",
			p.Phase, p.PodNamespace, p.PodName,
		)
		return errors.New(msg)

	case "Running":
		if len(p.ContainerStatuses) != 0 {
			for _, cst := range p.ContainerStatuses {
				if cst.State.Terminated == nil {
					continue
				}
				if cst.State.Terminated.Reason == "Completed" && cst.State.Terminated.ExitCode == 0 {
					continue
				}
				msg := fmt.Sprintf(
					"Pod is in a %s state and has issues: %s/%s\n%+v",
					p.Phase, p.PodNamespace, p.PodName,
					p.ContainerStatuses,
				)
				return errors.New(msg)
			}

			log.Printf(
				"Pod is in a %s state and is healthy: %s/%s",
				p.Phase, p.PodNamespace, p.PodName,
			)
			return nil

		}
		log.Printf(
			"Pod is in a %s state and has been evacuated?: %s/%s\n%+v",
			p.Phase, p.PodNamespace, p.PodName,
			p.ContainerStatuses,
		)
		return nil

	case "Failed":
		msg := fmt.Sprintf(
			"Pod is in a %s state: %s/%s",
			p.Phase, p.PodNamespace, p.PodName,
		)
		return errors.New(msg)

	case "Succeeded":
		log.Printf(
			"Pod is in a %s state: %s/%s",
			p.Phase, p.PodNamespace, p.PodName,
		)
		return nil

	case "Unknown":
		msg := fmt.Sprintf(
			"Pod is in a %s state: %s/%s",
			p.Phase, p.PodNamespace, p.PodName,
		)
		return errors.New(msg)
	}

	msg := fmt.Sprintf(
		"Pod is in a %s state ????????: %s/%s",
		p.Phase, p.PodNamespace, p.PodName,
	)
	return errors.New(msg)
}

// verify if element in slice
func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// returns nil if Pod has owner
func (p *PodDetails) VerifyPodHasOwner() error {
	if len(p.OwnerReferences) > 0 {
		return nil
	}
	msg := fmt.Sprintf(
		"Pod does not have owner/controller: %s/%s",
		p.PodNamespace, p.PodName,
	)
	return errors.New(msg)
}

// returns nil if Pod is not scheduled to be deleted
func (p *PodDetails) VerifyPodScheduledToBeDeleted() error {
	// verify Pod has not been scheduled to be deleted
	if p.DeletionTimestamp != nil {
		msg := fmt.Sprintf(
			"Pod has already been scheduled to be deleted: %s/%s\n%v",
			p.PodNamespace, p.PodName, p.DeletionTimestamp,
		)
		return errors.New(msg)
	}
	return nil
}

// returns a unique list of Pods that have Events that match Reason
func GetUniqueListOfPods(events []PodEvent) map[string]string {
	// defer timeTrack(time.Now(), "GetUniqueListOfPods") // calculates the time it takes to execute this method

	var uniquePodList = make(map[string]string)
	var uniqueUIDsList []string

	for _, event := range events {
		if contains(uniqueUIDsList, string(event.UID)) {
			continue
		}
		uniquePodList[event.PodName] = event.PodNamespace
		uniqueUIDsList = append(uniqueUIDsList, string(event.UID))
	}
	return uniquePodList
}

// returns a slice of latest Events not older than eventMaxAge
func RemoveOlderEvents(events []PodEvent, eventMaxAge time.Time) []PodEvent {
	var latestEvents []PodEvent
	for _, event := range events {

		if event.LastTimestamp.Before(eventMaxAge) {
			continue
		}
		latestEvents = append(latestEvents, event)
	}
	return latestEvents
}

// timeTrack calculates how long it takes to execute a function
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%v ran in %v \n", name, elapsed)
}
