#!/usr/bin/env bash
dest_path="/root/workspace/deployments/cluster-manager"
password="ketilinux"
ip="10.0.5.62"

#$1 deployment/d or " "

if [ "$1" == "deployment" ] || [ "$1" == "d" ]; then  
    echo scp -r deployments root@$ip:$dest_path copying...
    sshpass -p $password scp -r deployments root@$ip:$dest_path
else
    echo scp ./deployments/keti-cluster-manager.yaml root@$ip:$dest_path/deployments copying...
    sshpass -p $password scp ./deployments/keti-cluster-manager.yaml root@$ip:$dest_path/deployments
fi