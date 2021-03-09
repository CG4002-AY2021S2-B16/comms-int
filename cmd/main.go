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
	var us *upstream.IOHandler

	log.Println("waiting for incoming socket communication...")
	us, err = upstream.NewUpstreamConnection()
	if err != nil {
		log.Fatalf("Error occurred during socket setup| err=%s", err)
	}
	outBuf := upstream.CreateOutputBuffer()

	log.Println("Setup successful, sockets successfully connected.")

	// Append all Blunos to the monitoring goroutine
	for _, blno := range constants.RetrieveValidBlunos() {
		newBlnoState := appstate.CreateBlunoState(blno.Name, blno.Address)
		as.BlunoStates = append(as.BlunoStates, newBlnoState)
		blno.StateUpdateChan = newBlnoState.UpdateChan
	}

	// Start monitoring goroutine
	go as.MonitorBlunos()

	// Upon receiving new message, check if app should be running or stopped
	for {
		select {
		case msg := <-us.ReadChan:
			if as.GetState() == commsintconfig.Waiting && msg == constants.UpstreamResumeMsg {
				as.SetState(commsintconfig.Running)

				// Start up goroutines
				go outBuf.EnqueueChannelProcessor(as.MasterCtx)
				go outBuf.DequeueProcessor(as.MasterCtx, us)

				// Start application
				go startApp(as, outBuf.EnqueueBuffer)

			} else if as.GetState() == commsintconfig.Running && msg == constants.UpstreamResumeMsg {
				// Send time sync packets
				us.WriteTimestamp()
			}
		}
		log.Printf("Application is now in state %d", as.GetState())
	}
}

func startApp(as *appstate.AppState, wr func(commsintconfig.Packet)) {
	// Only one BT Client connection can be performed at a time safely via a single device
	// Use a FIFO Semaphore (channel of size 1 in golang to ensure non-starvation in reconnection queue)
	clientCreation := make(chan bool, 1)
	clientCreation <- true
	wg := sync.WaitGroup{}

	for _, bs := range as.BlunoStates {
		go bs.UpdateBlunoStatus(as.MasterCtx)
	}

	for _, b := range constants.RetrieveValidBlunos() {
		// 1 master goroutine per bluno
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func(blno *bluno.Bluno) {
			log.Printf("Master goroutine started for %s, addr=%s, stateUpdateChan=%v", blno.Name, blno.Address, blno.StateUpdateChan)
			var connected bool = false

			for {
				connChan := make(chan bool, 1)
				if !connected {
					go blno.Connect(as.MasterCtx, clientCreation, connChan)

					select {
					case success := <-connChan:
						connected = success
					case <-as.MasterCtx.Done():
						<-connChan // Await safe termination of connect attempt
						log.Println("what:", clientCreation)
						wg.Done()
						return
					}
				} else {
					go blno.Listen(as.MasterCtx, wr, connChan)

					select {
					case success := <-connChan:
						if success {
							wg.Done()
							return
						}
						connected = false
					}
				}
			}
		}(b)
	}

	log.Println("Waiting on goroutines...")
	wg.Wait()
	<-clientCreation // Drain the HCI mutex
	log.Println("All goroutines finalized. Exiting...")
}
