package runtime

import (
	"log"
	"sync"
	"time"
)

type RuntimeState string

const (
	StateStarting         RuntimeState = "STARTING"
	StateInitializing     RuntimeState = "INITIALIZING"
	StateCheckingLlama    RuntimeState = "CHECKING_LLAMA"
	StateCheckingGPU      RuntimeState = "CHECKING_GPU"
	StateReady            RuntimeState = "READY"
	StateLoadingModel     RuntimeState = "LOADING_MODEL"
	StatePreparingCluster RuntimeState = "PREPARING_CLUSTER"
	StateRunning          RuntimeState = "RUNNING"
	StateDraining         RuntimeState = "DRAINING"
	StateDegraded         RuntimeState = "DEGRADED"
	StateRecovering       RuntimeState = "RECOVERING"
	StateStopping         RuntimeState = "STOPPING"
	StateError            RuntimeState = "ERROR"
)

type StateMachine struct {
	mu           sync.RWMutex
	currentState RuntimeState
	lastUpdated  time.Time
	history      []StateTransition
}

type StateTransition struct {
	From      RuntimeState `json:"from"`
	To        RuntimeState `json:"to"`
	Reason    string       `json:"reason"`
	Timestamp time.Time    `json:"timestamp"`
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState: StateStarting,
		lastUpdated:  time.Now(),
		history:      make([]StateTransition, 0),
	}
}

func (sm *StateMachine) GetState() RuntimeState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

func (sm *StateMachine) TransitionTo(newState RuntimeState, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldState := sm.currentState
	if oldState == newState {
		return nil
	}

	sm.currentState = newState
	sm.lastUpdated = time.Now()
	trans := StateTransition{
		From:      oldState,
		To:        newState,
		Reason:    reason,
		Timestamp: sm.lastUpdated,
	}
	sm.history = append(sm.history, trans)

	log.Printf("[Runtime State] Transition %s -> %s (Reason: %s)", oldState, newState, reason)
	return nil
}

func (sm *StateMachine) GetHistory() []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	historyCopy := make([]StateTransition, len(sm.history))
	copy(historyCopy, sm.history)
	return historyCopy
}

func (sm *StateMachine) LastUpdated() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastUpdated
}
