package main

import (
	"log"
	"sync"

	"github.com/CG4002-AY2021S2-B16/comms-int/bluno"
	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/CG4002-AY2021S2-B16/comms-int/constants"
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

	wg := sync.WaitGroup{}

	for _, b := range constants.RetrieveValidBlunos() {
		// 1 master goroutine per bluno
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func(blno *bluno.Bluno) {
			for {
				success := blno.Connect()
				if success {
					if listenCancel := blno.Listen(&wg); listenCancel {
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
