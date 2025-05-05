# AWS Kubernetes setup instructions

## First time

Check that kops version is >= 1.32.0

Check that kubectl version is >= 1.32.0

## Kubernetes cluster setup

Enter head node: (pem key file attached)

```
ssh -i yaas-opensource-team.pem ubuntu@ec2-54-151-42-253.us-west-1.compute.amazonaws.com

cd setup

. ./launch_cluster.sh  <instance_type>  <your_name>
```

eg: . ./launch_cluster.sh m5a.xlarge chirag

AWS instance types are here: <https://aws.amazon.com/ec2/instance-types/>

We use m5a type (you can see different sizes listed under m5a tab)

Your name is used to track whose cluster it is

Above script will prompts you. Press enters, until it asks:

"On entering a number (less than 10), cluster with that many nodes will be created. Else cluster creation will be aborted:"

Enter the number of nodes you want in your cluster

Wait. This will prepare your cluster.

Once the command finishes, you can check the status of cluster using:

`kops validate cluster --wait 10m`

within 10 minutes cluster should be ready and it should say: "Cluster is Ready"

IF IT DOES NOT, LET. KNOW

With this, you will have a kubernetes cluster running. You can check the nodes in your cluster using:
`kubectl get nodes`

Check that the version of kubectl and the server is >= 1.32.0 using:
`kubectl version`

## yaas aws setup

`. ~/setup/vecter/vecter-setup.sh`

## DeathStar2Bench

Now you will have to launch a microservice application on the cluster

On the head node do:

`. ~/setup/vecter/hotelres.sh`

Now the application is deployed. Check the status of the pods using:
`kubectl get pods`
Wait until everyone of them is running.

To check which pod is on which node use:
`. ~/setup/utils/observe.sh`

Change inbound settings
`. ~/setup/utils/edit_securitygroup_inbound_rules.sh`

Label deployments to be autoscaled
`. ~/setup/vecter/label-hotelres.sh`

expose frontend service
`kubectl expose deployment frontend-hotelres --type=LoadBalancer --name=frontend-service-hotelres -n deathstarbench`

setup cloudwatch
`bash ~/setup/vecter/cloudwatch-setup.sh hotelres`

## Autoscaler and Watcher

build and run the autoscaler and watcher
`./hack/hotel-autoscaler-up`

copy watcher logs into a `.txt` file then run
`python ./read-watcher-output.py`

## Load generation

```
ssh -i yaas-opensource-team.pem ubuntu@ec2-54-193-191-84.us-west-1.compute.amazonaws.com

cd ~/DeathStarBench/hotelReservation

## Launch the load: # REPLACE the IP:PORT with ip:port you got while exposing your application
## This launches a load of 200 requests per second for 60 seconds (using 8 threads and 50 connections.)
## You can increase the load, but you will also have to increase the number of connections.
../wrk2/wrk -D exp -t 8 -c 50 -d 60 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://<IP>:<PORT> -R 200
```

As you send the load, check the CPU usuage of your pods actually go up using:
`kubectl top pods ## Do it on your head node (not the loadgen node)`

## Teardown

```
cd setup

. ./load_env_variables.sh

. ./destroy_all_clusters.sh
## Press y
## Choose your cluster to destroy
## Wait
## Press n everywhere else

Check: kubectl get nodes
## It should not show any nodes
```
