###### Tasks information

## Tools/Technologies

1. Docker
2. Minikube
3. Helm
4. Grafana API

## Approaches

The first approach will be to create an init container which will run as part of the Grafana pod initialization.
This approach is good in case we won't need to re-run/retry the users creation process.
In this approach, users creation will be a blocker for Grafana start up - If the init container fails the whole pod fails.

The second approach will be creating a k8s job which will have an helm hook and will run after the Grafana pod is started, But still as part of the Grafana installation.
This approach is good in case we want to be able to retry and rerun user creation without restarting the whole Grafana pod.
Also it won't run on each pod restart so we can have our users created once on helm installation and be done.

We will be using the second approach.

## Handling sensitive information

We would handle information such as user names, passwords and roles in

###### Installation manual

## Prerequisites

1. Docker
2. Minikube
3. kubectl
4. Helm

## Provision k8s cluster & test cluster

minikube start
kubectl cluster-info

## add grafana helm repo

helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

## use grafana namespace

kubectl apply -f namespace.yaml
kubectl config set-context --current --namespace=grafana

## deploy grafana admin secret

kubectl apply -f grafana-admin-secret.yaml

## create users configmap

kubectl apply -f users-list.yaml

## deploy and verify grafana

helm install grafana grafana/grafana --namespace grafana -f values.yaml
kubectl get pods
kubectl port-forward svc/grafana 3000:80
