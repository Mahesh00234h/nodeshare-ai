package unit

import (
	"testing"

	"github.com/distributed-ai/node-agent/internal/agent"
)

func TestStateMachineTransitions(t *testing.T) {
	var transitions []string
	sm := agent.NewStateMachine(agent.StateInitializing, func(from, to agent.NodeState) {
		transitions = append(transitions, string(from)+"->"+string(to))
	})

	if sm.GetState() != agent.StateInitializing {
		t.Fatalf("Expected initial state INITIALIZING, got %s", sm.GetState())
	}

	// Valid sequence: INITIALIZING -> CONNECTING -> AUTHENTICATING -> REGISTERING -> READY
	validSeq := []agent.NodeState{
		agent.StateConnecting,
		agent.StateAuthenticating,
		agent.StateRegistering,
		agent.StateReady,
	}

	for _, next := range validSeq {
		if err := sm.TransitionTo(next); err != nil {
			t.Fatalf("Transition to %s failed: %v", next, err)
		}
	}

	if sm.GetState() != agent.StateReady {
		t.Fatalf("Expected state READY, got %s", sm.GetState())
	}

	// Invalid transition check: READY -> INSTALLING should fail
	if err := sm.TransitionTo(agent.StateInstalling); err == nil {
		t.Fatalf("Expected transition READY -> INSTALLING to fail!")
	}
}
