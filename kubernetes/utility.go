package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// PodChecks returns nil if Pod
// 1. exists
// 2. has Owner
// 3. has not been scheduled to be deleted
// 4. and is not in a Healthy state (eg: Pending, Failed or Running with unhealthy containers)
func (c *kubeClient) PodChecks(ctx context.Context, podName, podNamespace string) error {
	// verify if Pod exists
	podInfo, err := c.GetPodDetails(ctx, podName, podNamespace)
	if err != nil {
		return err
	}

	// verify Pod has owner
	err = podInfo.verifyPodHasOwner()
	if err != nil {
		return err
	}

	// verify Pod is scheduled to be deleted
	err = podInfo.verifyPodScheduledToBeDeleted()
	if err != nil {
		return err
	}

	// verify Pod is in an Unhealthy state
	err = podInfo.verifyPodStatus()
	if err != nil {
		return nil
	} else {
		msg := fmt.Sprintf("Pod is in a Healthy State: %s/%s", podNamespace, podName)
		return errors.New(msg)
	}
}

// verifyPodStatus returns error if Pod is in a Pending, Failed or Running (with unhealthy containers) state
func (p *PodDetails) verifyPodStatus() error {

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
					"Pod is in a %s state and has issues: %s/%s",
					p.Phase, p.PodNamespace, p.PodName,
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

// verifyPodHasOwner returns nil if Pod has owner
func (p *PodDetails) verifyPodHasOwner() error {
	if len(p.OwnerReferences) > 0 {
		return nil
	}
	msg := fmt.Sprintf(
		"Pod does not have owner/controller: %s/%s",
		p.PodNamespace, p.PodName,
	)
	return errors.New(msg)
}

// verifyPodScheduledToBeDeleted returns nil if Pod is not scheduled to be deleted
func (p *PodDetails) verifyPodScheduledToBeDeleted() error {
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

// getUniqueListOfPods returns a unique list of Pods that have Events that match Reason
func getUniqueListOfPods(events []PodEvent) map[string]string {

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

// removeOlderEvents returns a slice of latest Events not older than eventMaxAge
func removeOlderEvents(events []PodEvent, eventMaxAge time.Time) []PodEvent {
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
