package commsintconfig

import "time"

// DebugMode enables debug log messages
var DebugMode bool = false

// BlunoServiceUUID is the single (predecided) Service used for Serial communications from the bluno beetle
var BlunoServiceUUID string = "0000dfb0-0000-1000-8000-00805f9b34fb"

// BlunoServiceReducedUUID is the same as the above, except it is trimmed down to 2 bytes for library compatibility
var BlunoServiceReducedUUID uint16 = 0xdfb0

// BlunoCharacteristicUUID is the single (predecided) Characteristic used for Serial communications from the bluno beetle
var BlunoCharacteristicUUID string = "0000dfb1-0000-1000-8000-00805f9b34fb"

// BlunoCharacteristicReducedUUID is the same as the above, except it is trimmed down to 2 bytes for library compatibility
var BlunoCharacteristicReducedUUID uint16 = 0xdfb1

// ClientCharacteristicConfigHandle - https://www.dfrobot.com/forum/viewtopic.php?t=148
var ClientCharacteristicConfigHandle uint16 = 0x0025

// CommandCharacteristicUUID is the single (predecided) Characteristic used for AT Commands
var CommandCharacteristicUUID string = "0000dfb2-0000-1000-8000-00805f9b34fb"

// CommandCharacteristicReducedUUID is the same as the above, except it is trimmed down to 2 bytes for library compatibility
var CommandCharacteristicReducedUUID uint16 = 0xdfb2

// CommandCharacteristicConfigHandle is used for reset of BLE chip - https://www.dfrobot.com/forum/viewtopic.php?t=26173
var CommandCharacteristicConfigHandle uint16 = 0x0028

// ClientCharacteristicConfig is the descriptor required for subscription
// References (some good stuff right here):
// https://github.com/pauldemarco/flutter_blue/issues/185
// https://www.dfrobot.com/forum/viewtopic.php?t=2035
var ClientCharacteristicConfig uint16 = 0x2902

// InitHandshakeSymbol is the symbol used for handshake initialization
var InitHandshakeSymbol byte = 'A'

// RespHandshakeSymbol is the symbol received from a successful handshake attempt
// We can OR the 17th byte received with this to see if it returns the same value.
// If so, the packet is indeed an ACK packet.
var RespHandshakeSymbol byte = 0xF3

// RespDataSymbol is the symbol received from a successful data response.
// We can AND the 17th byte received with this to see if it returns the same value.
// If so, the packet is indeed a Data packet.
var RespDataSymbol byte = 0x0C

// ADCmask is the mask used to extract upper 2 bits for the 10-bit muscle sensor ADC reading from an incoming packet
var ADCmask byte = 0x03

// ExpectedPacketSize refers to the number of useful bytes of data within an incoming packet
var ExpectedPacketSize int = 19

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
	Timestamp    int64  `json:"unix_timestamp_milliseconds"`
	X            uint16 `json:"x"`
	Y            uint16 `json:"y"`
	Z            uint16 `json:"z"`
	Pitch        uint16 `json:"pitch"`
	Roll         uint16 `json:"roll"`
	Yaw          uint16 `json:"yaw"`
	MuscleSensor uint16 `json:"muscle_sensor"`
	Type         PacketType
	BlunoNumber  uint8 `json:"bluno"`
}

// Connection timeout parameters

// ConnectionEstablishTimeout is the timeout for establishing connection, and then another 1 sec for handshake
var ConnectionEstablishTimeout time.Duration = 1 * time.Second

// ConnectionLivenessCheckInterval is the intervals in which it is checked whether a reconnection should be triggered
var ConnectionLivenessCheckInterval time.Duration = 40 * time.Millisecond

// ConnectionLivenessTimeout is the max leeway, after which reconnection is attempted
var ConnectionLivenessTimeout time.Duration = 2000 * time.Millisecond

// State indicates current program status
type State int

const (
	// Waiting refers to an idling application that is waiting on a message queue start signal
	Waiting State = 1
	// Running refers to a running application that is interacting with blunos and writing to output
	Running State = 2
)

// BLEResetString refer to the string version of "AT+RESTART<CR+LF>"
var BLEResetString string = "AT+VERSION=?\r\n" //"AT+RESTART\r\n"

// OutputSize refers to the number of packets accumulated before output is sent over to the ext comms interface
var OutputSize int = 20

// OutputDequeueInterval wakes up the dequeue goroutine to send data over via ext comms interface
var OutputDequeueInterval time.Duration = 100 * time.Millisecond

// BlunoStatus indicates the current status of blunos being managed by the int comm server
type BlunoStatus uint8

const (
	// NotConnected refers to a bluno that is not connected
	NotConnected BlunoStatus = 0

	// NotHandshaked refers to a bluno that is connected but has not had a successful handshake
	NotHandshaked BlunoStatus = 1

	// Transmitting refers to a bluno that is connected and transmitting data
	Transmitting BlunoStatus = 2
)
