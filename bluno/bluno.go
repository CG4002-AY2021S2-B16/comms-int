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
	Num                    uint8      `json:"num"`
	Client                 ble.Client `json:"client"`
	PacketsReceived        uint32     `json:"packets_received"`
	HandshakeAcknowledged  bool       `json:"handshake_acknowledged"`
	HandShakeInit          time.Time  `json:"handshake_sent_at"`
	HandshakedAt           time.Time  `json:"handshake_received_at"`
	LastPacketReceivedAt   time.Time  `json:"last_packet_received_at"`
	PacketsImmSuccess      uint32     `json:"packets_immediate_success"`
	PacketsInvalidType     uint32     `json:"packets_invalid_type"`
	PacketsIncorrectLength uint32     `json:"packets_incorrect_length"`
	PacketsReconciled      uint32     `json:"packets_reconciled"`
	StartTime              time.Time  `json:"start_time"`
	Buffer                 *list.List `json:"response_buffer"`
	StateUpdateChan        chan commsintconfig.BlunoStatus
}

// Connect establishes a connection with the physical bluno
// - Remember to close client when done
// - Remember to check disconnected before interacting with channel
// - To be run inside a goroutine
func (b *Bluno) Connect(pCtx context.Context, m chan bool) bool {
	// Dial to Bluno
	<-m
	timedCtx, cancel := context.WithTimeout(pCtx, commsintconfig.ConnectionEstablishTimeout)
	defer cancel()
	client, err := ble.Dial(timedCtx, ble.NewAddr(b.Address))
	m <- true
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_connection_fail|addr=%s|err=%s", b.Address, err)
		}
		time.Sleep(2 * time.Second) // Sleep fixed duration to induce predictability and allow other connection attempts
		return false
	}

	b.SetClient(&client)
	b.StateUpdateChan <- commsintconfig.NotHandshaked
	if commsintconfig.DebugMode {
		log.Printf("client_connection_succeeded|addr=%s", b.Address)
	}
	return true
}

// Listen receives incoming connections from bluno
// - to be called inside a goroutine
func (b *Bluno) Listen(pCtx context.Context, wg *sync.WaitGroup, wr func(commsintconfig.Packet)) bool {
	defer b.Client.CancelConnection()

	// Perform targeted find of characteristic
	svcUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoServiceReducedUUID), ble.MustParse(commsintconfig.BlunoServiceUUID)}
	charUUID := []ble.UUID{ble.UUID16(commsintconfig.BlunoCharacteristicReducedUUID), ble.MustParse(commsintconfig.BlunoCharacteristicUUID)}
	//commandUUID := []ble.UUID{ble.UUID16(commsintconfig.CommandCharacteristicReducedUUID), ble.MustParse(commsintconfig.CommandCharacteristicUUID)}

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

	// Add 2902 descriptor, because its missing from Bluno: https://www.dfrobot.com/forum/viewtopic.php?t=2035
	// Go requires BLE spec conformity for maintaining connections, otherwise errors
	characteristic := c[0]
	customDescriptor := ble.NewDescriptor(ble.UUID16(commsintconfig.ClientCharacteristicConfig))
	customDescriptor.Handle = commsintconfig.ClientCharacteristicConfigHandle
	characteristic.CCCD = customDescriptor

	hsFail := make(chan bool, 1)

	// Subscribe to notifications
	err = b.Client.Subscribe(characteristic, false, b.parseResponse(hsFail, wr))
	if err != nil {
		if commsintconfig.DebugMode {
			log.Printf("client_subscription_err|addr=%s|err=%s", b.Address, err)
		}
		return false
	}

	// Handshake
	log.Printf("Handshake initiated with %s (%s) service=%s char=%s", b.Name, b.Address, s[0].UUID.String(), characteristic.UUID.String())
	toSend := []byte{commsintconfig.InitHandshakeSymbol, byte('\r'), '\n'}
	err = b.Client.WriteCharacteristic(characteristic, toSend, false)
	if err != nil {
		log.Printf("write_handshake|err=%s", err)
	}
	b.HandShakeInit = time.Now()
	log.Printf("Handshake sent to %s (%s)|[ %X ]", b.Name, b.Address, toSend)

	// Start tickers
	tickChan := time.NewTicker(commsintconfig.ConnectionLivenessCheckInterval)
	establishTickChan := time.NewTicker(commsintconfig.ConnectionEstablishTimeout)

	// Read
	for {
		select {
		case <-b.Client.Disconnected():
			b.StateUpdateChan <- commsintconfig.NotConnected
			log.Printf("client_connection_disconnected|addr=%s", b.Address)
			//b.PrintStats()
			return false
		case <-hsFail:
			log.Printf("client_handshake_fail|addr=%s", b.Address)
			b.Client.Unsubscribe(characteristic, false)
			return false

		case t := <-tickChan.C:
			diff := t.Sub(b.LastPacketReceivedAt)
			if b.HandshakeAcknowledged && diff >= commsintconfig.ConnectionLivenessTimeout {
				//b.PrintStats()
				log.Printf(
					"client_connection_terminated|liveness_ticker_exceed|packets received=%d|lastPacketReceived=%s|curr_t=%s",
					b.PacketsReceived,
					b.LastPacketReceivedAt,
					t,
				)
				b.StateUpdateChan <- commsintconfig.NotConnected
				return false
			}
		case et := <-establishTickChan.C:
			log.Printf("time_check|et=%s", et.String())
			diff := et.Sub(b.LastPacketReceivedAt)
			if !b.HandshakeAcknowledged && diff >= 5*commsintconfig.ConnectionEstablishTimeout {
				log.Printf(
					"client_connection_terminated|establish_ticker_exceed|packets received=%d|lastPacketReceived=%s|curr_t=%s",
					b.PacketsReceived,
					b.LastPacketReceivedAt,
					et,
				)
				b.StateUpdateChan <- commsintconfig.NotConnected
				return false
			}
			// } else if !b.HandshakeAcknowledged && diff >= commsintconfig.ConnectionEstablishTimeout {
			// 	// We attempt a reset of built in BLE chip
			// 	log.Printf("Attempting reset...")

			// 	// Isolate the characteristic
			// 	cc, err := b.Client.DiscoverCharacteristics(commandUUID, s[0])
			// 	if err != nil || len(cc) != 2 { // two because we already discovered dfb1 earlier
			// 		if commsintconfig.DebugMode {
			// 			log.Printf("command_char_discovery_err|addr=%s|err=%s|num_characteristics=%d", b.Address, err, len(cc))
			// 		}
			// 		return false
			// 	}

			// 	// Add 2902
			// 	commandCharacteristic := cc[1]
			// 	commandCustomDescriptor := ble.NewDescriptor(ble.UUID16(commsintconfig.ClientCharacteristicConfig))
			// 	commandCustomDescriptor.Handle = commsintconfig.ClientCharacteristicConfigHandle
			// 	commandCharacteristic.CCCD = customDescriptor

			// 	// Subscribe
			// 	err = b.Client.Subscribe(commandCharacteristic, false, b.parseResponse)
			// 	if err != nil {
			// 		if commsintconfig.DebugMode {
			// 			log.Printf("command_subscription_err|addr=%s|err=%s", b.Address, err)
			// 		}
			// 		return false
			// 	}
			// 	defer b.Client.Unsubscribe(commandCharacteristic, false)

			// 	// Write reset BLE AT-Command
			// 	byteReset := []byte(commsintconfig.BLEResetString)
			// 	b.Client.WriteCharacteristic(commandCharacteristic, byteReset, false)
			// 	log.Printf("command_reset_sent|[ % X ]", byteReset)
			// }
		case <-pCtx.Done():
			log.Printf("client_connection_terminated|force=true|packets received=%d", b.PacketsReceived)
			//b.PrintStats()
			b.StateUpdateChan <- commsintconfig.NotConnected
			b.Client.ClearSubscriptions()
			wg.Done()
			return true
		}
	}
}

func (b *Bluno) parseResponse(hsFail chan bool, wr func(commsintconfig.Packet)) func([]byte) {
	return func(resp []byte) {
		b.LastPacketReceivedAt = time.Now()
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

			var reconciled bool
			p, reconciled = b.ReconcilePacket(resp)
			if !reconciled {
				return
			}
			b.PacketsReconciled++
		} else {
			p = constructPacket(b, resp)
		}

		log.Printf("Packet processed %+v for resp [ % X ]\n", p, resp)

		switch p.Type {
		case commsintconfig.Ack:
			log.Printf("Handshake successful with %s (%s)", b.Name, b.Address)
			b.StateUpdateChan <- commsintconfig.Transmitting
			b.HandshakeAcknowledged = true
		case commsintconfig.Invalid:
			b.PacketsInvalidType++
		default:
			if b.HandshakeAcknowledged == false {
				hsFail <- true
			} else {
				b.PacketsImmSuccess++
				wr(p) // Send to output buffer
			}
		}
	}
}

// determinePacketType returns the packet's type based on the 3rd and 4th bit of the 18th byte of the response
func determinePacketType(d []byte) commsintconfig.PacketType {
	if d[17]|commsintconfig.RespHandshakeSymbol == commsintconfig.RespHandshakeSymbol {
		return commsintconfig.Ack
	} else if d[17]&commsintconfig.RespDataSymbol == commsintconfig.RespDataSymbol {
		return commsintconfig.Data
	}
	return commsintconfig.Invalid
}

// twoByteToNum converts 2 consecutive bytes into a uint16
// it assumes the bytes are arranged in little endian format
func twoByteToNum(d []byte, start uint8) int16 {
	return int16(binary.LittleEndian.Uint16(d[start : start+2]))
}

// calculateChecksum takes a complete packet and finds its checksum
// returns true if checksum passes otherwise false
func calculateChecksum(d []byte) bool {
	givenChecksum := d[commsintconfig.ExpectedPacketSize-1]
	var c byte = 0x00

	for i := 0; i < commsintconfig.ExpectedPacketSize-1; i++ {
		c ^= d[i]
	}

	if commsintconfig.DebugMode {
		log.Printf("Checksum calculated for [ % X ] : %t \n", d, givenChecksum == c)
	}
	return givenChecksum == c
}

// formTimestamp takes in a bluno, a packet and performs unix timestamp creation
func formTimestamp(b *Bluno, resp []byte, start uint8) time.Time {
	d := time.Millisecond * time.Duration(binary.LittleEndian.Uint32(resp[start:start+4]))
	return b.HandShakeInit.Add(d)
}

// getMuscleSensorReading takes in a bluno, a packet and extracts the muscle sensor reading
// from the lower 8 bits and the 1st and 2nd bit from the 18th byte
func getMuscleSensorReading(b *Bluno, resp []byte, lower uint8, upper uint8) uint16 {
	l := resp[lower]
	h := resp[upper] & commsintconfig.ADCmask
	log.Printf("[ % X ] -> [ % X ] [ % X ]", resp[upper], h, l)
	return binary.LittleEndian.Uint16([]byte{l, h})
}

func constructPacket(b *Bluno, resp []byte) commsintconfig.Packet {
	if !calculateChecksum(resp) {
		return commsintconfig.Packet{Type: commsintconfig.Invalid}
	}

	t := determinePacketType(resp)
	if t == commsintconfig.Ack {
		b.HandshakedAt = time.Now()
	}

	return commsintconfig.Packet{
		Timestamp:    formTimestamp(b, resp, 0).UnixNano() / int64(time.Millisecond),
		X:            twoByteToNum(resp, 4),
		Y:            twoByteToNum(resp, 6),
		Z:            twoByteToNum(resp, 8),
		Pitch:        twoByteToNum(resp, 10),
		Roll:         twoByteToNum(resp, 12),
		Yaw:          twoByteToNum(resp, 14),
		MuscleSensor: getMuscleSensorReading(b, resp, 16, 17),
		Type:         t,
		BlunoNumber:  b.Num,
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
	b.PacketsImmSuccess = 0
	b.PacketsReconciled = 0
	b.HandshakeAcknowledged = false
	b.StartTime = time.Now()
	b.LastPacketReceivedAt = time.Now()
	b.Buffer = list.New()
}
