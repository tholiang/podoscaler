# podoscaler

## prereqs

- [docker](https://www.docker.com/)
- [minikube](https://minikube.sigs.k8s.io/docs/start/?arch=%2Fwindows%2Fx86-64%2Fstable%2F.exe+download)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## local dev setup

1. clone repo
2. `./hack/minikube-up.sh`

## build and deploy autoscaler to minikube cluster

run `./hack/minikube-autoscaler-up`

to check deployment, (from another terminal):

2. `kubectl get deployments`
3. `kubectl get pods`

to delete deployment:

4. `./hack/autoscaler-down`