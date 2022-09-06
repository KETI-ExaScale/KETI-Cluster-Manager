package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	resource "cluster-manager/resources"
	server "cluster-manager/server"
)

func main() {
	resource.KetiClusterManager = resource.NewClusterManager()
	err := resource.KetiClusterManager.InitClusterManager()
	if err != nil {
		fmt.Println("<error> Init Cluster Manager Error, reason: ", err)
	}

	quitChan := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go server.Run(quitChan, &wg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			log.Printf("Shutdown signal received, exiting...")
			close(quitChan)
			wg.Wait()
			os.Exit(0)
		}
	}
}
