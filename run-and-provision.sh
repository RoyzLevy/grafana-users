#!/bin/bash

# Provision k8s cluster
minikube start

# add grafana helm repo
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# use grafana namespace
kubectl apply -f namespace.yaml
kubectl config set-context --current --namespace=grafana

# deploy grafana admin secret
kubectl apply -f grafana-admin-secret.yaml

# create users configmap
kubectl apply -f users-list.yaml

# create docker image and load to minikube
docker build -t grafana-users-provision:1.0.0 .
minikube image load grafana-users-provision:1.0.0

# deploy and verify grafana
helm install grafana grafana/grafana --namespace grafana -f values.yaml
sleep 100
kubectl port-forward svc/grafana 3000:80