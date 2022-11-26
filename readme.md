### Description

Build a pod restarter with k8s client-go sdk.

* Requirements:
    * helm
    * taskfile
    * kubectl
    * go


### Run script on local machine/laptop

```
# install dependencies
go mod init
go mod tidy

# delete Pending Pods that have default error message in all namespaces, every 10 seconds
go run main.go

# delete Pending Pods that have this error message in all namespaces
go run main.go --error-message="Back-off pulling image"

# delete Pending Pods that have this error message in namespace default
go run main.go --error-message="Back-off pulling image" --namespace default

# delete Pending Pods that have this error message in namespace default, every 30 seconds
go run main.go --error-message="Back-off pulling image" --namespace default --polling-interval 30
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
cd k8s/tests
bash generate_pending_pods.sh

# check app logs
kubectl logs -l app=pod-restarter -f
```

### TBD
- verify namespace exists ?
