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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	k8s "github.com/andreistefanciprian/pod-restarter-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

// define variables
var (
	pollingInterval int
	kubeconfig      *string
	ctx             = context.TODO()
	errorMessage    string
	namespace       string
	healTime        time.Duration = 5 // allow Pending Pod time to self heal (seconds)
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
		log.Printf("Running every %d seconds", pollingInterval)

		p := &k8s.PodRestarter{
			Logger:     log.Default(),
			Kubeconfig: kubeconfig,
			Ctx:        ctx,
		}

		// authenticate to k8s cluster and initialise k8s client
		clientset, err := p.K8sClient()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		} else {
			p.Clientset = clientset
		}

		var pendingPods []k8s.PodDetails
		var pendingErroredPods = make(map[string]string)

		pendingPods, err = p.GetPendingPods(namespace)
		if err != nil {
			log.Println(err)
		} else {

			// check if Pending Pods have error message
			for _, pod := range pendingPods {

				// skip Pods without owner of Pods that were delted or merked to be deleted
				if pod.HasOwner {
					if pod.DeletionTimestamp == nil {
						// get Pod event
						var events []k8s.PodEvent
						events, err := p.GetPodEvents(pod.PodName, pod.PodNamespace)
						if err != nil {
							log.Println(err)
						}
						// if error message is in events
						// append Pod to map
						for _, event := range events {
							if strings.Contains(event.Message, errorMessage) {
								log.Printf("Pod %s/%s has error: \n%s", pod.PodNamespace, pod.PodName, event.Message)
								pendingErroredPods[pod.PodName] = pod.PodNamespace
								break // break after seeing message only once in the events
							}
						}

					} else {
						log.Printf(
							"Pod has already been deleted/scheduled to be deleted: %s/%s\n%v",
							pod.PodNamespace,
							pod.PodName,
							pod.DeletionTimestamp,
						)
					}
				} else {
					log.Printf(
						"Pod does not have owner: %s/%s",
						pod.PodNamespace,
						pod.PodName,
					)
				}
			}
			log.Printf(
				"There is a TOTAL of %d/%d Pods in Pending State with error message: %s",
				len(pendingErroredPods), len(pendingPods), errorMessage,
			)

		}

		// allow Pending Pods time to self heal
		time.Sleep(healTime * time.Second)

		// iterate through errored Pods map
		for pod, ns := range pendingErroredPods {
			// verify if Pod exists and is still in a Pending state
			var podInfo *k8s.PodDetails
			podInfo, err = p.GetPodDetails(pod, ns)
			if err != nil {
				log.Println(err)
			} else {
				if podInfo.Phase == "Pending" {
					log.Printf("Pod still in Pending state: %s/%s", ns, pod)

					// delete Pod
					err := p.DeletePod(pod, ns)
					if err != nil {
						log.Println(err)
					}
				} else {
					log.Printf("Pod HAS NEW STATE %s: %s/%s", podInfo.Phase, ns, pod)
				}
			}
		}
		time.Sleep(time.Duration(pollingInterval-int(healTime)) * time.Second) // sleep for n seconds
	}
}
