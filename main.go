// This script runs in/out a K8s cluster
// Deletes Pods that are in a Failing state due to a particular Event Reason (eg: FailedCreatePodSandBox)

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
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
	eventReason     string
	namespace       string
	dryRunMode      bool
	healTime        time.Duration = 5 // allow Pending Pod time to self heal (seconds)
)

func initFlags() {
	// define and parse cli params
	flag.BoolVar(&dryRunMode, "dry-run", false, "enable dry run mode (no changes are made, only logged)")
	flag.StringVar(&namespace, "namespace", "", "kubernetes namespace")
	flag.StringVar(&eventReason, "reason", "FailedCreatePodSandBox", "restart Pods that match Event Reason")
	flag.IntVar(&pollingInterval, "polling-interval", 30, "number of seconds between iterations")
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
}

func main() {

	// parse CLI params
	initFlags()
	flag.Parse()

	// we use this counter in first iteration where we look at all Events in the cluster
	// if counter > 0 we filter out events older than polling interval
	counter := 0

	for {
		log.Printf("Running every %d seconds", pollingInterval)

		c := &k8s.KubeClient{
			Logger:     log.Default(),
			Kubeconfig: kubeconfig,
		}

		// authenticate to k8s cluster and initialise k8s client
		clientset, err := c.NewClientSet()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		} else {
			c.Clientset = clientset
		}

		// generate a unique list of Pods that match Event Reason
		// we do this because a Pod might have multiple Events with the same Reason
		uniquePodList, err := c.GenerateToBeDeletedPodList(ctx, namespace, eventReason, errorMessage, counter, pollingInterval)
		if err != nil {
			log.Println(err)
		}

		// allow Pending Pods a few seconds to self heal
		time.Sleep(healTime * time.Second)

		// iterate through the list of Pods that match Event Reason
		for pod, ns := range uniquePodList {

			err = c.PodChecks(ctx, pod, ns)
			if err != nil {
				log.Println(err)
				continue
			}

			if dryRunMode {
				log.Printf("[DRY-RUN]: Would have deleted Pod: %s/%s", ns, pod)
				continue
			}
			// delete Pod
			err := c.DeletePod(ctx, pod, ns)
			if err != nil {
				log.Println(err)
			}

		}
		time.Sleep(time.Duration(pollingInterval-int(healTime)) * time.Second) // sleep for n seconds
		counter += 1
	}
}
