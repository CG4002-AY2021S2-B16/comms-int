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

// func advHandlerWrapper(dm *devicemanager.DeviceMap) ble.AdvHandler {
// 	return func(a ble.Advertisement) {
// 		detectedDevice := devicemanager.Device{
// 			Address:     a.Addr().String(),
// 			Detected:    time.Now(),
// 			Connectable: a.Connectable(),
// 			Services:    a.Services(),
// 			Name:        utils.CleanString(a.LocalName()),
// 			RSSI:        a.RSSI(),
// 		}

// 		dm.SetDevice(a.Addr().String(), detectedDevice)
// 	}
// }

// func scan(parentCtx context.Context, dm *devicemanager.DeviceMap, finished chan bool) {
// 	fmt.Printf("Scanning for %s...\n", bluno.DefaultTimeout)
// 	chkErr(ble.Scan(parentCtx, true, advHandlerWrapper(dm), nil))
// 	finished <- true
// }

// func chkErr(err error) {
// 	switch errors.Cause(err) {
// 	case nil:
// 	case context.DeadlineExceeded:
// 		fmt.Printf("done\n")
// 	case context.Canceled:
// 		fmt.Printf("canceled\n")
// 	default:
// 		log.Fatalf(err.Error())
// 	}
// }
