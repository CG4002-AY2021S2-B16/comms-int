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

	// Lock is used to create clients one at a time
	var clientCreation sync.Mutex

	// Setup application state and upstream connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var as *appstate.AppState = appstate.CreateAppState(ctx)
	var us *upstream.IOHandler

	log.Println("waiting for incoming socket communication...")
	us, err = upstream.NewUpstreamConnection()
	if err != nil {
		log.Fatalf("Error occurred during socket setup| err=%s", err)
	}
	outBuf := upstream.CreateOutputBuffer()

	// Start up goroutines
	go outBuf.EnqueueChannelProcessor()
	go outBuf.DequeueProcessor(us)
	log.Println("Setup successful, sockets successfully connected.")

	// Upon receiving new message, check if app should be running or stopped
	for {
		select {
		case msg := <-us.ReadChan:
			if as.GetState() == commsintconfig.Waiting && msg == constants.UpstreamResumeMsg {
				as.SetState(commsintconfig.Running)
				go startApp(as.MasterCtx, outBuf.EnqueueBuffer, &clientCreation)
			} else if as.GetState() == commsintconfig.Running && msg == constants.UpstreamPauseMsg {
				as.MasterCtxCancel()
				as = appstate.CreateAppState(ctx)
				as.SetState(commsintconfig.Waiting)
			}
		}
	}
}

func startApp(ctx context.Context, wr func(commsintconfig.Packet), m *sync.Mutex) {
	wg := sync.WaitGroup{}

	for _, b := range constants.RetrieveValidBlunos() {
		// 1 master goroutine per bluno
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func(blno bluno.Bluno) {
			log.Printf("Master goroutine started for %s, addr=%s", blno.Name, blno.Address)
			for {
				success := blno.Connect(ctx, m)
				if success {
					if listenCancel := blno.Listen(ctx, &wg, wr); listenCancel {
						return
					}
				}
			}
		}(b)
	}

	log.Println("Waiting on goroutines...")
	wg.Wait()
	log.Println("All goroutines finalized. Exiting...")
}
