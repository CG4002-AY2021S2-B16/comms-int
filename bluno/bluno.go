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
func (b *Bluno) Connect() bool {
	// Create a context that times out after 1 second
	ctx := ble.WithSigHandler(context.WithTimeout(
		context.Background(),
		DefaultTimeout,
	))
	client, err := ble.Dial(ctx, ble.NewAddr(b.Address))
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_connection_fail|addr=%s|err=%s", b.Address, err)
		}
		return false
	}
	if commsintconfig.DebugMode {
		log.Printf("client_connection_succeeded|addr=%s", b.Address)
	}

	b.Client = client
	return true
}

// Listen receives incoming connections from bluno
// - to be called inside a goroutine
func (b *Bluno) Listen(wg *sync.WaitGroup) bool {
	defer b.Client.CancelConnection()

	// Create a context that is sensitive to kill signal
	parentCtx := ble.WithSigHandler(context.WithCancel(context.Background()))

	svcUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoServiceReducedUUID), ble.MustParse(commsintconfig.BlunoServiceUUID)}
	charUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoCharacteristicReducedUUID), ble.MustParse(commsintconfig.BlunoCharacteristicUUID)}

	// Isolate the service
	s, err := b.Client.DiscoverServices(svcUUID)
	if err != nil || len(s) != 1 {
		if commsintconfig.DebugMode {
			log.Printf("client_svc_discovery_err|addr=%s|err=%s|len_svcs=%d", b.Address, err, len(s))
		}
		return false
	}

	// Isolate the characteristic
	c, err := b.Client.DiscoverCharacteristics(charUUID, s[0])
	if err != nil || len(c) != 1 {
		if commsintconfig.DebugMode {
			log.Printf("client_char_discovery_err|addr=%s|err=%s|num_characteristics=%d", b.Address, err, len(c))
		}
		return false
	}

	// Add 2902, because its missing from Bluno: https://www.dfrobot.com/forum/viewtopic.php?t=2035
	characteristic := c[0]
	customDescriptor := ble.NewDescriptor(ble.UUID16(commsintconfig.ClientCharacteristicConfig))
	customDescriptor.Handle = commsintconfig.ClientCharacteristicConfigHandle
	characteristic.CCCD = customDescriptor

	err = b.Client.Subscribe(characteristic, false, func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) })
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_subscription_err|addr=%s|err=%s", b.Address, err)
		}
		return false
	}
	defer b.Client.ClearSubscriptions()

	// Handshake
	log.Println("Handshaking")
	b.Client.WriteCharacteristic(characteristic, []byte{'A', 'B', 'C'}, true)
	log.Println("Handshaking done")

	// Read
	for {
		msgCh := make(chan []byte, 1)
		errorCh := make(chan bool, 1)

		select {
		case <-b.Client.Disconnected():
		case <-errorCh:
			log.Printf("exiting listen")
			return false
		case <-parentCtx.Done():
		case <-time.After(20 * time.Second):
			log.Printf("force exiting listen")
			wg.Done()
			return true
		case msg := <-msgCh:
			if commsintconfig.DebugMode {
				log.Printf("client_incoming_msg|addr=%s|msg=%s", b.Address, string(msg))
			}
		}
	}
}
