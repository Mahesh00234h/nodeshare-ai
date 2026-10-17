package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type AgentService interface {
	RunForeground() error
	StartService() error
	StopService() error
}

func WaitForShutdownSignal(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received shutdown signal. Stopping Node Agent...")
	if cancel != nil {
		cancel()
	}
}
