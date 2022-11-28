package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	e "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Logger interface {
	Print(v ...any)
	Println(v ...any)
	Printf(format string, v ...any)
}

// podRestarter holds K8s parameters
type podRestarter struct {
	logger     Logger
	kubeconfig *string
	ctx        context.Context
	clientset  *kubernetes.Clientset
}

// podDetails holds data associated with a Pod
type podDetails struct {
	podName           string
	podNamespace      string
	hasOwner          bool
	ownerData         interface{}
	phase             v1.PodPhase
	creationTimestamp time.Time
	deletionTimestamp *metav1.Time
}

// podEvent holds events data associated with a Pod
type podEvent struct {
	podName        string
	podNamespace   string
	eventType      string
	reason         string
	message        string
	firstTimestamp time.Time
	lastTimestamp  time.Time
}

// discover if kubeconfig creds are inside a Pod or outside the cluster
func (p *podRestarter) k8sClient() (*kubernetes.Clientset, error) {
	// read and parse kubeconfig
	config, err := rest.InClusterConfig() // creates the in-cluster config
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *p.kubeconfig) // creates the out-cluster config
		if err != nil {
			msg := fmt.Sprintf("The kubeconfig cannot be loaded: %v\n", err)
			return nil, errors.New(msg)
		}
		p.logger.Println("Running from OUTSIDE the cluster")
	} else {
		p.logger.Println("Running from INSIDE the cluster")
	}

	// create the clientset
	p.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		msg := fmt.Sprintf("The clientset cannot be created: %v\n", err)
		return nil, errors.New(msg)
	}
	return p.clientset, nil
}

// returns a map with Pending Pods (podName:podNamespace)
func (p *podRestarter) getPendingPods(namespace string) ([]podDetails, error) {
	api := p.clientset.CoreV1()
	var podData podDetails
	var podsData []podDetails
	// var pendingPods = make(map[string]string)

	// list all Pods in Pending state
	pods, err := api.Pods(namespace).List(
		p.ctx,
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
		podData = podDetails{
			podName:           pod.ObjectMeta.Name,
			podNamespace:      pod.ObjectMeta.Namespace,
			phase:             pod.Status.Phase,
			ownerData:         pod.ObjectMeta.OwnerReferences,
			creationTimestamp: pod.ObjectMeta.CreationTimestamp.Time,
			deletionTimestamp: pod.ObjectMeta.DeletionTimestamp,
		}

		// check if Pod has owner/controller
		if len(pod.ObjectMeta.OwnerReferences) > 0 {
			podData.hasOwner = true
		}

		podsData = append(podsData, podData)
	}
	p.logger.Printf("There is a TOTAL of %d Pods in Pending state in the cluster\n", len(podsData))
	return podsData, nil
}

// returns Pod Events
func (p *podRestarter) getPodEvents(pod, namespace string) ([]podEvent, error) {

	api := p.clientset.CoreV1()

	var podEvents []podEvent
	// get Pod events
	eventsStruct, err := api.Events(namespace).List(
		p.ctx,
		metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})

	if err != nil {
		msg := fmt.Sprintf("Could not go through Pod's Events: %s/%s\n%s", namespace, pod, err)
		return podEvents, errors.New(msg)
	}

	for _, item := range eventsStruct.Items {
		podEventData := podEvent{
			podName:        item.InvolvedObject.Name,
			podNamespace:   item.InvolvedObject.Namespace,
			reason:         item.Reason,
			eventType:      item.Type,
			message:        item.Message,
			firstTimestamp: item.FirstTimestamp.Time,
			lastTimestamp:  item.LastTimestamp.Time,
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
func (p *podRestarter) getPodDetails(pod, namespace string) (*podDetails, error) {
	api := p.clientset.CoreV1()
	var podRawData *v1.Pod
	var podData podDetails
	var err error

	podRawData, err = api.Pods(namespace).Get(
		p.ctx,
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
	podData = podDetails{
		podName:           podRawData.ObjectMeta.Name,
		podNamespace:      podRawData.ObjectMeta.Namespace,
		phase:             podRawData.Status.Phase,
		ownerData:         podRawData.ObjectMeta.OwnerReferences,
		creationTimestamp: podRawData.ObjectMeta.CreationTimestamp.Time,
	}

	if len(podRawData.ObjectMeta.OwnerReferences) > 0 {
		podData.hasOwner = true
	}
	return &podData, nil
}

// deletes a Pod
func (p *podRestarter) deletePod(pod, namespace string) error {
	api := p.clientset.CoreV1()

	err := api.Pods(namespace).Delete(
		p.ctx,
		pod,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}
	p.logger.Printf("DELETED Pod %s/%s", namespace, pod)
	return nil
}
