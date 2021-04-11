package bluno

import (
	"container/list"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/go-ble/ble"
)

// Bluno represents a BLE device
type Bluno struct {
	Address                string     `json:"address"`
	Name                   string     `json:"name"`
	Num                    uint8      `json:"num"`
	User                   string     `json:"user"`
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
	LeftIndication         uint8
	RightIndication        uint8
	NotSentIndication      uint8
	LeftSent               uint8
	RightSent              uint8

	lastSent time.Time
}

// Connect establishes a connection with the physical bluno
// - Remember to close client when done
// - Remember to check disconnected before interacting with channel
// - To be run inside a goroutine
func (b *Bluno) Connect(pCtx context.Context, m chan bool, done chan bool) {
	// Dial to Bluno
	if commsintconfig.FineDebugMode {
		log.Printf("Entering HCI critical region: %s", b.Name)
	}

	<-m
	timedCtx, cancel := context.WithTimeout(pCtx, commsintconfig.ConnectionEstablishTimeout)
	defer cancel()
	client, err := ble.Dial(timedCtx, ble.NewAddr(b.Address))
	m <- true

	if commsintconfig.FineDebugMode {
		log.Printf("Exiting HCI critical region: %s", b.Name)
	}

	if err != nil {
		log.Printf("client_connection_fail|addr=%s|err=%s", b.Address, err)
		time.Sleep(2 * time.Second) // Sleep fixed duration to induce predictability and allow other connection attempts
		done <- false
		return
	}

	b.SetClient(&client)
	b.StateUpdateChan <- commsintconfig.NotHandshaked
	log.Printf("client_connection_succeeded|addr=%s", b.Address)

	done <- true
}

// Listen receives incoming connections from bluno
// - to be called inside a goroutine
func (b *Bluno) Listen(pCtx context.Context, wr func(commsintconfig.Packet), done chan bool) {
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
		done <- false
		b.Client.CancelConnection()
		return
	}

	// Isolate the characteristic
	c, err := b.Client.DiscoverCharacteristics(charUUID, s[0])
	if err != nil || len(c) != 1 {
		if commsintconfig.DebugMode {
			log.Printf("client_char_discovery_err|addr=%s|err=%s|num_characteristics=%d", b.Address, err, len(c))
		}
		done <- false
		b.Client.CancelConnection()
		return
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
		log.Printf("client_subscription_err|addr=%s|err=%s", b.Address, err)
		done <- false
		b.Client.CancelConnection()
		return
	}
	defer b.Client.Unsubscribe(characteristic, false)

	// Handshake
	log.Printf("Handshake initiated with %s (%s) service=%s char=%s", b.Name, b.Address, s[0].UUID.String(), characteristic.UUID.String())
	toSend := []byte{commsintconfig.InitHandshakeSymbol, byte('\r'), '\n'}
	err = b.Client.WriteCharacteristic(characteristic, toSend, false)
	if err != nil {
		log.Printf("write_handshake|err=%s", err)
		done <- false
		b.Client.CancelConnection()
		return
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
			b.PrintStats()
			done <- false
			return
		case <-hsFail:
			log.Printf("client_handshake_fail|addr=%s", b.Address)
			b.Client.CancelConnection()
		case t := <-tickChan.C:
			diff := t.Sub(b.LastPacketReceivedAt)
			if b.HandshakeAcknowledged && diff >= commsintconfig.ConnectionLivenessTimeout {
				b.PrintStats()
				log.Printf(
					"client_connection_terminated|liveness_ticker_exceed|packets received=%d|lastPacketReceived=%s|curr_t=%s",
					b.PacketsReceived,
					b.LastPacketReceivedAt,
					t,
				)
				b.Client.CancelConnection()
			}
		case et := <-establishTickChan.C:
			diff := et.Sub(b.LastPacketReceivedAt)
			if !b.HandshakeAcknowledged && diff >= commsintconfig.ConnectionEstablishTimeout {
				log.Printf(
					"client_connection_terminated|establish_ticker_exceed|packets received=%d|lastPacketReceived=%s|curr_t=%s",
					b.PacketsReceived,
					b.LastPacketReceivedAt,
					et,
				)
				b.Client.CancelConnection()
			}
		case <-pCtx.Done():
			log.Printf("client_connection_terminated|force=true|packets received=%d", b.PacketsReceived)
			b.PrintStats()
			b.StateUpdateChan <- commsintconfig.NotConnected
			b.Client.ClearSubscriptions()
			b.Client.CancelConnection()
			done <- true
			return
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

		if commsintconfig.DebugMode {
			printPacket(&p, resp)
		}

		switch p.Type {
		case commsintconfig.Ack:
			log.Printf("Handshake successful with %s (%s)", b.Name, b.Address)
			b.StateUpdateChan <- commsintconfig.Transmitting
			b.HandshakeAcknowledged = true
		case commsintconfig.Invalid:
			b.PacketsInvalidType++
		case commsintconfig.Liveness:
			if b.HandshakeAcknowledged == false {
				hsFail <- true
			} else {
				b.PacketsImmSuccess++
				b.resetLeftIndicator()
				b.resetRightIndicator()
			}
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

// determinePacketType returns the packet's type based on the 1st and 2nd bit of the 1st byte of the response
func determinePacketType(d []byte) commsintconfig.PacketType {
	if d[0] == commsintconfig.RespHandshakeSymbol {
		return commsintconfig.Ack
	} else if d[0] == commsintconfig.EMGDataSymbol {
		return commsintconfig.DataEMG
	} else if d[0] == commsintconfig.IMUDataSymbol {
		return commsintconfig.DataIMU
	} else if d[0] == commsintconfig.RespLivenessSymbol {
		return commsintconfig.Liveness
	}
	return commsintconfig.Invalid
}

// twoByteToNum converts 2 consecutive bytes into a uint16
// it assumes the bytes are arranged in little endian format
func twoByteToNum(d []byte, start uint8) int16 {
	return int16(binary.LittleEndian.Uint16(d[start : start+2]))
}

// fourByteToFloat converts 4 consecutive bytes into a float64
// it assumes the bytes are arranged in little endian format
func fourByteToFloat(d []byte, start uint8) float32 {
	asIntbits := binary.LittleEndian.Uint32(d[start : start+4])
	return math.Float32frombits(asIntbits)
}

// calculateChecksum takes a complete packet and finds its checksum
// returns true if checksum passes otherwise false
func calculateChecksum(d []byte) bool {
	givenChecksum := d[commsintconfig.ExpectedPacketSize-1]
	var c byte = 0x00

	for i := 0; i < commsintconfig.ExpectedPacketSize-1; i++ {
		c ^= d[i]
	}
	return givenChecksum == c
}

// decryptPacket takes an encrypted complete packet and performs
// aes decryption in the following manner:
// decrypt bytes 0 to 15 using stage 1 AES key
func decryptPacket(resp []byte) []byte {
	temp := make([]byte, commsintconfig.AESSize)
	cOne := commsintconfig.CreateBlockCipher()

	// StageOneDecryptor is used to decrypt the packet from stage 1 (first 16 bytes are encrypted) to stage 0 (plaintext)
	cOne.Decrypt(temp, resp)

	return append(temp, resp[commsintconfig.AESSize:]...)
}

// formTimestamp takes in a bluno, a packet and performs unix timestamp creation
func formTimestamp(b *Bluno, resp []byte, start uint8) time.Time {
	// only 3 bytes are dedicated for timestamps, last byte should be mimicked using all zeroes
	ba := append(resp[start:start+3], byte(0))
	ts := time.Millisecond * time.Duration(binary.LittleEndian.Uint32(ba))
	delta := time.Duration(int64(b.HandshakedAt.Sub(b.HandShakeInit)) / 2)
	return b.HandShakeInit.Add(delta).Add(ts)
}

// getMuscleSensorReading takes in a bluno, a packet and extracts the muscle sensor reading
// from the lower 8 bits and the 1st and 2nd bit from the 18th byte
func getMuscleSensorReading(b *Bluno, resp []byte, lower uint8, upper uint8) *uint16 {
	l := resp[lower]
	h := resp[upper] & commsintconfig.ADCmask
	p := binary.LittleEndian.Uint16([]byte{l, h})
	return &p
}

// getEMGSensorData takes in a packet and extracts the emg sensor data to provide a single fatigue level value
// https://www.ncbi.nlm.nih.gov/pmc/articles/PMC6679263/
// MAV = mean absolute value
// RMS = root mean square
// MNF = mean frequency
func getEMGSensorData(b *Bluno, resp []byte) (float32, float32, float32) {
	return fourByteToFloat(resp, 4), fourByteToFloat(resp, 8), fourByteToFloat(resp, 12)
}

func checkValWithinThreshold(val int16) bool {
	return (val < commsintconfig.IndicationThreshold) && (val > -commsintconfig.IndicationThreshold)

}

func (b *Bluno) updateBlunoMovementIndicator(p *commsintconfig.Packet) {
	if checkValWithinThreshold(p.Pitch) {
		b.resetLeftIndicator()
		b.resetRightIndicator()
		b.NotSentIndication++
		if b.NotSentIndication >= commsintconfig.IndicationNotSentActivationCount {
			b.lastSent = time.Unix(0, 0) // Reset back to unix
		}

	} else if p.Pitch < -commsintconfig.IndicationThreshold && checkValWithinThreshold(p.Roll) && checkValWithinThreshold(p.Yaw) { // Left
		b.resetRightIndicator()
		b.resetNotSentIndicator()
		b.LeftIndication++
		if (time.Now().Sub(b.lastSent) < commsintconfig.ReducedThresholdAllowance && b.LeftIndication >= commsintconfig.IndicationLeftReducedActivationCount) ||
			b.LeftIndication >= commsintconfig.IndicationLeftActivationCount {
			b.LeftSent++
			p.Movement = int8(commsintconfig.LeftShift)
			b.lastSent = time.Now()
		}
	} else if p.Pitch > commsintconfig.IndicationThreshold && checkValWithinThreshold(p.Roll) && checkValWithinThreshold(p.Yaw) { // Right
		b.resetLeftIndicator()
		b.resetNotSentIndicator()
		b.RightIndication++
		if (time.Now().Sub(b.lastSent) < commsintconfig.ReducedThresholdAllowance && b.RightIndication >= commsintconfig.IndicationRightReducedActivationCount) ||
			(b.RightIndication >= commsintconfig.IndicationRightActivationCount) {
			b.RightSent++
			p.Movement = int8(commsintconfig.RightShift)
			b.lastSent = time.Now()
		}
	}
}

func (b *Bluno) resetLeftIndicator() {
	b.LeftIndication = 0
}

func (b *Bluno) resetRightIndicator() {
	b.RightIndication = 0
}

func (b *Bluno) resetNotSentIndicator() {
	b.NotSentIndication = 0
}

func constructPacket(b *Bluno, resp []byte) commsintconfig.Packet {
	if !calculateChecksum(resp) {
		return commsintconfig.Packet{Type: commsintconfig.Invalid}
	}

	resp = decryptPacket(resp)

	t := determinePacketType(resp)

	pkt := commsintconfig.Packet{
		Timestamp:    formTimestamp(b, resp, 1).UnixNano() / int64(time.Millisecond),
		MuscleSensor: false,
		Type:         t,
		BlunoNumber:  b.Num,
		Movement:     0,
	}

	if t == commsintconfig.Ack {
		b.HandshakedAt = time.Now()
	} else if t == commsintconfig.DataIMU {
		pkt.X = twoByteToNum(resp, 4)
		pkt.Y = twoByteToNum(resp, 6)
		pkt.Z = twoByteToNum(resp, 8)
		pkt.Pitch = twoByteToNum(resp, 10)
		pkt.Roll = twoByteToNum(resp, 12)
		pkt.Yaw = twoByteToNum(resp, 14)
	} else if t == commsintconfig.DataEMG {
		pkt.MuscleSensor = true
		pkt.MAV, pkt.RMS, pkt.MNF = getEMGSensorData(b, resp)
	}

	b.updateBlunoMovementIndicator(&pkt)
	return pkt
}

// PrintStats prints out transmission statistics for a given bluno
// Each BLE 4.0 packet is between 31 (best) - 41 (worst case) bytes
// -> implies each packet is between 248 - 328 bits
// -> implies up to 351 - 464 packets can be received per second
func (b *Bluno) PrintStats() {
	fmt.Printf("--------------------------------\nPrinting statistics for %s\n--------------------------------\n", b.Name)
	elapsedTime := time.Now().Sub(b.StartTime).Seconds()

	fmt.Printf("Received: %d\nImmediately successful: %d\nReconciled: %d\nInvalid: %d\nIncorrect length: %d\n\n",
		b.PacketsReceived,
		b.PacketsImmSuccess,
		b.PacketsReconciled,
		b.PacketsInvalidType,
		b.PacketsIncorrectLength,
	)

	fmt.Println("Before fragmentated packet reconciliation:")
	fmt.Printf("Successful Packets: %d\nElapsed time since connection: %f\nEffective packets per second: %f\n",
		b.PacketsImmSuccess,
		elapsedTime,
		float64(b.PacketsImmSuccess)/elapsedTime,
	)
	fmt.Printf("Successful packet ratio: %f\nIncorrect length ratio: %f\nInvalid data ratio: %f\n\n",
		float64(b.PacketsImmSuccess)/float64(b.PacketsReceived),
		float64(b.PacketsIncorrectLength)/float64(b.PacketsReceived),
		float64(b.PacketsInvalidType)/float64(b.PacketsReceived),
	)

	fmt.Println("After fragmentated packet reconciliation:")
	// Reconciliation causes extra 1 InvalidType, 1 IncorrectLength and 1 PacketReceived
	adjIncorrectLength := float64(b.PacketsIncorrectLength - 2*b.PacketsReconciled)
	adjSuccessfulPackets := float64(b.PacketsImmSuccess + b.PacketsReconciled)

	fmt.Printf("Successful Packets: %d\nElapsed time since connection: %f\nEffective packets per second: %f\n",
		int(adjSuccessfulPackets),
		elapsedTime,
		adjSuccessfulPackets/elapsedTime,
	)

	fmt.Printf("Successful packet ratio: %f\nIncorrect length ratio: %f\nInvalid data ratio: %f\n\n",
		adjSuccessfulPackets/float64(b.PacketsReceived),
		float64(adjIncorrectLength)/float64(b.PacketsReceived),
		float64(b.PacketsInvalidType)/float64(b.PacketsReceived),
	)

	fmt.Printf("Counts %d %d\n",
		b.LeftSent,
		b.RightSent,
	)

	fmt.Printf("--------------------------------\nEnd of statistics for %s\n--------------------------------", b.Name)
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
	b.resetLeftIndicator()
	b.resetRightIndicator()
	b.resetNotSentIndicator()
	b.LeftSent = 0
	b.RightSent = 0
	b.lastSent = time.Now()
}
