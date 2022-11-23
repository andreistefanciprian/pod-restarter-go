# This script generates Pods in Pending state due to failure to pull image

for ns in `kubectl get namespaces --no-headers | awk '{print $1}'`; do
# for ns in test; do
    random_number=$(( $RANDOM % 5000 + 1 ))

    # create Pending pods
    kubectl --namespace $ns run pod-$random_number --image=wrongimage
    kubectl --namespace $ns run transition2running-$random_number --image=wrongimage
    kubectl --namespace $ns run del-$random_number --image=wrongimage

    sleep 20

    # test 1
    # transition Pod from Pending to Running
    kubectl set image pod/transition2running-$random_number transition2running-$random_number=nginx

    # test 2
    # delete Pending Pod
    kubectl --namespace $ns delete pod del-$random_number --now
done

# echo Non Running State Pods
# kubectl get pods --field-selector status.phase!=Running --all-namespaces

echo
kubectl get pods --field-selector status.phase=Pending --all-namespaces
echo Pods in Pending State: $(kubectl get pods --field-selector status.phase=Pending --all-namespaces --no-headers | wc -l)
