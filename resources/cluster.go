package resources

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var KetiClusterManager *ClusterManager

const trimPostfix = "https://"

type ClusterManager struct {
	mutex           sync.RWMutex
	MyClusterName   string
	ClusterInfoList map[string]*ClusterInfo
}

func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		MyClusterName:   "",
		ClusterInfoList: make(map[string]*ClusterInfo),
	}
}

func findKubeConfig() (string, error) {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env, nil
	}
	//나중엔 config에 하나로 합치기!
	path, err := homedir.Expand("/root/.kube")
	if err != nil {
		return "", err
	}
	return path, nil
}

func (cm *ClusterManager) RLockCM() {
	cm.mutex.RLock()
}

func (cm *ClusterManager) UnRLockCM() {
	cm.mutex.RUnlock()
}

func (cm *ClusterManager) WLockCM() {
	cm.mutex.Lock()
}

func (cm *ClusterManager) UnWLockCM() {
	cm.mutex.Unlock()
}

type ClusterInfo struct {
	ClusterName  string
	Config       *rest.Config
	Clientset    *kubernetes.Clientset
	ClusterIP    string
	NodeInfoList map[string]*NodeInfo
	Avaliable    bool //해당 클러스터의 점수 초기화 여부
	Initialized  bool //해당 클러스터에 내 클러스터 점수 초기화 여부
}

func NewClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		ClusterName:  "",
		Config:       nil,
		Clientset:    nil,
		ClusterIP:    "",
		NodeInfoList: make(map[string]*NodeInfo),
		Avaliable:    false,
		Initialized:  false,
	}
}

type NodeInfo struct {
	NodeName  string
	NodeScore int64
	GPUCount  int64
}

func NewNodeInfo(name string, score int64, gpu int64) *NodeInfo {
	return &NodeInfo{
		NodeName:  name,
		NodeScore: score,
		GPUCount:  gpu,
	}
}

// func FindClusterManagerHost(hostKubeClient *kubernetes.Clientset) string {
// 	fmt.Println("find cluster manager host")
// 	pods, err := hostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
// 	if err != nil {
// 		fmt.Println("<error> get pods error-", err)
// 	}

// 	for _, pod := range pods.Items {
// 		fmt.Println("#pod:", pod.Name)
// 		if strings.HasPrefix(pod.Name, "keti-cluster-manager") {
// 			fmt.Println("pod.Status.PodIP::", pod.Status.PodIP)
// 			return pod.Status.PodIP
// 		}
// 	}
// 	return ""
// }

func (cm *ClusterManager) InitClusterManager() error { //컨피그맵 읽고 create할 수 있도록
	KETI_LOG_L2("\n#Init Cluster Manager")
	kubeConfigPath, err := findKubeConfig()
	if err != nil {
		log.Fatal(err)
		return err
	}

	files, err := ioutil.ReadDir(kubeConfigPath)
	if err != nil {
		KETI_LOG_L3(fmt.Sprintf("<error> Read Kubeconfig Path error-%s", err))
	}

	for num, file := range files {
		if file.Name() == "cache" {
			continue
		}

		kubeConfigPath_ := ""
		kubeConfigPath_ = fmt.Sprintf("%v/%v", kubeConfigPath, file.Name())
		KETI_LOG_L1(fmt.Sprintf("%d-1. path:%s", num, kubeConfigPath_))

		kubeConfig, err := clientcmd.LoadFromFile(kubeConfigPath_)
		if err != nil {
			log.Fatal(err)
			return err
		}

		clusters := kubeConfig.Clusters
		currentContext := kubeConfig.CurrentContext
		currentCluster := kubeConfig.Contexts[currentContext].Cluster

		if file.Name() == "config" { //이름이 바뀔수도 있으니 다른 방법 생각
			cm.MyClusterName = currentCluster
		}

		for name, cluster := range clusters {
			clusterInfo := NewClusterInfo()

			config, err := clientcmd.BuildConfigFromFlags(cluster.Server, kubeConfigPath_)
			if err != nil {
				KETI_LOG_L3(fmt.Sprintf("<error> %s", err))
				cm.ClusterInfoList[name] = clusterInfo
				return err
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				KETI_LOG_L3(fmt.Sprintf("<error> %s", err))
				clusterInfo.Config = config
				cm.ClusterInfoList[name] = clusterInfo
				return err
			}

			clusterInfo.Config = config
			clusterInfo.Clientset = clientset
			ip := strings.TrimLeft(cluster.Server, trimPostfix)
			ipSlice := strings.Split(ip, ":")
			clusterInfo.ClusterIP = ipSlice[0]
			clusterInfo.ClusterName = name

			cm.ClusterInfoList[name] = clusterInfo
			KETI_LOG_L1(fmt.Sprintf("%d-2. cluster name: %s", num, name))
			KETI_LOG_L1(fmt.Sprintf("%d-3. cluster ip: %s", num, clusterInfo.ClusterIP))
		}
	}

	cm.DumpCache() //확인용

	return nil
}

func (cm *ClusterManager) DumpCache() {
	KETI_LOG_L1("\n#Dump Cluster Manager Cache")
	num := 1
	for clustername, clusterinfo := range cm.ClusterInfoList {
		KETI_LOG_L1(fmt.Sprintf("%d-1. cluster name: %s", num, clustername))
		KETI_LOG_L1(fmt.Sprintf("%d-2. cluster ip : %s", num, clusterinfo.ClusterIP))
		KETI_LOG_L1(fmt.Sprintf("%d-3. cluster available: %v", num, clusterinfo.Avaliable))
		KETI_LOG_L1(fmt.Sprintf("%d-4. cluster initialized: %v", num, clusterinfo.Initialized))
		num2 := 1
		for nodename, nodeinfo := range clusterinfo.NodeInfoList {
			KETI_LOG_L1(fmt.Sprintf("%d-5-%d. node name: %s | node score: %d | node gpu count: %d", num, num2, nodename, nodeinfo.NodeScore, nodeinfo.GPUCount))
			num2++
		}
		num++
	}
}
