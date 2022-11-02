package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	pb "cluster-manager/proto"
	r "cluster-manager/resources"
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

//gRPC Server Run
func Run(quit chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-quit:
			wg.Done()
			r.KETI_LOG_L3("Stopped scheduler.")
			return
		default:
			lis, err := net.Listen("tcp", ":"+portNumber)
			if err != nil {
				r.KETI_LOG_L3(fmt.Sprintf("failed to listen: %v", err))
			}

			grpcServer := grpc.NewServer()
			pb.RegisterClusterServer(grpcServer, &ClusterServer{})

			r.KETI_LOG_L2(fmt.Sprintf("\n# Start gRPC Server >> Port:%s", portNumber))
			if err := grpcServer.Serve(lis); err != nil {
				r.KETI_LOG_L3(fmt.Sprintf("<error> failed to serve: %v", err))
			}
		}
	}

}

//내 클러스터 스케줄러가 호출하는 내 클러스터 점수 초기화 함수 (called by my cluster scheduler)
func (s *ClusterServer) InitMyCluster(ctx context.Context, req *pb.InitMyClusterRequest) (*pb.ResponseMessage, error) {
	r.KetiClusterManager.WLockCM()
	r.KETI_LOG_L3("# Init My Cluster Called")

	mycluster := r.KetiClusterManager.MyClusterName
	var requestMessage []*pb.RequestMessage

	for _, message := range req.RequestMessage {
		r.KETI_LOG_L3(fmt.Sprintf("- Init My Cluster Message: %v", message))
		nodeName := message.NodeName
		score := message.NodeScore
		gpuCount := message.GpuCount
		var msg = &pb.RequestMessage{
			NodeName:  nodeName,
			NodeScore: score,
			GpuCount:  gpuCount,
		}
		requestMessage = append(requestMessage, msg)

		nodeInfo := r.NewNodeInfo(nodeName, score, gpuCount)
		r.KetiClusterManager.ClusterInfoList[mycluster].NodeInfoList[nodeName] = nodeInfo
	}

	r.KetiClusterManager.ClusterInfoList[mycluster].Avaliable = true
	r.KetiClusterManager.ClusterInfoList[mycluster].Initialized = true

	//Init Other Cluster
	for name, cluster := range r.KetiClusterManager.ClusterInfoList {
		if name != mycluster {
			// //다른 모든 클러스터로 나의 Init Node Score 전달
			ctx_, cancel := context.WithTimeout(context.Background(), time.Second*15)
			s.CallInitOtherCluster(ctx_, cluster, requestMessage)
			cancel()
		}
	}
	r.KetiClusterManager.DumpCache() //확인용

	r.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

//조인 클러스터의 클러스터 매니저가 호출하는 조인 클러스터 점수 초기화 함수 (called by other cluster manager)
func (s *ClusterServer) InitOtherCluster(ctx context.Context, req *pb.InitOtherClusterRequest) (*pb.ResponseMessage, error) {
	r.KetiClusterManager.WLockCM()

	targetCluster := r.KetiClusterManager.ClusterInfoList[req.ClusterName]
	r.KETI_LOG_L3(fmt.Sprintf("\n# Init Other Cluster Called {%s}", req.ClusterName))

	for _, message := range req.RequestMessage {
		r.KETI_LOG_L3(fmt.Sprintf("- Init Other Cluster Message: %v", message))
		nodeName := message.NodeName
		score := message.NodeScore
		gpuCount := message.GpuCount

		nodeInfo := r.NewNodeInfo(nodeName, score, gpuCount)
		targetCluster.NodeInfoList[nodeName] = nodeInfo
	}

	targetCluster.Avaliable = true

	myCluster := r.KetiClusterManager.ClusterInfoList[r.KetiClusterManager.MyClusterName]
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
	}

	r.KetiClusterManager.DumpCache() //확인용
	r.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

//내 클러스터 스케줄러가 호출하는 내 클러스터 점수 업데이트 함수 (called by my cluster scheudler)
func (s *ClusterServer) UpdateMyCluster(ctx context.Context, req *pb.UpdateMyClusterRequest) (*pb.ResponseMessage, error) {
	r.KetiClusterManager.WLockCM()

	r.KETI_LOG_L3("\n# Update My Cluster")
	nodeName := req.RequestMessage.NodeName
	score := req.RequestMessage.NodeScore
	r.KETI_LOG_L3(fmt.Sprintf("- Update My Cluster Message: %v", req.RequestMessage))

	mycluster := r.KetiClusterManager.MyClusterName
	r.KetiClusterManager.ClusterInfoList[mycluster].NodeInfoList[nodeName].NodeScore = score
	r.KetiClusterManager.DumpCache() //확인용

	//Update Other Cluster
	for name, cluster := range r.KetiClusterManager.ClusterInfoList {
		if cluster.Avaliable && name != mycluster {
			r.KETI_LOG_L2(fmt.Sprintf("# Update Other Cluster-%s", name))
			host := cluster.ClusterIP + ":" + nodePort
			conn, err := grpc.Dial(host, grpc.WithInsecure())
			if err != nil {
				cluster.Avaliable = false
				r.KETI_LOG_L3(fmt.Sprintf("<error> Update Other Cluster Connection - %s", err))
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
				r.KETI_LOG_L3(fmt.Sprintf("<error> Update Other Cluster Call - %s", err))
				continue
			}

			cancel()
		}
	}

	r.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

//조인 클러스터의 클러스터 매니저가 호출하는 조인 클러스터 점수 업데이트 함수 (called by other cluster manager)
func (s *ClusterServer) UpdateOtherCluster(ctx context.Context, req *pb.UpdateOtherClusterRequest) (*pb.ResponseMessage, error) {
	r.KetiClusterManager.WLockCM()
	r.KETI_LOG_L3("\n# Update Other Cluster")

	r.KETI_LOG_L3(fmt.Sprintf("- Update Other Cluster Message: %v", req))

	targetCluster := req.ClusterName
	nodeName := req.RequestMessage.NodeName
	score := req.RequestMessage.NodeScore

	r.KetiClusterManager.ClusterInfoList[targetCluster].NodeInfoList[nodeName].NodeScore = score
	r.KetiClusterManager.DumpCache() //확인용
	r.KetiClusterManager.UnWLockCM()

	return &pb.ResponseMessage{
		Success: true,
	}, nil
}

//내 클러스터 초기 점수를 다른 클러스터의 클러스터 매니저로 전달하는 함수 (called in InitMyCluster Function)
func (s *ClusterServer) CallInitOtherCluster(ctx context.Context, targetCluster *r.ClusterInfo, requestMessage []*pb.RequestMessage) {
	//해당 클러스터로 나의 Init Node Score 정보 전달
	r.KETI_LOG_L2(fmt.Sprintf("# Call Init Other Cluster {%s}", targetCluster.ClusterName))
	host := targetCluster.ClusterIP + ":" + nodePort
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("<error> Init Other Cluster Connection - %s", err))
	}
	defer conn.Close()

	grpcClient := pb.NewClusterClient(conn)

	// InitOtherClusterRequest 구성
	var initOtherClusterRequest = &pb.InitOtherClusterRequest{
		ClusterName:    r.KetiClusterManager.MyClusterName,
		RequestMessage: requestMessage,
	}

	flag, err := grpcClient.InitOtherCluster(ctx, initOtherClusterRequest)
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("<error> %s", err))
		r.KETI_LOG_L3(fmt.Sprintf("<error> cluster {%s} dosen't have cluster manager", targetCluster.ClusterName))
	} else if !flag.Success {
		r.KETI_LOG_L3(fmt.Sprintf("<error> init other cluster call - %s", err))
	} else {
		targetCluster.Initialized = true
	}
}

//내 클러스터 스케줄러가 호출하는 클러스터 스케줄링 요청 함수 (called by my cluster scheduler)
func (s *ClusterServer) RequestClusterScheduling(ctx context.Context, req *pb.ClusterSchedulingRequest) (*pb.ClusterSchedulingResponse, error) {
	r.KetiClusterManager.RLockCM()
	r.KETI_LOG_L3("\n# Request Cluster Scheduling Called")

	gpuCount := int(req.GpuCount)
	var filteredCluster []string
	r.KETI_LOG_L3(fmt.Sprintf("- Scheuling Request Pod Spec: (%v)", req))

	flag := true
	bestCluster := scheduling.FindCluster(gpuCount, filteredCluster)
	if bestCluster.ClusterName == "" {
		r.KETI_LOG_L3("<error> cannot find best cluster")
		flag = false
	}

	r.KETI_LOG_L3(fmt.Sprintf("- Best Cluster Is: %v", bestCluster))

	r.KetiClusterManager.UnRLockCM()

	return &pb.ClusterSchedulingResponse{
		ClusterName: bestCluster.ClusterName,
		Success:     flag,
	}, nil
}
