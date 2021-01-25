package main

import (
	"log"
	"sync"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/CG4002-AY2021S2-B16/comms-int/constants"
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func initHCI() {
	d, err := linux.NewDevice()
	if err != nil {
		log.Fatal("Can't create new device ", err)
	}
	ble.SetDefaultDevice(d)
}

func main() {
	if commsintconfig.DebugMode {
		log.SetFlags(log.Ldate | log.Lmicroseconds)
	}
	initHCI()
	//devices := devicemanager.DeviceMap{}

	// 1 mastergoroutine per bluno
	// - manages connecting
	wg := sync.WaitGroup{}

	for _, b := range constants.RetrieveValidBlunos() {
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func() {
			// A channel is used to block until successful connection
			complete := make(chan bool)
			go b.Connect(complete)
			success := <-complete
			if success {
				go b.Listen(&wg)
			} else {
				wg.Done()
			}
		}()
	}

	wg.Wait()
}
