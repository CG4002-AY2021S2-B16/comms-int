package appstate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/CG4002-AY2021S2-B16/comms-int/constants"
	"github.com/fatih/color"
	"github.com/go-ble/ble"
)

// BlunoState keeps track of currently running blunos
type BlunoState struct {
	sync.RWMutex
	Name       string
	Address    string
	Status     commsintconfig.BlunoStatus
	UpdateChan chan commsintconfig.BlunoStatus
}

// CreateBlunoState creates and returns a pointer to a BlunoState
func CreateBlunoState(n string, addr string) *BlunoState {
	return &BlunoState{
		Name:       n,
		Address:    addr,
		Status:     commsintconfig.NotConnected,
		UpdateChan: make(chan commsintconfig.BlunoStatus, 1),
	}
}

// UpdateBlunoStatus is to be run within a goroutine to perform updates on Blunos
func (b *BlunoState) UpdateBlunoStatus(ctx context.Context) {
	for {
		select {
		case u := <-b.UpdateChan:
			b.Lock()
			b.Status = u
			b.Unlock()
		case <-ctx.Done():
			return
		}

	}
}

// FetchBlunoStatus is to be run sychronously by a status printing goroutine
func (b *BlunoState) FetchBlunoStatus() commsintconfig.BlunoStatus {
	b.RLock()
	defer b.RUnlock()
	return b.Status
}

// AppState keeps track of the currently running application's state
type AppState struct {
	sync.RWMutex
	S               commsintconfig.State
	MasterCtx       context.Context
	MasterCtxCancel context.CancelFunc
	BlunoStates     []*BlunoState
}

// CreateAppState creates and returns a new app state, with a default state of waiting
func CreateAppState(pCtx context.Context) *AppState {
	ctx, cancel := context.WithCancel(pCtx)
	return &AppState{
		S:               commsintconfig.Waiting,
		MasterCtx:       ble.WithSigHandler(ctx, cancel),
		MasterCtxCancel: cancel,
		BlunoStates:     make([]*BlunoState, 0),
	}
}

// MonitorBlunos is a permanently running goroutine that prints
func (a *AppState) MonitorBlunos() {
	ticker := time.NewTicker(constants.BlunoStatusCheckFreq)

	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	for {
		select {
		case <-ticker.C:
			fmt.Println("---- BLUNO STATUS REPORT ----")

			for _, b := range a.BlunoStates {
				stat := b.FetchBlunoStatus()

				switch stat {
				case commsintconfig.NotConnected:
					fmt.Printf("%s [%s] is currently %s, retrying...\n", b.Address, b.Name, red("not connected"))
				case commsintconfig.NotHandshaked:
					fmt.Printf("%s [%s] is connected, %s...\n", b.Address, b.Name, yellow("attempting handshake"))
				default:
					fmt.Printf("%s [%s] is connected and is %s \n", b.Address, b.Name, green("transmitting packets"))
				}
			}

			fmt.Println("---- END OF BLUNO STATUS REPORT ----")
		}
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
