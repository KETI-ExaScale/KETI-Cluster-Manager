package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "cluster-manager/proto"
	resource "cluster-manager/resources"
	scheduling "cluster-manager/scheduling"

	"google.golang.org/grpc"
)

const (
	portNumber = "8686"
	nodePort   = "31000"
)

type ClusterServer struct {
	pb.ClusterServer
}

func Run(quit chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-quit:
			wg.Done()
			log.Println("Stopped scheduler.")
			return
		default:
			lis, err := net.Listen("tcp", ":"+portNumber)
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			grpcServer := grpc.NewServer()
			pb.RegisterClusterServer(grpcServer, &ClusterServer{})

			log.Printf("start gRPC server on %s port", portNumber)
			if err := grpcServer.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %s", err)
			}
		}
	}

}

func (s *ClusterServer) InitMyCluster(ctx context.Context, req *pb.InitMyClusterRequest) (*pb.ResponseMessage, error) {
	resource.KetiClusterManager.WLockCM()
	fmt.Println("#Init My Cluster Called")

	mycluster := resource.KetiClusterManager.MyClusterName
	var requestMessage []*pb.RequestMessage

	for _, message := range req.RequestMessage {
		fmt.Println("message:", message)
		nodeName := message.NodeName
		score := message.NodeScore
		gpuCount := message.GpuCount
		var msg = &pb.RequestMessage{
			NodeName:  nodeName,
			NodeScore: score,
			GpuCount:  gpuCount,
		}
		requestMessage = append(requestMessage, msg)

		nodeInfo := resource.NewNodeInfo(nodeName, score, gpuCount)
		fmt.Println("nodeInfo:", nodeInfo)
		resource.KetiClusterManager.ClusterInfoList[mycluster].NodeInfoList[nodeName] = nodeInfo
	}

	resource.KetiClusterManager.ClusterInfoList[mycluster].Avaliable = true
	resource.KetiClusterManager.ClusterInfoList[mycluster].Initialized = true

	//Init Other Cluster
	for name, cluster := range resource.KetiClusterManager.ClusterInfoList {
		if name != mycluster {
			// //다른 모든 클러스터로 나의 Init Node Score 전달
			ctx_, cancel := context.WithTimeout(context.Background(), time.Second*15)
			s.CallInitOtherCluster(ctx_, cluster, requestMessage)
			cancel()
			// fmt.Println("#Init Other Cluster-", name)
			// host := cluster.ClusterIP + ":" + nodePort
			// conn, err := grpc.Dial(host, grpc.WithInsecure())
			// if err != nil {
			// 	fmt.Println("<error> Init Other Cluster Connection - ", err)
			// 	continue
			// }
			// defer conn.Close()

			// grpcClient := pb.NewClusterClient(conn)
			// ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

			// // InitOtherClusterRequest 구성
			// var initOtherClusterRequest = &pb.InitOtherClusterRequest{
			// 	ClusterName:    mycluster,
			// 	RequestMessage: requestMessage,
			// }

			// flag, err := grpcClient.InitOtherCluster(ctx, initOtherClusterRequest)
			// if err != nil {
			// 	fmt.Println("cluster {", name, "} doesn't have cluster manager")
			// } else if !flag.Success {
			// 	fmt.Println("<error> Init Other Cluster Call - ", err)
			// } else {
			// 	cluster.Initialized = true
			// }
			// cancel()
		}
	}
	resource.KetiClusterManager.DumpCache() //확인용

	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

func (s *ClusterServer) InitOtherCluster(ctx context.Context, req *pb.InitOtherClusterRequest) (*pb.ResponseMessage, error) {
	resource.KetiClusterManager.WLockCM()

	targetCluster := resource.KetiClusterManager.ClusterInfoList[req.ClusterName]
	fmt.Println("#Init Other Cluster Called-", req.ClusterName)

	for _, message := range req.RequestMessage {
		nodeName := message.NodeName
		score := message.NodeScore
		gpuCount := message.GpuCount

		nodeInfo := resource.NewNodeInfo(nodeName, score, gpuCount)
		targetCluster.NodeInfoList[nodeName] = nodeInfo
	}

	targetCluster.Avaliable = true

	myCluster := resource.KetiClusterManager.ClusterInfoList[resource.KetiClusterManager.MyClusterName]
	if myCluster.Avaliable && !targetCluster.Initialized {
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*15)
		// InitOtherClusterRequest 구성
		var requestMessageList []*pb.RequestMessage
		for _, nodeinfo := range myCluster.NodeInfoList {
			var requestMessage = &pb.RequestMessage{
				NodeName:  nodeinfo.NodeName,
				NodeScore: nodeinfo.NodeScore,
				GpuCount:  nodeinfo.GPUCount,
			}
			requestMessageList = append(requestMessageList, requestMessage)
		}
		s.CallInitOtherCluster(ctx_, targetCluster, requestMessageList)
		cancel()
		// //해당 클러스터로 나의 Init Node Score 정보 전달
		// fmt.Println("#Init Other Cluster-", req.ClusterName)
		// host := targetCluster.ClusterIP + ":" + nodePort
		// conn, err := grpc.Dial(host, grpc.WithInsecure())
		// if err != nil {
		// 	fmt.Println("<error> Init Other Cluster Connection - ", err)
		// }
		// defer conn.Close()

		// grpcClient := pb.NewClusterClient(conn)
		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

		// // InitOtherClusterRequest 구성
		// var requestMessageList []*pb.RequestMessage
		// for _, nodeinfo := range myCluster.NodeInfoList {
		// 	var requestMessage = &pb.RequestMessage{
		// 		NodeName:  nodeinfo.NodeName,
		// 		NodeScore: nodeinfo.NodeScore,
		// 		GpuCount:  nodeinfo.GPUCount,
		// 	}
		// 	requestMessageList = append(requestMessageList, requestMessage)
		// }
		// var initOtherClusterRequest = &pb.InitOtherClusterRequest{
		// 	ClusterName:    resource.KetiClusterManager.MyClusterName,
		// 	RequestMessage: requestMessageList,
		// }

		// flag, err := grpcClient.InitOtherCluster(ctx, initOtherClusterRequest)
		// if err != nil {
		// 	fmt.Println(err)
		// 	fmt.Println("cluster {", req.ClusterName, "} doesn't have cluster manager")
		// } else if !flag.Success {
		// 	fmt.Println("<error> Init Other Cluster Call - ", err)
		// } else {
		// 	targetCluster.Initialized = true
		// }
		// cancel()
	}
	resource.KetiClusterManager.DumpCache() //확인용

	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

func (s *ClusterServer) CallInitOtherCluster(ctx context.Context, targetCluster *resource.ClusterInfo, requestMessage []*pb.RequestMessage) {
	//해당 클러스터로 나의 Init Node Score 정보 전달
	fmt.Println("#Call Init Other Cluster-", targetCluster.ClusterName)
	host := targetCluster.ClusterIP + ":" + nodePort
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("<error> Init Other Cluster Connection - ", err)
	}
	defer conn.Close()

	grpcClient := pb.NewClusterClient(conn)

	// InitOtherClusterRequest 구성
	var initOtherClusterRequest = &pb.InitOtherClusterRequest{
		ClusterName:    resource.KetiClusterManager.MyClusterName,
		RequestMessage: requestMessage,
	}

	flag, err := grpcClient.InitOtherCluster(ctx, initOtherClusterRequest)
	if err != nil {
		fmt.Println(err)
		fmt.Println("cluster {", targetCluster.ClusterName, "} dosen't have cluster manager")
	} else if !flag.Success {
		fmt.Println("<error> Init Other Cluster Call - ", err)
	} else {
		targetCluster.Initialized = true
	}
}

func (s *ClusterServer) UpdateMyCluster(ctx context.Context, req *pb.UpdateMyClusterRequest) (*pb.ResponseMessage, error) {
	resource.KetiClusterManager.WLockCM()

	fmt.Println("#Update My Cluster")
	nodeName := req.RequestMessage.NodeName
	score := req.RequestMessage.NodeScore
	fmt.Println("message:", req.RequestMessage)

	mycluster := resource.KetiClusterManager.MyClusterName
	resource.KetiClusterManager.ClusterInfoList[mycluster].NodeInfoList[nodeName].NodeScore = score
	resource.KetiClusterManager.DumpCache() //확인용

	//Update Other Cluster
	for name, cluster := range resource.KetiClusterManager.ClusterInfoList {
		if cluster.Avaliable && name != mycluster {
			fmt.Println("#Update Other Cluster-", name)
			host := cluster.ClusterIP + ":" + nodePort
			conn, err := grpc.Dial(host, grpc.WithInsecure())
			if err != nil {
				cluster.Avaliable = false
				fmt.Println("<error> Update Other Cluster Connection - ", err)
				continue
			}
			defer conn.Close()

			grpcClient := pb.NewClusterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

			var requestMessage = &pb.RequestMessage{
				NodeName:  nodeName,
				NodeScore: score,
			}
			var updateOtherClusterRequest = &pb.UpdateOtherClusterRequest{
				ClusterName:    mycluster,
				RequestMessage: requestMessage,
			}

			flag, err := grpcClient.UpdateOtherCluster(ctx, updateOtherClusterRequest)
			if err != nil || !flag.Success {
				cancel()
				fmt.Println("<error> Update Other Cluster Call - ", err)
				continue
			}

			cancel()
		}
	}

	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

func (s *ClusterServer) UpdateOtherCluster(ctx context.Context, req *pb.UpdateOtherClusterRequest) (*pb.ResponseMessage, error) {
	resource.KetiClusterManager.WLockCM()
	fmt.Println("#Update Other Cluster")

	targetCluster := req.ClusterName
	nodeName := req.RequestMessage.NodeName
	score := req.RequestMessage.NodeScore

	resource.KetiClusterManager.ClusterInfoList[targetCluster].NodeInfoList[nodeName].NodeScore = score
	resource.KetiClusterManager.DumpCache() //확인용
	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

func (s *ClusterServer) RequestClusterScheduling(ctx context.Context, req *pb.ClusterSchedulingRequest) (*pb.ClusterSchedulingResponse, error) {
	resource.KetiClusterManager.RLockCM()
	fmt.Println("#Request Cluster Scheduling Called")

	gpuCount := int(req.GpuCount)
	var filteredCluster []string
	fmt.Println("-pod requested gpu :", gpuCount)

	flag := true
	bestCluster := scheduling.FindCluster(gpuCount, filteredCluster)
	if bestCluster.ClusterName == "" {
		fmt.Println("<error> Cannot Find Best Cluster")
		flag = false
	}

	fmt.Println("#Best Cluster Is: ", bestCluster)

	resource.KetiClusterManager.UnRLockCM()

	return &pb.ClusterSchedulingResponse{
		ClusterName: bestCluster.ClusterName,
		Success:     flag,
	}, nil
}
