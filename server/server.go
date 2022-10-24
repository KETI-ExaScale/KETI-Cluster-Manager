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
	resource.KetiClusterManager.DumpCache() //확인용

	//Init Other Cluster
	for name, cluster := range resource.KetiClusterManager.ClusterInfoList {
		if name != mycluster {
			fmt.Println("#Init Other Cluster-", name)
			host := cluster.ClusterIP + ":" + nodePort
			conn, err := grpc.Dial(host, grpc.WithInsecure())
			if err != nil {
				cluster.Avaliable = false
				fmt.Println("<error> Init Other Cluster Connection - ", err)
				continue
			}
			defer conn.Close()

			grpcClient := pb.NewClusterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

			if(cluster.Avaliable)

			// InitOtherClusterRequest 구성
			var initOtherClusterRequest = &pb.InitOtherClusterRequest{
				ClusterName:    mycluster,
				RequestMessage: requestMessage,
			}

			flag, err := grpcClient.InitOtherCluster(ctx, initOtherClusterRequest)
			if err != nil {
				cancel()
				fmt.Println("cluster {", name, "} doesn't have cluster manager")
				continue
			} else if !flag.Success {
				cancel()
				fmt.Println("<error> Init Other Cluster Call - ", err)
				continue
			}

			// cluster.Avaliable = true

			cancel()

		}
	}

	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

func (s *ClusterServer) InitOtherCluster(ctx context.Context, req *pb.InitOtherClusterRequest) (*pb.ResponseMessage, error) {
	resource.KetiClusterManager.WLockCM()

	targetCluster := req.ClusterName
	fmt.Println("#Init Other Cluster-", targetCluster)

	for _, message := range req.RequestMessage {
		nodeName := message.NodeName
		score := message.NodeScore
		gpuCount := message.GpuCount

		nodeInfo := resource.NewNodeInfo(nodeName, score, gpuCount)
		resource.KetiClusterManager.ClusterInfoList[targetCluster].NodeInfoList[nodeName] = nodeInfo
	}

	resource.KetiClusterManager.ClusterInfoList[targetCluster].Avaliable = true
	resource.KetiClusterManager.DumpCache() //확인용

	resource.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
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
