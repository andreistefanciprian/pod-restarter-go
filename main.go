// This script runs in/out a K8s cluster
// Deletes Pods that are in a Pending state due to a particular error

// The script goes through this sequence of steps:
// - get an array of all Pending Pods that have the error event
// - for each Pending Pod that has the error event
//   - delete the Pod if it still exists and in a Pending state
//
// Script executes the above steps every n seconds

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	e "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// podRestarter holds k8s parameters
type podRestarter struct {
	errorLog   *log.Logger
	infoLog    *log.Logger
	kubeconfig *string
	ctx        context.Context
	clientset  *kubernetes.Clientset
}

func (p *podRestarter) k8sClient() (*kubernetes.Clientset, error) {
	// read and parse kubeconfig
	config, err := rest.InClusterConfig() // creates the in-cluster config
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *p.kubeconfig) // creates the out-cluster config
		if err != nil {
			msg := fmt.Sprintf("The kubeconfig cannot be loaded: %v\n", err)
			return nil, errors.New(msg)
		}
		p.infoLog.Println("Running from OUTSIDE the cluster")
	} else {
		p.infoLog.Println("Running from INSIDE the cluster")
	}

	// create the clientset
	p.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		msg := fmt.Sprintf("The clientset cannot be created: %v\n", err)
		return nil, errors.New(msg)
	}
	return p.clientset, nil
}

// get a map with Pending Pods (podName:podNamespace)
func (p *podRestarter) getPendingPods(namespace string) (map[string]string, error) {
	api := p.clientset.CoreV1()
	var pendingPods = make(map[string]string)

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
		return pendingPods, errors.New(msg)
	}

	for _, pod := range pods.Items {
		p.infoLog.Printf("Pod %s/%s is in Pending state", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		pendingPods[pod.ObjectMeta.Name] = pod.ObjectMeta.Namespace
	}
	p.infoLog.Printf("There is a TOTAL of %d Pods in Pending state in the cluster\n", len(pendingPods))
	return pendingPods, nil
}

// get Pod Events
func (p *podRestarter) getPodEvents(pod, namespace string) ([]string, error) {
	var events []string
	api := p.clientset.CoreV1()

	// get Pod events
	eventsStruct, err := api.Events(namespace).List(
		p.ctx,
		metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})

	if err != nil {
		msg := fmt.Sprintf("Could not go through Pod %s/%s Events: \n%v", namespace, pod, err)
		return events, errors.New(msg)
	}

	for _, item := range eventsStruct.Items {
		events = append(events, item.Message)
	}

	if len(events) == 0 {
		msg := fmt.Sprintf(
			"Pod %s/%s has 0 Events. Probably it does not exist or it does not have any events in the last hour",
			namespace, pod,
		)
		return events, errors.New(msg)
	}
	return events, nil
}

// verify if a Pod exists and is in Pending state
func (p *podRestarter) verifyPendingPodExists(pod, namespace string) (*v1.Pod, error) {
	api := p.clientset.CoreV1()

	podStruct, err := api.Pods(namespace).Get(
		p.ctx,
		pod,
		metav1.GetOptions{},
	)
	if e.IsNotFound(err) {
		msg := fmt.Sprintf("Pod %s/%s does not exist anymore", namespace, pod)
		return nil, errors.New(msg)
	} else if statusError, isStatus := err.(*e.StatusError); isStatus {
		msg := fmt.Sprintf("Error getting pod %s/%s: %v",
			namespace, pod, statusError.ErrStatus.Message)
		return podStruct, errors.New(msg)
	} else if err != nil {
		msg := fmt.Sprintf("Pod %s/%s has a problem: %v", namespace, pod, err)
		return podStruct, errors.New(msg)
	} else {
		if podStruct.Status.Phase == "Pending" {
			p.infoLog.Printf("Pod %s/%s exists and is in a %s state", namespace, pod, podStruct.Status.Phase)
			return podStruct, nil
		}
		msg := fmt.Sprintf(
			"Pod %s/%s exists but is not in a Pending state anymore. Pod state: %s",
			namespace, pod, podStruct.Status.Phase,
		)
		return podStruct, errors.New(msg)
	}
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
	p.infoLog.Printf("DELETED Pod %s/%s", namespace, pod)
	return nil
}

// define variables
var (
	infoLog         = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog        = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	pollingInterval int
	kubeconfig      *string
	ctx             = context.TODO()
	errorMessage    string
	namespace       string
)

func main() {

	// define and parse cli params
	flag.StringVar(&namespace, "namespace", "", "kubernetes namespace")
	flag.IntVar(&pollingInterval, "polling-interval", 10, "number of seconds between iterations")
	flag.StringVar(
		&errorMessage,
		"error-message",
		"container veth name provided (eth0) already exists",
		"number of seconds between iterations",
	)
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	for {

		fmt.Println("\n############## POD-RESTARTER ##############")
		infoLog.Printf("Running every %d seconds", pollingInterval)

		p := &podRestarter{
			errorLog:   errorLog,
			infoLog:    infoLog,
			kubeconfig: kubeconfig,
			ctx:        ctx,
		}

		// authenticate to k8s cluster and initialise k8s client
		clientset, err := p.k8sClient()
		if err != nil {
			errorLog.Println(err)
			os.Exit(1)
		} else {
			p.clientset = clientset
		}

		var pendingPods = make(map[string]string)
		var pendingErroredPods = make(map[string]string)

		pendingPods, err = p.getPendingPods(namespace)
		if err != nil {
			errorLog.Println(err)
			// continue
		} else {
			for pod, ns := range pendingPods {

				// get Pod events
				events, err := p.getPodEvents(pod, ns)
				if err != nil {
					errorLog.Println(err)
				}
				// if error message is in events
				// append Pod to map
				for _, event := range events {
					if strings.Contains(event, errorMessage) {
						infoLog.Printf("Pod %s/%s has error: \n%s", ns, pod, event)
						pendingErroredPods[pod] = ns
						break // break after seeing message only once in the events
					}
				}
			}
			infoLog.Printf(
				"There is a TOTAL of %d/%d Pods in Pending State with error message: %s",
				len(pendingErroredPods), len(pendingPods), errorMessage,
			)
		}
		// // infoLog.Printf("There are %d pending Pods: %+v", len(pendingPods), pendingPods)	// DEBUG
		// // infoLog.Printf("There are %d errored Pods: %+v", len(pendingErroredPods), pendingErroredPods)	// DEBUG

		// time.Sleep(20 * time.Second) //DEBUG allow pods to heal

		// iterate through error Pods map
		for pod, ns := range pendingErroredPods {
			// verify if Pod exists and is still in a Pending state
			// var podData *v1.Pod	//DEBUG
			_, err = p.verifyPendingPodExists(pod, ns)
			// fmt.Printf("\n%+v\n", &podData.ObjectMeta)	//DEBUG
			if err != nil {
				errorLog.Println(err)
			} else {
				// delete Pod
				err := p.deletePod(pod, ns)
				if err != nil {
					errorLog.Println(err)
				}
			}
		}
		time.Sleep(time.Duration(pollingInterval) * time.Second) // sleep for n seconds
	}
}
