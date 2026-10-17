//go:build !windows

package service

import (
	"context"
)

func RunWindowsService(serviceName string, runFn func(ctx context.Context) error, isInteractive bool) error {
	return nil
}

func IsWindowsService() (bool, error) {
	return false, nil
}
