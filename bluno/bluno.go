package bluno

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/go-ble/ble"
)

// Bluno represents a BLE device
type Bluno struct {
	Address            string     `json:"address"`
	Name               string     `json:"name"`
	Client             ble.Client `json:"client"`
	ConnectionPriority uint8      `json:"connection_priority"`
}

// DefaultTimeout is the timeout used per connection
const DefaultTimeout time.Duration = 1 * time.Second

// Connect establishes a connection with the physical bluno
// - Remember to close client when done
// - Remember to check disconnected before interacting with channel
// - To be run inside a goroutine
func (b *Bluno) Connect(c chan bool) {
	// Create a channel that times out after 1 second
	ctx := ble.WithSigHandler(context.WithTimeout(
		context.Background(),
		DefaultTimeout,
	))

	client, err := ble.Dial(ctx, ble.NewAddr(b.Address))
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_connection_fail|addr=%s|err=%s", b.Address, err)
		}
		c <- false
	} else {
		if commsintconfig.DebugMode {
			log.Printf("client_connection_succeeded|addr=%s", b.Address)
		}
		b.Client = client
		c <- true // Inform via channel that a connection has been established to calling to goroutine
	}
}

// Listen receives incoming connections from bluno
// - to be called inside a goroutine
func (b *Bluno) Listen(wg *sync.WaitGroup) {
	//h := func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) }
	svcUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoServiceReducedUUID)}
	charUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoCharacteristicReducedUUID)}

	// Isolate the service
	s, err := b.Client.DiscoverServices(svcUUID)
	if err != nil || len(s) != 1 {
		if commsintconfig.DebugMode {
			log.Printf("client_svc_discovery_err|addr=%s|err=%s|len_svcs=%d", b.Address, err, len(s))
		}
		b.Client.CancelConnection()
		return
	}

	// Isolate the characteristic
	c, err := b.Client.DiscoverCharacteristics(charUUID, s[0])
	if err != nil || len(c) != 1 {
		if commsintconfig.DebugMode {
			log.Printf("client_char_discovery_err|addr=%s|err=%s", b.Address, err)
		}
		b.Client.CancelConnection()
		return
	}
	characteristic := c[0]
	characteristic.HandleRead(ble.ReadHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) { log.Printf("Read %s", string(req.Data())) }))
	characteristic.HandleWrite(ble.WriteHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) { log.Printf("Wrote %s", string(req.Data())) }))
	characteristic.HandleNotify(ble.NotifyHandlerFunc(func(req ble.Request, n ble.Notifier) { log.Printf("count: Notification arrived %s", req.Data()) }))
	b.Client.Subscribe(characteristic, false, func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) })

	b.Client.WriteCharacteristic(characteristic, []byte{'e', 'v'}, true)

	for {
		msgCh := make(chan []byte)
		errorCh := make(chan bool)

		go func(ch chan []byte, eCh chan bool) {
			var msg []byte

			msg, err := b.Client.ReadCharacteristic(characteristic)
			if err != nil {
				if commsintconfig.DebugMode {
					log.Printf("client_incoming_msg_err|addr=%s|err=%s", b.Address, err)
				}
				eCh <- true
			} else {
				log.Printf("client_incoming_msg_success|addr=%s|msg=%s", b.Address, msg)
				fmt.Printf("        Value         %x | %q\n", msg, msg)
				ch <- msg
			}
		}(msgCh, errorCh)

		select {
		case <-b.Client.Disconnected():
		case <-errorCh:
			b.Client.CancelConnection()
			return
		case msg := <-msgCh:
			if commsintconfig.DebugMode {
				log.Printf("client_incoming_msg|addr=%s|msg=%s", b.Address, msg)
			}
			b.Client.CancelConnection()
			return
		}
	}
	wg.Done()
}
