# podoscaler

## prereqs
- [docker](https://www.docker.com/)
- [minikube](https://minikube.sigs.k8s.io/docs/start/?arch=%2Fwindows%2Fx86-64%2Fstable%2F.exe+download)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## setup
1. clone repo
2. `minikube start --feature-gates=InPlacePodVerticalScaling=true` [(with docker)](https://minikube.sigs.k8s.io/docs/drivers/docker/)

## build and deploy dummy app (testapp) to minikube cluster
1. might need to run `eval $(minikube -p minikube docker-env)` to enter minikube's docker env for the build
2. `minikube image build -t testapp-img ./testapp` **or** `docker image build -t testapp-img ./testapp`
3. `minikube image ls` to see if it built
4. `kubectl apply -f ./testapp/deployment.yaml` to launch
5. check deployment with `kubectl get deployments` and `kubectl get pods`
6. delete deployment with `kubectl delete deployment manuscaler` (if you're done)

## build and deploy manuscaler to minikube cluster
1. might need to run `eval $(minikube -p minikube docker-env)` to enter minikube's docker env for the build
2. `minikube image build -t manuscaler-img ./manuscaler` **or** `docker image build -t manuscaler-img ./manuscaler`
3. `minikube image ls` to see if it built
4. `kubectl apply -f ./manuscaler/rbac.yaml` to apply roles and service accounts and stuff (for permissions)
5. `kubectl apply -f ./manuscaler/deployment.yaml` to launch
6. check deployment with `kubectl get deployments` and `kubectl get pods`
7. `kubectl expose deployment/manuscaler --type="NodePort" --port 3001` to open a port
8. `kubectl port-forward svc/manuscaler 3001` (needed for windows and mac i think)
9. test `localhost:3001/` - should return "hello"
10. delete deployment with `kubectl delete deployment manuscaler` (if you're done)

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
