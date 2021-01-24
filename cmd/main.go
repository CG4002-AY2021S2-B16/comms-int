package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/pkg/errors"
	"github.com/rssujay/golang-ble-test/bluno"
	"github.com/rssujay/golang-ble-test/constants"
	"github.com/rssujay/golang-ble-test/devicemanager"
)

func initHCI() {
	d, err := linux.NewDevice()
	if err != nil {
		log.Fatal("Can't create new device", err)
	}
	ble.SetDefaultDevice(d)
}

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds)
	initHCI()
	//devices := devicemanager.DeviceMap{}

	for _, b := range constants.RetrieveValidBlunos() {
		b.Connect()
	}
}

func scan(parentCtx context.Context, dm *devicemanager.DeviceMap, finished chan bool) {
	fmt.Printf("Scanning for %s...\n", bluno.DefaultTimeout)
	chkErr(ble.Scan(parentCtx, true, advHandlerWrapper(dm), nil))
	finished <- true
}

func advHandlerWrapper(dm *devicemanager.DeviceMap) ble.AdvHandler {
	return func(a ble.Advertisement) {
		detectedDevice := devicemanager.Device{
			Address:     a.Addr().String(),
			Detected:    time.Now(),
			Connectable: a.Connectable(),
			Services:    a.Services(),
			Name:        clean(a.LocalName()),
			RSSI:        a.RSSI(),
		}

		dm.SetDevice(a.Addr().String(), detectedDevice)
	}
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}

// reformat string for proper display of hex
func formatHex(instr string) (outstr string) {
	outstr = ""
	for i := range instr {
		if i%2 == 0 {
			outstr += instr[i:i+2] + " "
		}
	}
	return
}

// clean up the non-ASCII characters
func clean(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
}
