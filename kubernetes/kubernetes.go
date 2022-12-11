package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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

// returns a list with all the Pods in the Cluster
func (p *PodRestarter) ListPods(ctx context.Context, namespace string) (*[]PodDetails, error) {
	api := p.Clientset.CoreV1()
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
	p.Logger.Printf("There is a TOTAL of %d Pods in the cluster\n", len(podsData))
	return &podsData, nil
}

// GetEvents returns a list of namespaced Events that match Reason
func (p *PodRestarter) GetEvents(ctx context.Context, namespace string, eventReason string) ([]PodEvent, error) {
	// defer timeTrack(time.Now(), "GetEvents") // calculates the time it takes to execute this method

	api := p.Clientset.CoreV1()
	var podEvents []PodEvent

	eventList, err := api.Events(namespace).List(
		ctx,
		metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{Kind: "Pod"},
		})

	if err != nil {
		msg := fmt.Sprintf("Could not get Events in namespace: %s\n%s", namespace, err)
		return podEvents, errors.New(msg)
	}

	// keep only Events that match event Reason (eg: FailedCreatePodSandBox)
	// TO ADD filter out Events older than polling interval
	for _, item := range eventList.Items {
		if item.Reason == eventReason {
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
func (p *PodRestarter) GetPodEvents(ctx context.Context, pod, namespace string) ([]PodEvent, error) {

	api := p.Clientset.CoreV1()

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
func (p *PodRestarter) GetPodDetails(ctx context.Context, pod, namespace string) (*PodDetails, error) {
	// defer timeTrack(time.Now(), "GetPodDetails") // calculates the time it takes to execute this method

	api := p.Clientset.CoreV1()
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
func (p *PodRestarter) DeletePod(ctx context.Context, pod, namespace string) error {
	api := p.Clientset.CoreV1()

	err := api.Pods(namespace).Delete(
		ctx,
		pod,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}
	p.Logger.Printf("DELETED Pod %s/%s", namespace, pod)
	return nil
}

// utility functions

// returns true if Pod is in a healthy/Succeded state
func VerifyPodStatus(pod PodDetails) bool {
	// defer timeTrack(time.Now(), "VerifyPodStatus") // calculates the time it takes to execute this method

	switch pod.Phase {
	case "Pending":
		log.Printf(
			"Pod is in a %s state: %s/%s",
			pod.Phase, pod.PodNamespace, pod.PodName,
		)
		return false

	case "Running":
		if len(pod.ContainerStatuses) != 0 {
			for _, cst := range pod.ContainerStatuses {
				if cst.State.Terminated == nil {
					continue
				}
				if cst.State.Terminated.Reason == "Completed" && cst.State.Terminated.ExitCode == 0 {
					continue
				}
				log.Printf(
					"Pod is in a %s state and has issues: %s/%s\n%+v",
					pod.Phase, pod.PodNamespace, pod.PodName,
					pod.ContainerStatuses,
				)
				return false
			}

			log.Printf(
				"Pod is in a %s state and is healthy: %s/%s",
				pod.Phase, pod.PodNamespace, pod.PodName,
			)
			return true

		}
		log.Printf(
			"Pod is in a %s state and has been evacuated?: %s/%s\n%+v",
			pod.Phase, pod.PodNamespace, pod.PodName,
			pod.ContainerStatuses,
		)
		return true

	case "Failed":
		log.Printf(
			"Pod is in a %s state: %s/%s",
			pod.Phase, pod.PodNamespace, pod.PodName,
		)
		return false

	case "Succeeded":
		log.Printf(
			"Pod is in a %s state: %s/%s",
			pod.Phase, pod.PodNamespace, pod.PodName,
		)
		return true

	case "Unknown":
		log.Printf(
			"Pod is in a %s state: %s/%s",
			pod.Phase, pod.PodNamespace, pod.PodName,
		)
		return false
	}
	log.Printf(
		"Pod is in a %s state ????????: %s/%s",
		pod.Phase, pod.PodNamespace, pod.PodName,
	)
	return false
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

// returns True if Pod has owner
func VerifyPodHasOwner(pod PodDetails) bool {
	if len(pod.OwnerReferences) > 0 {
		return true
	}
	log.Printf(
		"Pod does not have owner/controller: %s/%s",
		pod.PodNamespace, pod.PodName,
	)
	return false
}

// returns True if Pod is scheduled to be deleted
func VerifyPodWasDeleted(pod PodDetails) bool {
	// verify Pod has not been scheduled to be deleted
	if pod.DeletionTimestamp != nil {
		log.Printf(
			"Pod has already been scheduled to be deleted: %s/%s\n%v",
			pod.PodNamespace, pod.PodName, pod.DeletionTimestamp,
		)
		return true
	}
	return false
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
