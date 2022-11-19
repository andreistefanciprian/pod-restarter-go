package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

	// define variables
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	errorMessage := "Back-off pulling image"
	// errorMessage := "container veth name provided (eth0) already exists"
	var pollingInterval time.Duration = 10
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

		pendingPods := make(map[string]string)
		pendingErroredPods := make(map[string]string)
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

		api := clientset.CoreV1()

		// list all Pods in Pending state
		pods, err := api.Pods(*namespace).List(
			ctx,
			v1.ListOptions{
				TypeMeta: v1.TypeMeta{Kind: "Pod"},
				// Status:        v1.Status{Status: "Pending"},
				FieldSelector: "status.phase=Pending",
			},
		)
		if err != nil {
			panic(err.Error())
		}
		infoLog.Printf("There are %d pods in Pending state in the cluster\n", len(pods.Items))

		// marshal_struct, _ := json.Marshal(pods)
		// _ = ioutil.WriteFile("pods.json", marshal_struct, 0644)
		// os.Exit(1)

		// get a map of all Pending Pods
		for _, pod := range pods.Items {
			pendingPods[pod.ObjectMeta.Name] = pod.ObjectMeta.Namespace
			// infoLog.Printf("Pod %s/%s is in Pending state", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			// if pod.Status.Phase == "Pending" {
			// 	pendingPods[pod.ObjectMeta.Name] = pod.ObjectMeta.Namespace
			// }
			// for k, _ := range pod {
			// 	fmt.Println(k)
			// }
		}

		// for each name/pod
		for pod, namespace := range pendingPods {
			infoLog.Printf("Pod %s/%s is in a Pending state", namespace, pod)

			// get Pod events
			events, _ := api.Events(namespace).List(
				ctx,
				v1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod),
					TypeMeta:      v1.TypeMeta{Kind: "Pod"},
				})

			// if error message is in events
			// append Pod to a map of all Pending Pods with error message
			for _, item := range events.Items {
				if strings.Contains(item.Message, errorMessage) {
					infoLog.Printf("Pod %s/%s has error: %s", namespace, pod, item.Message)
					pendingErroredPods[pod] = namespace
					break // break after seeing message only once in the events
				}
			}
		}

		infoLog.Printf("There are %d/%d Pods in Pending State with error message: %s", len(pendingErroredPods), len(pendingPods), errorMessage)

		// delete Pending Pods with error message
		for pod, namespace := range pendingErroredPods {
			infoLog.Printf("Deleting Pod %s/%s ...", namespace, pod)
			err := api.Pods(namespace).Delete(ctx, pod, v1.DeleteOptions{})
			if err != nil {
				errorLog.Printf("Probably Pod %s/%s does not exist anymore.", namespace, pod)
			}
		}

		time.Sleep(pollingInterval * time.Second) // sleep for n seconds
		fmt.Println()
		// break
	}

}
