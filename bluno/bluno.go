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
			if len(s) > 0 {
				for i, j := range s {
					fmt.Println("wew", i, j.Characteristics, j.UUID.Len(), j.UUID.String(), j.UUID.Equal(svcUUID[0]))
				}
			}
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
				log.Printf("client_incoming_msg_success|addr=%s|err=%s", b.Address, msg)
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

// func explore(cln ble.Client, p *ble.Profile) error {
// 	sub := 0 * time.Second
// 	fmt.Println("TEST", p)
// 	fmt.Println(p.Services)

// 	for _, s := range p.Services {
// 		fmt.Printf("    Service: %s %s, Handle (0x%02X)\n", s.UUID, ble.Name(s.UUID), s.Handle)

// 		for _, c := range s.Characteristics {
// 			fmt.Printf("      Characteristic: %s %s, Property: 0x%02X (%s), Handle(0x%02X), VHandle(0x%02X)\n",
// 				c.UUID, ble.Name(c.UUID), c.Property, propString(c.Property), c.Handle, c.ValueHandle)
// 			if (c.Property & ble.CharRead) != 0 {
// 				b, err := cln.ReadCharacteristic(c)
// 				if err != nil {
// 					fmt.Printf("Failed to read characteristic: %s\n", err)
// 					continue
// 				}
// 				fmt.Printf("        Value         %x | %q\n", b, b)
// 			}

// 			for _, d := range c.Descriptors {
// 				fmt.Printf("        Descriptor: %s %s, Handle(0x%02x)\n", d.UUID, ble.Name(d.UUID), d.Handle)
// 				b, err := cln.ReadDescriptor(d)
// 				if err != nil {
// 					fmt.Printf("Failed to read descriptor: %s\n", err)
// 					continue
// 				}
// 				fmt.Printf("        Value         %x | %q\n", b, b)
// 			}

// 			if sub != 0 {
// 				// Don't bother to subscribe the Service Changed characteristics.
// 				if c.UUID.Equal(ble.ServiceChangedUUID) {
// 					continue
// 				}

// 				// Don't touch the Apple-specific Service/Characteristic.
// 				// Service: D0611E78BBB44591A5F8487910AE4366
// 				// Characteristic: 8667556C9A374C9184ED54EE27D90049, Property: 0x18 (WN),
// 				//   Descriptor: 2902, Client Characteristic Configuration
// 				//   Value         0000 | "\x00\x00"
// 				if c.UUID.Equal(ble.MustParse("8667556C9A374C9184ED54EE27D90049")) {
// 					continue
// 				}

// 				if (c.Property & ble.CharNotify) != 0 {
// 					fmt.Printf("\n-- Subscribe to notification for %s --\n", sub)
// 					h := func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) }
// 					if err := cln.Subscribe(c, false, h); err != nil {
// 						log.Fatalf("subscribe failed: %s", err)
// 					}
// 					time.Sleep(sub)
// 					if err := cln.Unsubscribe(c, false); err != nil {
// 						log.Fatalf("unsubscribe failed: %s", err)
// 					}
// 					fmt.Printf("-- Unsubscribe to notification --\n")
// 				}
// 				if (c.Property & ble.CharIndicate) != 0 {
// 					fmt.Printf("\n-- Subscribe to indication of %s --\n", sub)
// 					h := func(req []byte) { fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) }
// 					if err := cln.Subscribe(c, true, h); err != nil {
// 						log.Fatalf("subscribe failed: %s", err)
// 					}
// 					time.Sleep(sub)
// 					if err := cln.Unsubscribe(c, true); err != nil {
// 						log.Fatalf("unsubscribe failed: %s", err)
// 					}
// 					fmt.Printf("-- Unsubscribe to indication --\n")
// 				}
// 			}
// 		}
// 		fmt.Printf("\n")
// 	}
// 	return nil
// }

// func propString(p ble.Property) string {
// 	var s string
// 	for k, v := range map[ble.Property]string{
// 		ble.CharBroadcast:   "B",
// 		ble.CharRead:        "R",
// 		ble.CharWriteNR:     "w",
// 		ble.CharWrite:       "W",
// 		ble.CharNotify:      "N",
// 		ble.CharIndicate:    "I",
// 		ble.CharSignedWrite: "S",
// 		ble.CharExtended:    "E",
// 	} {
// 		if p&k != 0 {
// 			s += v
// 		}
// 	}
// 	return s
// }
