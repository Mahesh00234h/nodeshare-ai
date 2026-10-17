//go:build windows

package service

import (
	"context"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type WindowsNodeAgentService struct {
	runFn func(ctx context.Context) error
}

func NewWindowsService(runFn func(ctx context.Context) error) *WindowsNodeAgentService {
	return &WindowsNodeAgentService{runFn: runFn}
}

func (s *WindowsNodeAgentService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.runFn(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				break loop
			}
		case err := <-errChan:
			if err != nil {
				errno = 1
			}
			break loop
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return false, errno
}

func RunWindowsService(serviceName string, runFn func(ctx context.Context) error, isInteractive bool) error {
	s := NewWindowsService(runFn)
	if isInteractive {
		return debug.Run(serviceName, s)
	}
	return svc.Run(serviceName, s)
}

func IsWindowsService() (bool, error) {
	return svc.IsWindowsService()
}
