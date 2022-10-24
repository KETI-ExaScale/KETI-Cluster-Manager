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
		fmt.Println("check1", env)
		return env, nil
	}
	//나중엔 config에 하나로 합치기!
	path, err := homedir.Expand("/root/.kube")
	if err != nil {
		fmt.Println("check2", path)
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
	kubeConfigPath, err := findKubeConfig()
	if err != nil {
		log.Fatal(err)
		return err
	}
	fmt.Println("kubeConfigPath: ", kubeConfigPath)

	files, err := ioutil.ReadDir(kubeConfigPath)
	if err != nil {
		fmt.Println("<error> Read Kubeconfig Path error-", err)
	}

	for _, file := range files {
		if file.Name() == "cache" {
			continue
		}

		kubeConfigPath_ := ""
		fmt.Println("filename:", file.Name())
		kubeConfigPath_ = fmt.Sprintf("%v/%v", kubeConfigPath, file.Name())
		fmt.Println("path:", kubeConfigPath_)

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
				fmt.Println("<error> ", err)
				cm.ClusterInfoList[name] = clusterInfo
				return err
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				fmt.Println("<error> ", err)
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
			fmt.Println("cluster name: ", name)
			fmt.Println("-ClusterIP ", clusterInfo.ClusterIP)
			fmt.Println("---")
		}
	}

	cm.DumpCache() //확인용

	return nil
}

func (cm *ClusterManager) DumpCache() {
	fmt.Println("#Dump Cluster Manager Cache")
	for clustername, clusterinfo := range cm.ClusterInfoList {
		fmt.Println("--")
		fmt.Println("1. cluster name: ", clustername)
		fmt.Println("2. cluster ip", clusterinfo.ClusterIP)
		fmt.Println("3. cluster available", clusterinfo.Avaliable)
		fmt.Println("4. cluster initialized", clusterinfo.Initialized)
		for nodename, nodeinfo := range clusterinfo.NodeInfoList {
			fmt.Print("*5. node name: ", nodename)
			fmt.Print(" | node score: ", nodeinfo.NodeScore)
			fmt.Println(" | node gpu count: ", nodeinfo.GPUCount)
		}
	}
}

// hostConfig, _ := rest.InClusterConfig()
// hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
// podsInNode, _ := hostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
// for _, pod := range podsInNode.Items {
// 	fmt.Println("# pod: ", pod.Name)
// }
// fmt.Println("~~")

// kubeConfigPath, err := findKubeConfig()
// if err != nil {
// 	fmt.Println("check3")
// 	log.Fatal(err)
// 	return err
// }
// fmt.Println("kubeConfigPath: ", kubeConfigPath)

// kubeConfig, err := clientcmd.LoadFromFile(kubeConfigPath)
// if err != nil {
// 	fmt.Println("check4")
// 	log.Fatal(err)
// 	return err
// }
// current_context := kubeConfig.CurrentContext
// fmt.Println("current context is ", current_context)

// contexts := kubeConfig.Contexts
// current_cluster := contexts[current_context].Cluster
// fmt.Println("current cluster is ", current_cluster)

// clusters := kubeConfig.Clusters
// fmt.Println(clusters)

// // for name, context := range contexts {
// // 	fmt.Println("-context name: ", name)
// // 	fmt.Println("-context.Cluster ", context.Cluster)
// // 	fmt.Println("-context.AuthInfo ", context.AuthInfo)
// // 	fmt.Println("-context.Namespace ", context.Namespace)
// // 	fmt.Println("-context.LocationOfOrigin ", context.LocationOfOrigin)
// // 	fmt.Println("-context.Extensions ", context.Extensions)
// // 	fmt.Println("---")
// // 	myclustername = name
// // }
// server := ""
// for name, cluster := range clusters {
// 	fmt.Println("-cluster name: ", name)
// 	fmt.Println("-cluster.Server ", cluster.Server)
// 	fmt.Println("---")
// 	if name == current_cluster {
// 		server = cluster.Server
// 	}
// }
// fmt.Println("current server: ", server)
// config, err := clientcmd.BuildConfigFromFlags(server, kubeConfigPath)
// if err != nil {
// 	fmt.Println("check5")
// 	fmt.Println(err)
// }
// config.TLSClientConfig = rest.TLSClientConfig{Insecure: true}

// clientset, err := kubernetes.NewForConfig(config)
// if err != nil {
// 	fmt.Println("check6")
// 	fmt.Println(err)
// }

// podsInNode, err = clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
// if err != nil {
// 	fmt.Println("check7")
// 	fmt.Println(err)
// }
// for _, pod := range podsInNode.Items {
// 	fmt.Println("# pod: ", pod.Name)
// }

// fmt.Println("~~~")
// server2 := "https://10.0.5.66:6443"
// config, err = clientcmd.BuildConfigFromFlags(server2, kubeConfigPath)
// if err != nil {
// 	fmt.Println("check5")
// 	fmt.Println(err)
// }
// config.TLSClientConfig = rest.TLSClientConfig{Insecure: true}

// clientset, err = kubernetes.NewForConfig(config)
// if err != nil {
// 	fmt.Println("check6")
// 	fmt.Println(err)
// }

// podsInNode, _ = clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

// for _, pod := range podsInNode.Items {
// 	fmt.Println("# pod: ", pod.Name)
// }
// return nil

// func ListKubeFedClusters(genClient genericclient.Client) *fedv1b1.KubeFedClusterList {
// 	tempClusterList := &fedv1b1.KubeFedClusterList{}
// 	clusterList := &fedv1b1.KubeFedClusterList{}

// 	err := genClient.List(context.TODO(), tempClusterList, "", &client.ListOptions{})

// 	if err != nil {
// 		fmt.Printf("Error retrieving list of federated clusters: %+v\n", err)
// 	}

// 	for _, cluster := range tempClusterList.Items {
// 		status := true

// 		for _, cond := range cluster.Status.Conditions {
// 			if cond.Type == "Offline" {
// 				status = false
// 				break
// 			}
// 		}
// 		if status {
// 			clusterList.Items = append(clusterList.Items, cluster)
// 		}
// 	}

// 	return clusterList
// }

// func KubeFedClusterConfigs(clusterList *fedv1b1.KubeFedClusterList, genClient genericclient.Client) map[string]*rest.Config {
// 	clusterConfigs := make(map[string]*rest.Config)
// 	for _, cluster := range clusterList.Items {
// 		config, _ := util.BuildClusterConfig(&cluster, genClient, "")
// 		clusterConfigs[cluster.Name] = config
// 	}
// 	return clusterConfigs
// }

// func KubeFedClusterKubeClients(clusterList *fedv1b1.KubeFedClusterList, cluster_configs map[string]*rest.Config) map[string]*kubernetes.Clientset {

// 	cluster_clients := make(map[string]*kubernetes.Clientset)
// 	for _, cluster := range clusterList.Items {
// 		clusterName := cluster.Name
// 		cluster_config := cluster_configs[clusterName]
// 		cluster_client := kubernetes.NewForConfigOrDie(cluster_config)
// 		cluster_clients[clusterName] = cluster_client
// 	}
// 	return cluster_clients
// }

// type Client struct {
// 	secretsInterface typedcorev1.SecretInterface
// }

// func New(secretsInterface typedcorev1.SecretInterface) Client {
// 	return Client{secretsInterface}
// }

// var resourceName = func(name string) string {
// 	return fmt.Sprintf("%s-kubeconfig", name)
// }

// const secretKey = "value"

// func (g Client) Get(ctx context.Context, name string) (*clientcmdapi.Config, error) {
// 	resourceName := resourceName(name)
// 	secret, err := g.secretsInterface.Get(ctx, resourceName, metav1.GetOptions{})
// 	if errors.IsNotFound(err) {
// 		return clientcmdapi.NewConfig(), nil
// 	}
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get kubeconfig secret: %w", err)
// 	}
// 	data, ok := secret.Data[secretKey]
// 	if !ok {
// 		return nil, fmt.Errorf("key %q not found in secret %s/%s", secretKey, secret.Namespace, secret.Name)
// 	}
// 	v, err := clientcmd.Load(data)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to unmarshall data: %w", err)
// 	}
// 	return v, nil
// }
