package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distributed-ai/distributed-runtime/pkg/runtime"
)

func main() {
	cacheDir := flag.String("cache-dir", `C:\ProgramData\DistributedAI\models`, "Directory path for local GGUF model cache")
	nodeID := flag.String("node-id", "node-local", "Unique identifier for this compute node")
	rpcPort := flag.Int("rpc-port", 50052, "Local LAN port for llama-rpc-server worker")
	flag.Parse()

	log.Printf("==========================================================")
	log.Printf(" Starting Distributed AI Inference Runtime Daemon v1.0")
	log.Printf(" Engine: llama.cpp | Node ID: %s | RPC Port: %d", *nodeID, *rpcPort)
	log.Printf(" Cache Directory: %s", *cacheDir)
	log.Printf("==========================================================")

	rt, err := runtime.NewDistributedRuntime(*cacheDir, *nodeID, *rpcPort)
	if err != nil {
		log.Fatalf("Failed to initialize distributed runtime: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		log.Fatalf("Failed to start runtime daemon: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Runtime daemon running. Press Ctrl+C to terminate.")
	<-sigChan

	log.Printf("Shutdown signal received. Stopping distributed runtime...")
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	_ = rt.Stop(stopCtx)
	log.Printf("Distributed runtime daemon stopped gracefully.")
}
