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
	"github.com/rssujay/golang-ble-test/devicemanager"
)

var (
	device = "laptop"
	du     = 5 * time.Second
	dup    = true
)

func main() {
	devices := devicemanager.DeviceMap{
		Dm: make(map[string]devicemanager.Device),
	}
	central, err := linux.NewDeviceWithName(device)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}

	ble.SetDefaultDevice(central)

	// Scan for specified duration, or until interrupted by user.
	finished := make(chan bool)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), du))
	go scan(ctx, &devices, finished)

	// Print Entries
	<-finished
	devices.PrintEntries()
}

func scan(parentCtx context.Context, dm *devicemanager.DeviceMap, finished chan bool) {
	fmt.Printf("Scanning for %s...\n", du)
	chkErr(ble.Scan(parentCtx, dup, advHandlerWrapper(dm), nil))
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
