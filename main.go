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

	e "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type podRestarter struct {
	errorLog           *log.Logger
	infoLog            *log.Logger
	errorMessage       string
	pollingInterval    time.Duration
	kubeconfig         *string
	ctx                context.Context
	clientset          *kubernetes.Clientset
	namespace          string
	pendingPods        map[string]string
	pendingErroredPods map[string]string
}

func (p *podRestarter) getPendingPods() error {
	api := p.clientset.CoreV1()

	// list all Pods in Pending state
	pods, err := api.Pods(p.namespace).List(
		p.ctx,
		v1.ListOptions{
			TypeMeta:      v1.TypeMeta{Kind: "Pod"},
			FieldSelector: "status.phase=Pending",
		},
	)
	if err != nil {
		// panic(err.Error())
		msg := fmt.Sprintf("Could not get a list of Pending Pods: \n%v", err)
		return errors.New(msg)
	}

	p.pendingPods = make(map[string]string)
	for _, pod := range pods.Items {
		p.infoLog.Printf("Pod %s/%s is in Pending state", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		p.pendingPods[pod.ObjectMeta.Name] = pod.ObjectMeta.Namespace
	}
	p.infoLog.Printf("There are a total of %d Pods in Pending state in the cluster\n", len(p.pendingPods))
	return nil
}

func (p *podRestarter) getErroredPendingPods() {
	api := p.clientset.CoreV1()
	p.pendingErroredPods = make(map[string]string)
	// for each name/pod
	for pod, namespace := range p.pendingPods {

		// get Pod events
		events, _ := api.Events(namespace).List(
			p.ctx,
			v1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod),
				TypeMeta:      v1.TypeMeta{Kind: "Pod"},
			})

		// if error message is in events
		// append Pod to map
		for _, item := range events.Items {
			if strings.Contains(item.Message, p.errorMessage) {
				p.infoLog.Printf("Pod %s/%s has error: %s", namespace, pod, item.Message)
				p.pendingErroredPods[pod] = namespace
				break // break after seeing message only once in the events
			}
		}
	}

	p.infoLog.Printf("There are a total of %d/%d Pods in Pending State with error message: %s", len(p.pendingErroredPods), len(p.pendingPods), p.errorMessage)
}

// verify if a Pod exists and is in Pending state
func (p *podRestarter) verifyPendingPodExists(pod, namespace string) (error, bool) {
	api := p.clientset.CoreV1()
	podStruct, err := api.Pods(namespace).Get(
		p.ctx,
		pod,
		v1.GetOptions{},
	)
	if e.IsNotFound(err) {
		msg := fmt.Sprintf("Pod %s/%s does not exist anymore", namespace, pod)
		return errors.New(msg), false
	} else if statusError, isStatus := err.(*e.StatusError); isStatus {
		msg := fmt.Sprintf("Error getting pod %s/%s: %v",
			namespace, pod, statusError.ErrStatus.Message)
		return errors.New(msg), false
	} else if err != nil {
		msg := fmt.Sprintf("Pod %s/%s has a problem: %v", namespace, pod, err)
		return errors.New(msg), false
	} else {
		if podStruct.Status.Phase == "Pending" {
			p.infoLog.Printf("Pod %s/%s exists and is in a %s state", namespace, pod, podStruct.Status.Phase)
			return nil, true
		} else {
			msg := fmt.Sprintf("Pod %s/%s exists but is not in a Pending state anymore. Pod state: %s", namespace, pod, podStruct.Status.Phase)
			// fmt.Printf("%+v", podStruct)	// DEBUG
			return errors.New(msg), false
		}

	}

}

// deletes a Pod
func (p *podRestarter) deletePod(pod, namespace string) error {
	api := p.clientset.CoreV1()

	err := api.Pods(namespace).Delete(
		p.ctx,
		pod,
		v1.DeleteOptions{},
	)
	if err != nil {
		msg := fmt.Sprintf("For some reason Pod %s/%s could not be deleted: %v", namespace, pod, err)
		return errors.New(msg)
	} else {
		p.infoLog.Printf("Deleted Pod %s/%s", namespace, pod)
		return nil
	}
}

func main() {

	// define variables
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	// errorMessage := "Failed to pull image"
	errorMessage := "Back-off pulling image"
	// errorMessage := "container veth name provided (eth0) already exists"
	var pollingInterval time.Duration = 5
	var kubeconfig *string
	ctx := context.TODO()

	// define cli params
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	namespace := flag.String("namespace", "", "kubernetes namespace")
	// pollingInterval := flag.DurationVar("polling-interval", 10, "polling interval")
	flag.Parse()

	for {
		time.Sleep(pollingInterval * time.Second) // sleep for n seconds
		infoLog.Printf("Running every %d seconds", pollingInterval)

		// read and parse kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig) // creates the out-cluster config
		// config, err := rest.InClusterConfig()                          // creates the in-cluster config
		if err != nil {
			// panic(err.Error())
			errorLog.Printf("The kubeconfig cannot be loaded: %v\n", err)
			os.Exit(1)
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			// panic(err.Error())
			errorLog.Printf("The clientset cannot be created: %v\n", err)
			os.Exit(1)
		}

		p := &podRestarter{
			errorLog:        errorLog,
			infoLog:         infoLog,
			errorMessage:    errorMessage,
			pollingInterval: 10,
			kubeconfig:      kubeconfig,
			ctx:             ctx,
			namespace:       *namespace,
			clientset:       clientset,
		}

		err = p.getPendingPods()
		if err != nil {
			errorLog.Println(err)
			continue
		}
		p.getErroredPendingPods()
		// infoLog.Printf("There are %d pending Pods: %+v", len(p.pendingPods), p.pendingPods)	// DEBUG
		// infoLog.Printf("There are %d errored Pods: %+v", len(p.pendingErroredPods), p.pendingErroredPods)	// DEBUG

		// iterate through error Pods map
		for pod, namespace := range p.pendingErroredPods {
			// verify if Pod exists and is still in a Pending state
			err, _ = p.verifyPendingPodExists(pod, namespace)
			if err != nil {
				p.errorLog.Println(err)
			} else {
				// delete Pod
				err := p.deletePod(pod, namespace)
				if err != nil {
					p.errorLog.Println(err)
				}
			}
		}

		fmt.Println("\n\n\n") // DEBUG
		// os.Exit(0)	// DEBUG
		// break	// DEBUG
	}

}
