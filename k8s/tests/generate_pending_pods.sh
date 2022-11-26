# This script generates Pods in Pending state due to failure to pull image

for (( c=1; c<=500; c++ ))
do
    echo -e "\n\nThis is iteration number $c"
    # for ns in `kubectl get namespaces --no-headers | awk '{print $1}'`; do
    for ns in test1 test2; do
        echo Create namespace $ns
        kubectl create ns $ns --dry-run=client -o yaml | kubectl apply -f -
        
        # vars
        random_number=$(( $RANDOM % 5000 + 1 ))
        no_controller_pod="pod-no-controller-${random_number}"
        transition2running_pod="pod-running-${random_number}"
        deleted_pod="pod-deleted-${random_number}"

        echo Create bad deployment/daemonset
        kubectl --namespace $ns delete deployment busybox3-dep
        kubectl --namespace $ns delete daemonset busybox3-ds
        kubectl --namespace $ns apply -f not_ok_controller_resources.yaml

        echo Create not owned Pending pods
        kubectl --namespace $ns run $no_controller_pod --image=wrongimage
        kubectl --namespace $ns run $transition2running_pod --image=wrongimage
        kubectl --namespace $ns run $deleted_pod --image=wrongimage

        sleep 15

        #  test 4 
        echo Transition controller owned Pods to Running
        for i in `kubectl --namespace $ns get pods | grep busybox3 | awk '{print $1}'`; do kubectl --namespace $ns set image pod/$i busybox=busybox; done
        # test 1
        echo Transition not owned Pod from Pending to Running
        kubectl --namespace $ns set image pod/$transition2running_pod $transition2running_pod=nginx

        # test 2
        echo Delete not owned Pending Pod
        kubectl --namespace $ns delete pod $deleted_pod --now

        # test 3
        echo  Delete controller owned Pods
        kubectl --namespace $ns delete deployment busybox2-dep
        kubectl --namespace $ns delete daemonset busybox2-ds

        sleep 15
    done

    # echo Non Running State Pods
    # kubectl get pods --field-selector status.phase!=Running --all-namespaces

    echo
    kubectl get pods --field-selector status.phase=Pending --all-namespaces
    echo Pods in Pending State: $(kubectl get pods --field-selector status.phase=Pending --all-namespaces --no-headers | wc -l)
    kubectl get pods

    # delete all Pending Pods every five iterations
    if [ $(expr $c % 3) == "0" ]; then
        echo Deleting all Pending pods across namespace $ns
        kubectl delete namespace $ns
        kubectl create namespace $ns
        for i in `kubectl --namespace $ns get pods --field-selector status.phase=Pending --no-headers | awk '{print $1}'`; do kubectl --namespace $ns delete pod $i --now; done
    fi
done
