package scheduling

import (
	resource "cluster-manager/resources"
	"fmt"
	"math"
)

type BestCluster struct {
	ClusterName string
	Score       int64
}

func FindCluster(gpu int, filtered []string) BestCluster {
	bestCluster := BestCluster{"", 0}

	for name, cinfo := range resource.KetiClusterManager.ClusterInfoList {
		if cinfo.Avaliable && !isFilteredCluster(name, filtered) {
			fmt.Println("--Cluster Name: ", name)
			clusterScore := calcClusterScore(gpu, cinfo)

			if clusterScore > bestCluster.Score {
				bestCluster.ClusterName = name
				bestCluster.Score = clusterScore
			}
		}
	}

	return bestCluster
}

//정확도를 높이는 방법은?
func calcClusterScore(gpu int, cinfo *resource.ClusterInfo) int64 {
	clusterScore := int64(0)
	var scoreList []int64
	var std float64
	totalScore := 0
	cnt := 0

	for _, ninfo := range cinfo.NodeInfoList {
		if int(ninfo.GPUCount) >= gpu {
			scoreList = append(scoreList, ninfo.NodeScore)
			totalScore += int(ninfo.NodeScore)
			cnt += 1
		}
	}

	if cnt == 0 {
		return 0
	}
	mean := int64(totalScore / cnt)
	// fmt.Println("clusterScore1:", totalScore, "/", cnt, "=", mean)

	for i := 0; i < cnt; i++ {
		std += math.Pow(float64(scoreList[i]-mean), 2)
	}
	std = math.Sqrt(std / float64(cnt))
	clusterScore = mean + int64(std)
	fmt.Println("---clusterScore:", clusterScore)

	// if gpu == 1 { //node 요청개수 1개
	// 	for i := 0; i < cnt; i++ {
	// 		std += math.Pow(float64(scoreList[i]-mean), 2)
	// 		fmt.Println("std1:", std)
	// 	}
	// 	std = math.Sqrt(std / float64(cnt))
	// 	fmt.Println("std2:", std)
	// 	clusterScore = mean + int64(std)
	// 	fmt.Println("clusterScore2:", clusterScore)

	// } else { //node 요청 개수 2개 이상
	// 	clusterScore = int64(mean)
	// }

	return clusterScore
}

func isFilteredCluster(c string, clist []string) bool {
	for _, cluster := range clist {
		if cluster == c {
			return true
		}
	}
	return false
}
