#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/keti-cluster-manager.yaml
    kubectl delete -f deployments/keti-cluster-manager.yaml
else
    echo kubectl apply -f deployments/keti-cluster-manager.yaml
    kubectl apply -f deployments/keti-cluster-manager.yaml
fi