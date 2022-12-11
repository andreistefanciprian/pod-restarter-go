### Description

Build a tool that restarts Pods in a bad state using client-go kubernetes packages.

Performs the following steps:
* Looks for latest Pod Events that matches Event Reason (eg: reason "FailedCreatePodSandBox")
* If there are matching Pods, these Pods will go through a sequence of steps before they get deleted:
** verify Pod exists
** verify Pod has owner/controller
** verify Pod has not been scheduled to be deleted
* If all above checks pass, Pod will be deleted

These steps are repeated in a loop on a polling interval basis.


### Run script on local machine/laptop

```
# install dependencies
go mod init
go mod tidy

# runs in dry-run mode
go run main.go --dry-run

# delete Pods that have Events with Reason "FailedCreatePodSandBox"
go run main.go

# delete Pods that have Events with Reason "BackOff"
go run main.go --reason "BackOff"

# delete Pods that have Events with Reason "BackOff" in namespace default
go run main.go --reason="BackOff" --namespace default

# delete Pods that have Events with Reason "FailedCreatePodSandBox" every 30 seconds
go run main.go --reason="BackOff" --namespace default --polling-interval 30
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

### Test 

```
# generate Pending pods
cd infra/tests
bash generate_pending_pods.sh

# check app logs
kubectl logs -l app=pod-restarter -f
```