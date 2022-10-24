#!/usr/bin/env bash

dest_path="/root/workspace/jhk/cluster-manager/deployments"
password="ketilinux"
ip="10.0.5.24"

scr_deployment_name=$1

cd deployments/
echo scp $scr_deployment_name root@$ip:$dest_path copying...
sshpass -p $password scp $scr_deployment_name root@$ip:$dest_path
