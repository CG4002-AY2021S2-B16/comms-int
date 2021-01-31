package appstate

import (
	"context"
	"sync"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/go-ble/ble"
)

// AppState keeps track of the currently running application's state
type AppState struct {
	sync.RWMutex
	S               commsintconfig.State
	MasterCtx       context.Context
	MasterCtxCancel context.CancelFunc
}

// CreateAppState creates and returns a new app state, with a default state of waiting
func CreateAppState(pCtx context.Context) *AppState {
	ctx, cancel := context.WithCancel(pCtx)
	return &AppState{
		S:               commsintconfig.Waiting,
		MasterCtx:       ble.WithSigHandler(ctx, cancel),
		MasterCtxCancel: cancel,
	}
}

// HaltAppState stops the current app
func (a *AppState) HaltAppState() {
	a.MasterCtxCancel()
}

// GetState retrieves the app's instantaneous state
func (a *AppState) GetState() commsintconfig.State {
	a.RLock()
	defer a.RUnlock()
	return a.S
}

// SetState sets the app's state at a particular point in time
func (a *AppState) SetState(state commsintconfig.State) {
	a.Lock()
	defer a.Unlock()
	a.S = state
}
