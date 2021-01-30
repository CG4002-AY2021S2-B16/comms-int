package commsintconfig

import "time"

// DebugMode enables debug log messages
var DebugMode bool = true

// BlunoServiceUUID is the single (predecided) Service used for Serial communications from the bluno beetle
var BlunoServiceUUID string = "0000dfb0-0000-1000-8000-00805f9b34fb"

// BlunoServiceReducedUUID is the same as the above, except it is trimmed down to 2 bytes for library compatibility
var BlunoServiceReducedUUID uint16 = 0xdfb0

// BlunoCharacteristicUUID is the single (predecided) Characteristic used for Serial communications from the bluno beetle
var BlunoCharacteristicUUID string = "0000dfb1-0000-1000-8000-00805f9b34fb"

// BlunoCharacteristicReducedUUID is the same as the above, except it is trimmed down to 2 bytes for library compatibility
var BlunoCharacteristicReducedUUID uint16 = 0xdfb1

// ClientCharacteristicConfig is the descriptor required for subscription
// References (some good stuff right here):
// https://github.com/pauldemarco/flutter_blue/issues/185
// https://www.dfrobot.com/forum/viewtopic.php?t=2035
var ClientCharacteristicConfig uint16 = 0x2902

// ClientCharacteristicConfigHandle - https://www.dfrobot.com/forum/viewtopic.php?t=148
var ClientCharacteristicConfigHandle uint16 = 0x0025

// InitHandshakeSymbol is the symbol used for handshake initialization
var InitHandshakeSymbol byte = 'A'

// RespHandshakeSymbol is the symbol received from a successful handshake attempt
var RespHandshakeSymbol byte = 'B'

// RespDataSymbol is the symbol received from a successful data response
var RespDataSymbol byte = 'C'

// ExpectedPacketSize refers to the number of useful bytes of data within an incoming packet
var ExpectedPacketSize int = 20

// PacketType is an enum type which signifies the type of packet received from the Bluno
type PacketType uint8

const (
	// Ack is a PacketType that refers to a handshake response
	Ack PacketType = 0
	// Data is a PacketType that refers to a response containing data
	Data PacketType = 1
	// Invalid is a PacketType that we are not sure about
	Invalid PacketType = 2
)

// Packet is constructed from a complete bluetooth response
type Packet struct {
	Type  PacketType
	X     uint16
	Y     uint16
	Z     uint16
	Yaw   uint16
	Pitch uint16
	Roll  uint16
}

// Connection timeout parameters

// ConnectionEstablishTimeout is the timeout for establishing connection
var ConnectionEstablishTimeout time.Duration = 1 * time.Second

// ConnectionLivenessTimeout is the timeout after which reconnection is attempted
var ConnectionLivenessTimeout time.Duration = 2000 * time.Millisecond
