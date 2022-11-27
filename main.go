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

	"k8s.io/client-go/util/homedir"
)

// define variables
var (
	infoLog         = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog        = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
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

		var pendingPods []podDetails
		var pendingErroredPods = make(map[string]string)

		pendingPods, err = p.getPendingPods(namespace)
		if err != nil {
			errorLog.Println(err)
		} else {

			// check if Pending Pods have error message
			for _, pod := range pendingPods {

				// skip Pods without owner of Pods that were delted or merked to be deleted
				if pod.hasOwner {
					if pod.deletionTimestamp == nil {
						// get Pod event
						var events []podEvent
						events, err := p.getPodEvents(pod.podName, pod.podNamespace)
						if err != nil {
							errorLog.Println(err)
						}
						// if error message is in events
						// append Pod to map
						for _, event := range events {
							if strings.Contains(event.message, errorMessage) {
								infoLog.Printf("Pod %s/%s has error: \n%s", pod.podNamespace, pod.podName, event.message)
								pendingErroredPods[pod.podName] = pod.podNamespace
								break // break after seeing message only once in the events
							}
						}

					} else {
						errorLog.Printf(
							"Pod has already been deleted/scheduled to be deleted: %s/%s\n%v",
							pod.podNamespace,
							pod.podName,
							pod.deletionTimestamp,
						)
					}
				} else {
					p.errorLog.Printf(
						"Pod does not have owner: %s/%s",
						pod.podNamespace,
						pod.podName,
					)
				}
			}
			infoLog.Printf(
				"There is a TOTAL of %d/%d Pods in Pending State with error message: %s",
				len(pendingErroredPods), len(pendingPods), errorMessage,
			)

		}

		// allow Pending Pods time to self heal
		time.Sleep(healTime * time.Second)

		// iterate through errored Pods map
		for pod, ns := range pendingErroredPods {
			// verify if Pod exists and is still in a Pending state
			var podInfo *podDetails
			podInfo, err = p.getPodDetails(pod, ns)
			if err != nil {
				errorLog.Println(err)
			} else {
				if podInfo.phase == "Pending" {
					infoLog.Printf("Pod still in Pending state: %s/%s", ns, pod)

					// delete Pod
					err := p.deletePod(pod, ns)
					if err != nil {
						errorLog.Println(err)
					}
				} else {
					infoLog.Printf("Pod HAS NEW STATE %s: %s/%s", podInfo.phase, ns, pod)
				}
			}
		}
		time.Sleep(time.Duration(pollingInterval-int(healTime)) * time.Second) // sleep for n seconds
	}
}
