# podoscaler

## prereqs
- docker
- minikube
- kubectl

## setup
1. clone repo
2. `minikube start` (with docker)

## build and deploy dummy app (testapp) to minikube cluster
1. might need to run `eval $(minikube -p minikube docker-env)` to enter minikube's docker env for the build
2. `minikube image build -t testapp-img ./testapp`
or `docker image build -t testapp-img ./testapp`
3. `minikube image ls` to see if it built
4. `kubectl apply -f ./testapp/deployment.yaml` to launch
5. check deployment with `kubectl get deployments` and `kubectl get pods`
6. Delete deployment with `kubectl delete deployment manuscaler` (if you're done)

## build and deploy manuscaler to minikube cluster
1. might need to run `eval $(minikube -p minikube docker-env)` to enter minikube's docker env for the build
2. `minikube image build -t manuscaler-img ./manuscaler`
or `docker image build -t manuscaler-img ./manuscaler`
3. `minikube image ls` to see if it built
4. `kubectl apply -f ./manuscaler/rbac.yaml` to apply roles and service accounts and stuff (for permissions)
5. `kubectl apply -f ./manuscaler/deployment.yaml` to launch
6. check deployment with `kubectl get deployments` and `kubectl get pods`
7. `kubectl expose deployment/manuscaler --type="NodePort" --port [PORT]` to open a port
8. `kubectl port-forward svc/manuscaler [PORT]` (needed for windows and mac i think??)
9. test `localhost:[PORT]/`
10. Delete deployment with `kubectl delete deployment manuscaler` (if you're done)

## horizontially scaling testapp with manuscaler
make a REST API call to
`localhost:[PORT]/hscale`
with a json body
```
{
    "deploymentnamespace": "default",
    "deploymentname": "testapp",
    "replicas": [DESIRED REPLICAS]
}
```