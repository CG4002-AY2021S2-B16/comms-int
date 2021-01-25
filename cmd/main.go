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

func initHCI() *linux.Device {
	d, err := linux.NewDevice()
	if err != nil {
		log.Fatal("Can't create new device ", err)
	}
	ble.SetDefaultDevice(d)
	return d
}

func main() {
	if commsintconfig.DebugMode {
		log.SetFlags(log.Ldate | log.Lmicroseconds)
	}
	d := initHCI()
	//devices := devicemanager.DeviceMap{}

	// 1 mastergoroutine per bluno
	// - manages connecting
	wg := sync.WaitGroup{}

	for _, b := range constants.RetrieveValidBlunos() {
		// Asynchronously establish connection to Bluno and listen to incoming messages from peripheral
		wg.Add(1)

		go func(blno *bluno.Bluno) {
			// A channel is used to block until successful connection
			complete := make(chan bool)
			go blno.Connect(complete)
			success := <-complete
			if success {
				go blno.Listen(&wg)
			} else {
				wg.Done()
			}
		}(&b)
	}

	wg.Wait()
	d.Stop()
	log.Println("Stop called")
}
