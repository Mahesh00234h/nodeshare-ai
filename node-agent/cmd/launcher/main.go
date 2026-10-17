package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const (
	defaultAgentPath  = `C:\ProgramData\DistributedAI\bin\node-agent.exe`
	defaultController = "https://llm-cluster.onrender.com:443"
	defaultToken      = "dev-enrollment-token"
)

func main() {
	controllerURL := flag.String("controller-url", defaultController, "Target Python Controller URL")
	enrollmentToken := flag.String("enrollment-token", defaultToken, "Cluster enrollment token")
	flag.Parse()

	// 1. Locate Node Agent Executable
	agentPath := defaultAgentPath
	if !fileExists(agentPath) {
		if fileExists("node-agent.exe") {
			agentPath = "node-agent.exe"
		} else if fileExists(`bin\node-agent.exe`) {
			agentPath = `bin\node-agent.exe`
		} else {
			// Search current executable directory
			exePath, err := os.Executable()
			if err == nil {
				localAgent := filepath.Join(filepath.Dir(exePath), "node-agent.exe")
				if fileExists(localAgent) {
					agentPath = localAgent
				}
			}
		}
	}

	if !fileExists(agentPath) {
		showGUIError("Distributed AI Agent Error", fmt.Sprintf("Could not find node-agent.exe at %s\nPlease run node-agent-installer.exe first.", agentPath))
		os.Exit(1)
	}

	// 2. Launch node-agent.exe as a completely detached hidden process
	cmd := exec.Command(agentPath, "--controller-url", *controllerURL, "--enrollment-token", *enrollmentToken)
	cmd.Dir = `C:\ProgramData\DistributedAI`
	
	// Hide console window and breakaway from parent job object in Windows
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW | windows.DETACHED_PROCESS | windows.CREATE_BREAKAWAY_FROM_JOB,
	}

	if err := cmd.Start(); err != nil {
		showGUIError("Distributed AI Agent Error", fmt.Sprintf("Failed to launch background agent: %v", err))
		os.Exit(1)
	}

	// Exit launcher immediately leaving node-agent running continuously in background
	os.Exit(0)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func showGUIError(title, message string) {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	msgPtr, _ := windows.UTF16PtrFromString(message)
	windows.MessageBox(0, msgPtr, titlePtr, windows.MB_OK|windows.MB_ICONERROR)
}
