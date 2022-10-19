# Cluster Manager
**Contents**
- [1.Cluster-Manager 소개](#introduction-of-GPU-Scheduler)
- [2.구성환경](#environment)
- [3.설치순서](#install-step)
----
## 1.Cluster-Manager 소개 
쿠버네티스 멀티 클러스터 환경 기반 접속 클러스터 제한 없는 클러스터간 스케줄링을 지원하는 모듈<br>
조인 클러스터의 노드 점수를 유지하여 GPU Scheduler의 클러스터 스케줄링을 돕는다.
#### 제공기능
- GPU Scheduler 요청 시 최적 클러스터 선정
#### 필요모듈
- *[GPU-Scheduler](https://github.com/KETI-ExaScale/GPU-Scheduler)*
---
## 2.구성환경
모듈 설치는 다음과 같은 환경에서 구성되어야 한다.<br>
### (1) 사전 설치 모듈
GPU Scheduler와 함께 동작하기 때문에 GPU Scheduler와 동일 환경에서 동작하며<br>
GPU Scheduler의 구성이 정상적으로 이루어져야 Cluster Manager의 동작이 가능하다.<br>

클러스터 내에 'gpu' 네임스페이스가 존재하며 해당 네임스페이스에 gpu-scheduler 모듈이 설치되어 있음을 다음과 같이 확인<br>
```
[root@master ~]# kubectl get pods -A | grep gpu-scheduler
gpu           gpu-scheduler-66f95d7c8d-b9wwj          1/1     Running     0                      23h
```
### (2) 멀티 클러스터 구성
1. config 파일 복사<br> 
각 클러스터의 '~/.kube/config' 경로에 존재하는 config 파일을 조인할 클러스터의 '~/.kube/config' 경로에 서로 복사한다
2. 환경변수 등록<br>
a. 컨피그 파일 등록 : export KUBECONFIG=\$HOME/.kube/config <br>
b. 클러스터 조인 : export KUBECONFIG=\$KUBECONFIG:\$HOME/.kube/\$상대컨피그네임 (:로 덧붙이기) <br>
클러스터가 정상적으로 조인되었음을 다음과 같이 확인
```
[root@master ~]# kubectl config get-clusters
NAME
keti-gpu-cluster1
keti-gpu-cluster2
```
---
## 3.설치순서
### (1) Service 등록
외부 클러스터에서 해당 모듈에 접근할 수 있도록 {cluster-manager-service.yaml} 배포
```
[service yaml 배포]
# kubectl create -f cluster-manager-service.yaml
or
[service 쉘스크립트 실행]
# ./1.service-cluster-manager.sh
```
```
[root@master ~]# kubectl get service -A | grep cluster-manager
gpu           cluster-manager          NodePort       10.96.52.12      <none>        80:31000/TCP             23h
```
### (2) Cluster rbac 설정
Cluster Manaer 모듈에 권한부여를 위한 {role-binding.yaml / service-account.yaml} 배포
```
[rbac yaml 배포]
# kubectl create -f rbac/role-binding.yaml
# kubectl create -f rbac/service-account.yaml
or
[rbac 쉘스크립트 실행]
# ./2.rbac-gpucluster-manager.sh
```
```
[root@master ~]# kubectl get serviceaccount -A | grep cluster-manager
gpu               cluster-manager                      1         23h
```
### (3) Cluster Manager 모듈 배포
클러스터 롤바인딩, 서비스 어카운트 배포 후 클러스터 매니저 정상 동작 가능 <br>
Cluster Manager의 클러스터 스케줄링을 위해 다음 모듈 {
*[GPU-Scheduler](https://github.com/KETI-ExaScale/GPU-Scheduler)* }이 정상 동작하고 있어야 함
```
[cluster manager yaml 배포]
# kubectl apply -f cluster-manager.yaml
or
[cluster manager 쉘스크립트 실행]
# ./3.create-cluster-manager.sh
```
최종적으로 다음과 같은 모듈 4개가 Running 상태일 때 정상 스케줄링 가능
```
[root@master ~]# k get po -n gpu
NAME                                    READY   STATUS    RESTARTS      AGE
gpu-scheduler-66f95d7c8d-b9wwj          1/1     Running   0             23h
influxdb-0                              1/1     Running   0             19d
keti-cluster-manager-57489c8ddc-sqrx2   1/1     Running   0             23h
keti-gpu-device-plugin-lg7jx            2/2     Running   3 (22d ago)   22d
keti-gpu-metric-collector-rtd2s         1/1     Running   44 (8h ago)   19d
```
### (5) 클러스터 스케줄링 수행
+ Pod yaml에 Cluster명을 지정하는 경우 metadata > annotation > clusterName에 클러스터명 지정된 클러스터에 정상 배포
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
+ Pod yaml에 Cluster명을 지정하지 않는 경우 클러스터 스케줄링 수행 후 결과 반환
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
