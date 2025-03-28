# podoscaler

## prereqs
- docker
- minikube
- kubectl

## build and deploy to minikube cluster
1. clone this repo and navigate into it
2. `minikube start` (with docker)
3. might need to run `eval $(minikube -p minikube docker-env)` to enter minikube's docker env for the build
4. `minikube image build -t podoscaler-img .`
5. `minikube image ls` to see if it built
6. `kubectl apply -f deployment.yaml` to launch
7. check deployment with `kubectl get deployments` and `kubectl get pods`
8. `kubectl expose deployment/podoscaler --type="NodePort" --port [PORT]` to open a port
9. `kubectl port-forward svc/podoscaler [PORT]` (needed for windows and mac i think??)
10. Interact with the service via `localhost:[PORT]`