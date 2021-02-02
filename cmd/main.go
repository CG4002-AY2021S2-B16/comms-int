package main

import (
	"context"
	"log"
	"sync"

	"github.com/CG4002-AY2021S2-B16/comms-int/appstate"
	"github.com/CG4002-AY2021S2-B16/comms-int/bluno"
	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/CG4002-AY2021S2-B16/comms-int/constants"
	"github.com/CG4002-AY2021S2-B16/comms-int/upstream"
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func main() {
	if commsintconfig.DebugMode {
		log.SetFlags(log.Ldate | log.Lmicroseconds)
	}

	d, err := linux.NewDevice()
	if err != nil {
		log.Fatal("Can't create new device ", err)
	}
	ble.SetDefaultDevice(d)
	defer d.Stop()

	// Setup application state and upstream connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var as *appstate.AppState = appstate.CreateAppState(ctx)
	var us *upstream.IOHandler = upstream.NewUpstreamConnection()

	log.Println("Setup successful, waiting for incoming socket communication...")
	// Upon receiving new message, check if app should be running or stopped
	for {
		select {
		case msg := <-us.ReadChan:
			if as.GetState() == commsintconfig.Waiting && msg == constants.UpstreamResumeMsg {
				as.SetState(commsintconfig.Running)
				go startApp(as.MasterCtx, us.WriteChan)
			} else if as.GetState() == commsintconfig.Running && msg == constants.UpstreamPauseMsg {
				as.MasterCtxCancel()
				as = appstate.CreateAppState(ctx)
				as.SetState(commsintconfig.Waiting)
			}
		}
	}
}

func startApp(ctx context.Context, wc chan commsintconfig.Packet) {
	wg := sync.WaitGroup{}

	for _, b := range constants.RetrieveValidBlunos() {
		// 1 master goroutine per bluno
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func(blno *bluno.Bluno) {
			for {
				success := blno.Connect(ctx)
				if success {
					if listenCancel := blno.Listen(ctx, &wg, wc); listenCancel {
						return
					}
				}
			}
		}(&b)
	}

	log.Println("Waiting on goroutines...")
	wg.Wait()
	log.Println("All goroutines finalized. Exiting...")
}
