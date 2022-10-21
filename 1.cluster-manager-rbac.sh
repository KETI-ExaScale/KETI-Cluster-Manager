#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/rbac/cluster-manager-role-binding.yaml
    echo kubectl delete -f deployments/rbac/cluster-manager-service-account.yaml
    kubectl delete -f deployments/rbac/cluster-manager-role-binding.yaml
    kubectl delete -f deployments/rbac/cluster-manager-service-account.yaml
else
    echo kubectl create -f deployments/rbac/cluster-manager-role-binding.yaml
    echo kubectl create -f deployments/rbac/cluster-manager-service-account.yaml
    kubectl create -f deployments/rbac/cluster-manager-role-binding.yaml
    kubectl create -f deployments/rbac/cluster-manager-service-account.yaml
fi
