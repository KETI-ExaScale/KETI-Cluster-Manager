#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/cluster-manager-service.yaml
    kubectl delete -f deployments/cluster-manager-service.yaml
else
    echo kubectl apply -f deployments/cluster-manager-service.yaml
    kubectl apply -f deployments/cluster-manager-service.yaml
fi