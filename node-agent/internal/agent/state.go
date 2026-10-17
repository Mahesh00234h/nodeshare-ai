package agent

import (
	"fmt"
	"sync"
)

type NodeState string

const (
	StateInstalling     NodeState = "INSTALLING"
	StateInitializing   NodeState = "INITIALIZING"
	StateConnecting     NodeState = "CONNECTING"
	StateAuthenticating NodeState = "AUTHENTICATING"
	StateRegistering    NodeState = "REGISTERING"
	StateReady          NodeState = "READY"
	StateBusy           NodeState = "BUSY"
	StateDraining       NodeState = "DRAINING"
	StateDegraded       NodeState = "DEGRADED"
	StateDisconnected   NodeState = "DISCONNECTED"
	StateRecovering     NodeState = "RECOVERING"
	StateStopping       NodeState = "STOPPING"
	StateError          NodeState = "ERROR"
)

type StateMachine struct {
	mu           sync.RWMutex
	current      NodeState
	onTransition func(from, to NodeState)
}

func NewStateMachine(initial NodeState, onTransition func(from, to NodeState)) *StateMachine {
	return &StateMachine{
		current:      initial,
		onTransition: onTransition,
	}
}

func (sm *StateMachine) GetState() NodeState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

func (sm *StateMachine) TransitionTo(next NodeState) error {
	sm.mu.Lock()
	from := sm.current

	if !isValidTransition(from, next) {
		sm.mu.Unlock()
		return fmt.Errorf("invalid state transition from %s to %s", from, next)
	}

	sm.current = next
	sm.mu.Unlock()

	if sm.onTransition != nil {
		sm.onTransition(from, next)
	}
	return nil
}

func isValidTransition(from, to NodeState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateInstalling:
		return to == StateInitializing || to == StateError
	case StateInitializing:
		return to == StateConnecting || to == StateError
	case StateConnecting:
		return to == StateAuthenticating || to == StateDisconnected || to == StateError
	case StateAuthenticating:
		return to == StateRegistering || to == StateDisconnected || to == StateError
	case StateRegistering:
		return to == StateReady || to == StateDisconnected || to == StateError
	case StateReady:
		return to == StateBusy || to == StateDraining || to == StateDegraded || to == StateDisconnected || to == StateStopping || to == StateError
	case StateBusy:
		return to == StateReady || to == StateDraining || to == StateDegraded || to == StateDisconnected || to == StateStopping || to == StateError
	case StateDraining:
		return to == StateStopping || to == StateDisconnected || to == StateError
	case StateDegraded:
		return to == StateReady || to == StateDraining || to == StateDisconnected || to == StateStopping || to == StateError
	case StateDisconnected:
		return to == StateConnecting || to == StateRecovering || to == StateStopping || to == StateError
	case StateRecovering:
		return to == StateConnecting || to == StateReady || to == StateError
	case StateStopping:
		return to == StateError
	case StateError:
		return to == StateInitializing || to == StateConnecting || to == StateStopping
	}
	return false
}
