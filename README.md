# Cluster Manager
**Contents**
- [1.Introduction-of-Cluster-Manager](#1introduction-of-cluster-manager)
- [2.Environment](#2environment)
- [3.Installation](#3installation)
----
## 1.Introduction of Cluster Manager
A module that supports multi-cluster scheduling without limiting access clusters based on a Kubernetes multi-cluster environment.<br>
It helps the GPU Scheduler's cluster scheduling by maintaining the node score of the joined cluster.
#### Main Function
- Select the optimal cluster when requested by the GPU Scheduler
#### Required Module
- *[GPU-Scheduler](https://github.com/KETI-ExaScale/GPU-Scheduler)*
---
## 2.Environment
Module installation should be configured in the following environment<br>
### (1) Pre Installed Modules
Because it works with GPU Scheduler, it operates in the same environment as GPU Scheduler<br>
The operation of Cluster Manager is possible only when the configuration of GPU Scheduler is successful.<br>

The 'gpu' namespace exists in the cluster and
Confirm that the gpu-scheduler module is installed in that namespace as follows<br>
```
[root@master ~]# kubectl get pods -A | grep gpu-scheduler
gpu           gpu-scheduler-66f95d7c8d-b9wwj          1/1     Running     0                      23h
```
### (2) How to configure Multi-cluster
1. copy the cluster config file<br> 
Copy the config files existing in the {~/.kube/config} path of each cluster to the {~/.kube/config} path of the cluster to be joined.
2. Register environment variable<br>
a. Register config file : export KUBECONFIG=\$HOME/.kube/config <br>
b. Join cluster : export KUBECONFIG=\$KUBECONFIG:\$HOME/.kube/\$another_cluster_config_file_name (append by :) <br>
Confirm that the cluster is successfully joined as follows
```
[root@master ~]# kubectl config get-clusters
NAME
keti-gpu-cluster1
keti-gpu-cluster2
```
---
## 3.Installation
### (1) Set Service
Deploy {cluster-manager-service.yaml} so that the module can be accessed from external clusters
```
[creates service yaml]
# kubectl create -f cluster-manager-service.yaml
or
[run service script]
# ./1.service-cluster-manager.sh
```
```
[root@master ~]# kubectl get service -A | grep cluster-manager
gpu           cluster-manager          NodePort       10.96.52.12      <none>        80:31000/TCP             23h
```
### (2) Set Cluster rbac
Deploy {role-binding.yaml / service-account.yaml} for authorization to Cluster Manager module
```
[create rbac yaml]
# kubectl create -f rbac/role-binding.yaml
# kubectl create -f rbac/service-account.yaml
or
[run rbac script]
# ./2.rbac-gpucluster-manager.sh
```
```
[root@master ~]# kubectl get serviceaccount -A | grep cluster-manager
gpu               cluster-manager                      1         23h
```
### (3) Create Cluster Manager
After cluster roll binding and service account deployment, the cluster manager can operate successfully <br>
For cluster scheduling of Cluster Manager, the following module {*[GPU-Scheduler](https://github.com/KETI-ExaScale/GPU-Scheduler)* } must be operating successfully.
```
[create cluster manager yaml]
# kubectl apply -f cluster-manager.yaml
or
[run cluster manager script]
# ./3.create-cluster-manager.sh
```
Finally, normal scheduling is possible when the following 4 modules are in the Running state.
```
[root@master ~]# k get po -n gpu
NAME                                    READY   STATUS    RESTARTS      AGE
gpu-scheduler-66f95d7c8d-b9wwj          1/1     Running   0             23h
influxdb-0                              1/1     Running   0             19d
keti-cluster-manager-57489c8ddc-sqrx2   1/1     Running   0             23h
keti-gpu-device-plugin-lg7jx            2/2     Running   3 (22d ago)   22d
keti-gpu-metric-collector-rtd2s         1/1     Running   44 (8h ago)   19d
```
### (5) Cluster Scheduling
+ When specifying the cluster name in the pod yaml, specify the cluster name in metadata > annotation > clusterName and deploy successfully to the specified cluster.
```
   apiVersion: batch/v1
    kind: Job
    metadata:
    name: kjhtest-nbody-benchmark-mps
    namespace: userpod
    spec:
    template:
        metadata:
          annotations:
            clusterName: keti-gpu-cluster2
        spec:
        hostIPC: true
        schedulerName: gpu-scheduler
        containers:
            - image: seedjeffwan/nbody:cuda-10.1
            name: nbody1
            args:
                - nbody
                - -benchmark
                - -numdevices=1
                - -numbodies=198200
            resources:
                limits:
                keti.com/mpsgpu: 1
                requests:
                cpu: "250m"
                memory: "5000Mi"
            volumeMounts:
                - name: nvidia-mps
                mountPath: /tmp/nvidia-mps 
        volumes:
            - name: nvidia-mps
            hostPath:
                path: /tmp/nvidia-mps
        restartPolicy: Never
```
+ If the cluster name is not specified in the pod yaml, the result is returned after performing cluster scheduling.
```
[root@master ~]# kubectl logs keti-cluster-manager-57489c8ddc-sqrx2 -n gpu
#Request Cluster Scheduling Called
-pod requested gpu : 1
--Cluster Name:  keti-gpu-cluster1
---clusterScore: 54
--Cluster Name:  keti-gpu-cluster2
---clusterScore: 50
#Best Cluster Is:  {keti-gpu-cluster1 54}
```
