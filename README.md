# podoscaler

## prereqs

- [docker](https://www.docker.com/)
- [minikube](https://minikube.sigs.k8s.io/docs/start/?arch=%2Fwindows%2Fx86-64%2Fstable%2F.exe+download)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## setup

1. clone repo
2. `./hack/minikube-up.sh`

## install linkerd and ingress

1. install linkerd cli using brew or look at instructions `https://linkerd.io/2-edge/getting-started/`
2. install linkerd crds `linkerd install --crds | kubectl apply -f -`
3. install linkerd `linkerd install | kubectl apply -f -`
4. make sure ingress addon is enabled `minikube addons enable ingress`
5. install linkerd on cluster metrics `linkerd viz install | kubectl apply -f -`
6. access linkerd dashboard `linkerd viz dashboard`
7. get prometheus url in linkerd-viz namespace
8. setup nginx-ingress-controller to auto inject `linkerd inject <(kubectl get deploy -n ingress-nginx ingress-nginx-controller -o yaml) | kubectl apply -f -`

## build and deploy dummy app (testapp) to minikube cluster

1. `./hack/testapp-up`

to check deployment:

2. `kubectl get deployments`
3. `kubectl get pods`

to delete deployment:

4. `./hack/testapp-down`

## build and deploy manuscaler to minikube cluster

run `./hack/manuscaler-up`

this will open a port to communicate to the app with

to check deployment, (from another terminal):

2. `kubectl get deployments`
3. `kubectl get pods`
4. make a request to `localhost:3001/`

to delete deployment:

5. `./hack/manuscaler-down`

## horizontially scaling testapp with manuscaler

make a REST API call to
`localhost:3001/hscale`
with a json body

```
{
    "deploymentnamespace": "default",
    "deploymentname": "testapp",
    "replicas": [DESIRED REPLICAS]
}
```

check pods with `kubectl get pods`

## vertically scaling testapp with manuscaler

choose a pod from `kubectl get pods`
then make a REST API call to
`localhost:3001/vscale`
with a json body

```
{
    "podnamespace": "default",
    "podname": "[POD NAME]",
    "containername": "testapp-container",
    "cpurequests": "900m",
    "cpulimits": "900m"
}
```

(or another value instead of 900m; 900m means 90% of a CPU, use "1", "2",...for allocating 1, 2,... full cpus)

check status of pod with `kubectl get pod [POD NAME] --output=yaml`

## build and deploy autoscaler to minikube cluster

run `./hack/autoscaler-up`

to check deployment, (from another terminal):

2. `kubectl get deployments`
3. `kubectl get pods`

to delete deployment:

4. `./hack/autoscaler-down`

## send load to testapp

```
kubectl expose deployment/testapp --type="NodePort" --port 3000
kubectl port-forward svc/testapp 3000
kubectl run -i --tty load-generator --rm --image=busybox:1.28 --restart=Never -- /bin/sh -c "while sleep 0.01; do wget -q -O- http://testapp:3000/; done"
```

