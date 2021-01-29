package bluno

import (
	"container/list"
	"context"
	"encoding/binary"
	"log"
	"sync"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/go-ble/ble"
)

// Bluno represents a BLE device
type Bluno struct {
	Address                string     `json:"address"`
	Name                   string     `json:"name"`
	Client                 ble.Client `json:"client"`
	ConnectionPriority     uint8      `json:"connection_priority"`
	PacketsReceived        uint32     `json:"packets_received"`
	PacketsImmSuccess      uint32     `json:"packets_immediate_success"`
	PacketsInvalidType     uint32     `json:"packets_invalid_type"`
	PacketsIncorrectLength uint32     `json:"packets_incorrect_length"`
	PacketsReconciled      uint32     `json:"packets_reconciled"`
	StartTime              time.Time  `json:"start_time"`
	Buffer                 *list.List `json:"response_buffer"`
}

// DefaultTimeout is the timeout for establishing connection
const DefaultTimeout time.Duration = 1 * time.Second

// ConnectionTimeout is the timeout/duration for an active connection
// Should equal infinity in non-testing use
const ConnectionTimeout = 20 * time.Second

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

	b.SetClient(&client)
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

	err = b.Client.Subscribe(characteristic, false, b.parseResponse)
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_subscription_err|addr=%s|err=%s", b.Address, err)
		}
		return false
	}
	defer b.Client.ClearSubscriptions()

	// Handshake
	log.Printf("Handshake initiated with %s (%s)", b.Name, b.Address)
	b.Client.WriteCharacteristic(characteristic, []byte{commsintconfig.InitHandshakeSymbol}, true)

	// Read
	for {
		msgCh := make(chan []byte, 1)
		errorCh := make(chan bool, 1)

		select {
		case <-b.Client.Disconnected():
		case <-errorCh:
			log.Printf("client_connection_terminated|force=false|packets received=%d", b.PacketsReceived)
			b.PrintStats()
			return false
		case <-parentCtx.Done():
		case <-time.After(ConnectionTimeout):
			log.Printf("client_connection_terminated|force=true|packets received=%d", b.PacketsReceived)
			b.PrintStats()
			wg.Done()
			return true
		case msg := <-msgCh:
			if commsintconfig.DebugMode {
				log.Printf("client_incoming_msg|addr=%s|msg=%s", b.Address, string(msg))
			}
		}
	}
}

func (b *Bluno) parseResponse(resp []byte) {
	b.PacketsReceived++
	if commsintconfig.DebugMode {
		log.Printf("Received packet from bluno: [ % X ]\n", resp)
	}

	var p commsintconfig.Packet = commsintconfig.Packet{Type: commsintconfig.Invalid}

	if len(resp) != commsintconfig.ExpectedPacketSize {
		b.PacketsIncorrectLength++
		if commsintconfig.DebugMode {
			log.Printf("Received packet from bluno of incorrect size = %d: [ % X ]\n", len(resp), resp)
		}
		// return // discard

		var reconciled bool
		p, reconciled = b.ReconcilePacket(resp)
		if !reconciled {
			return
		}
		b.PacketsReconciled++
	} else {
		p = constructPacket(resp)
	}

	if commsintconfig.DebugMode {
		log.Printf("Response parsed|%+v\n", p)
	}

	switch p.Type {
	case commsintconfig.Ack:
		log.Printf("Handshake successful with %s (%s)", b.Name, b.Address)
	case commsintconfig.Invalid:
		b.PacketsInvalidType++
	default:
		b.PacketsImmSuccess++
	}

}

// determinePacketType returns the packet's type based on the first byte
func determinePacketType(d []byte) commsintconfig.PacketType {
	if d[0] == commsintconfig.RespHandshakeSymbol {
		return commsintconfig.Ack
	} else if d[0] == commsintconfig.RespDataSymbol {
		return commsintconfig.Data
	}
	return commsintconfig.Invalid
}

// twoByteToNum converts 2 consecutive bytes into a uint16
// it assumes the bytes are arranged in little endian format
func twoByteToNum(d []byte, start uint8) uint16 {
	return binary.BigEndian.Uint16(d[start : start+2])
}

func constructPacket(resp []byte) commsintconfig.Packet {
	return commsintconfig.Packet{
		Type:  determinePacketType(resp),
		X:     twoByteToNum(resp, 1),
		Y:     twoByteToNum(resp, 3),
		Z:     twoByteToNum(resp, 5),
		Yaw:   twoByteToNum(resp, 7),
		Pitch: twoByteToNum(resp, 9),
		Roll:  twoByteToNum(resp, 11),
	}
}

// PrintStats prints out transmission statistics for a given bluno
// Each BLE 4.0 packet is between 31 (best) - 41 (worst case) bytes
// -> implies each packet is between 248 - 328 bits
// -> implies up to 351 - 464 packets can be received per second
func (b *Bluno) PrintStats() {
	elapsedTime := time.Now().Sub(b.StartTime).Seconds()
	log.Printf(
		"print_stats_pre_reconciliation|successful_packets=%d|elapsed_time=%f|effective_packets_per_second=%f",
		b.PacketsImmSuccess,
		elapsedTime,
		float64(b.PacketsImmSuccess)/elapsedTime,
	)
	log.Printf(
		"print_stats_pre_reconciliation|success_ratio=%f|incorrect_length_ratio=%f|invalid_data_ratio=%f",
		float64(b.PacketsImmSuccess)/float64(b.PacketsReceived),
		float64(b.PacketsIncorrectLength)/float64(b.PacketsReceived),
		float64(b.PacketsInvalidType)/float64(b.PacketsReceived),
	)

	// Print absolute numbers
	log.Printf(
		"print_stats_post_reconciliation|packets_received=%d|immediately_successful_packets=%d|reconciled_packets=%d|invalid_packets=%d|incorrect_length_packets=%d",
		b.PacketsReceived,
		b.PacketsImmSuccess,
		b.PacketsReconciled,
		b.PacketsInvalidType,
		b.PacketsIncorrectLength,
	)

	// Reconciliation causes extra 1 InvalidType, 1 IncorrectLength and 1 PacketReceived
	adjIncorrectLength := float64(b.PacketsIncorrectLength - 2*b.PacketsReconciled)
	adjSuccessfulPackets := float64(b.PacketsImmSuccess + b.PacketsReconciled)

	log.Printf(
		"print_stats_post_reconciliation|successful_packets=%f|elapsed_time=%f|effective_packets_per_second=%f",
		adjSuccessfulPackets,
		elapsedTime,
		adjSuccessfulPackets/elapsedTime,
	)

	log.Printf(
		"print_stats_post_reconciliation|success_ratio=%f|incorrect_length_ratio=%f|invalid_data_ratio=%f",
		adjSuccessfulPackets/float64(b.PacketsReceived),
		float64(adjIncorrectLength)/float64(b.PacketsReceived),
		float64(b.PacketsInvalidType)/float64(b.PacketsReceived),
	)

}

// SetClient attaches an active client to the given bluno, and resets its statistics e.g. transmission counters
func (b *Bluno) SetClient(c *ble.Client) {
	b.Client = *c
	b.PacketsInvalidType = 0
	b.PacketsIncorrectLength = 0
	b.PacketsReceived = 0
	b.StartTime = time.Now()
	b.Buffer = list.New()
}
