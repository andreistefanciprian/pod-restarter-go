### Description

[![ci](https://github.com/andreistefanciprian/pod-restarter-go/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/andreistefanciprian/pod-restarter-go/actions/workflows/ci.yaml)

Build a tool that restarts Pods in a bad state using client-go kubernetes packages.

Performs the following steps:
* Looks for latest Pod Events that matches Event Reason (default Reason "FailedCreatePodSandBox") and Event Message (default Message "container veth name provided (eth0) already exists")
* If there are matching Pods, these Pods will go through a sequence of steps before they get deleted:
    - verify Pod exists
    - verify Pod has owner/controller
    - verify Pod has not been scheduled to be deleted
    - verify Pod is in a Failing State (Pending/Failed or Running with failing containers)
* If all above checks pass, Pod will be deleted

These steps are repeated in a loop on a polling interval basis.


### Run script on local machine/laptop

```
# install dependencies
go mod init
go mod tidy

# runs in dry-run mode
go run main.go --dry-run

# delete Pods that have Events with default Reason "FailedCreatePodSandBox" and default Message "container veth name provided (eth0) already exists"
go run main.go

# delete Pods that have Events with Reason "BackOff" and Message "Back-off pulling image"
go run main.go --reason "BackOff" --error-message "Back-off pulling image"

# delete Pods that have Events with Reason "BackOff" and Message "Back-off pulling image" in namespace default
go run main.go --reason="BackOff" --error-message "Back-off pulling image" --namespace default

# delete Pods that have matching Events with default Reason and Message every 30 seconds
go run main.go --polling-interval 30
```

### Deploy to k8s with helm

```
# build container image
task build

# deploy helm chart
task install

# verify helm release
helm list -A

# uninstall helm chart
task uninstall
```

### Run Unit Tests 

```
go test -v ./...
```

### Simulate Failing Pods and test 

```
# generate failing pods due to pulling wrong images
cd infra/tests
bash generate_pending_pods.sh


# check app logs
kubectl logs -l app=pod-restarter -f
```