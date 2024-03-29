package commsintconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"time"
)

// DebugMode enables debug log messages
var DebugMode bool = true

// ProcessedPacketDisplay allows output packets to be displayed in a colored manner
var ProcessedPacketDisplay bool = true

// FineDebugMode enables debug mode at a stricter level
var FineDebugMode bool = false

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

// RespLivenessSymbol is the symbol received from a successful liveness packet.
// We can AND the 17th byte received to see if it returns the same value.
// If so, the packet is indeed a Liveness packet.
var RespLivenessSymbol byte = 0x04

// RespDataSymbol is the symbol received from a successful data response.
// We can AND the 17th byte received with this to see if it returns the same value.
// If so, the packet is indeed a Data packet.
// NOTE: A further check for 0x04 should be done to determine if muscle sensor data is available or not.
var RespDataSymbol byte = 0x08

// NonMuscleSensorSymbol is the symbol associated with an incoming data packet that does not have muscle sensor attached.
// We can AND the 17th byte received with this to see if it returns the same value.
// If so, the packet is a Data packet that does not contain a valid muscle sensor value.
var NonMuscleSensorSymbol byte = 0x04

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
	// DataEMG is a PacketType that refers to a response containing EMG data
	DataEMG PacketType = 2
	// Liveness is a PacketType that refers to a liveness check packet
	Liveness PacketType = 3
	// Invalid is a PacketType that we are not sure about
	Invalid PacketType = 4
)

// Packet is constructed from a complete bluetooth response
type Packet struct {
	Timestamp    int64      `json:"unix_timestamp_milliseconds"`
	X            int16      `json:"x"`
	Y            int16      `json:"y"`
	Z            int16      `json:"z"`
	Pitch        int16      `json:"pitch"`
	Roll         int16      `json:"roll"`
	Yaw          int16      `json:"yaw"`
	Type         PacketType `json:"-"`
	BlunoNumber  uint8      `json:"bluno"`
	Movement     int8       `json:"movement"`
	MuscleSensor bool       `json:"muscle_sensor"` // If this key is present, ignore the 6 IMU values - they are dummy readings.
	MAV          float32    `json:"mean_absolute_value,omitempty"`
	RMS          float32    `json:"root_mean_square,omitempty"`
	MNF          float32    `json:"mean_frequency,omitempty"`
}

func (p Packet) String() string {
	s := fmt.Sprintf("Timestamp: %d X:%d Y:%d Z:%d Pitch:%d Roll:%d Yaw:%d Type:%d BlunoNumber:%d Movement:%d",
		p.Timestamp,
		p.X,
		p.Y,
		p.Z,
		p.Pitch,
		p.Roll,
		p.Yaw,
		p.Type,
		p.BlunoNumber,
		p.Movement,
	)

	if p.MuscleSensor {
		return s + fmt.Sprintf(
			" MuscleSensor active MAV:%f RMS:%f MNF:%f ",
			p.MAV,
			p.RMS,
			p.MNF,
		)
	}
	return s
}

// Connection timeout parameters

// ConnectionEstablishTimeout is the timeout for establishing connection, and then refreshed for handshake
var ConnectionEstablishTimeout time.Duration = 1500 * time.Millisecond

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
// low number -> stream-like data
// high number -> windowed data
var OutputSize int = 4

// OutputDequeueInterval wakes up the dequeue goroutine to send data over via ext comms interface
var OutputDequeueInterval time.Duration = 5 * time.Millisecond

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

// AESSize refers to the standard size of the buffers used
var AESSize int = 16

// StageTwoOffset refers to the offset at which the second layer of encryption is performed
var StageTwoOffset int = 2

// CreateBlockCiphers creates 2 decryption ciphers to decrypt a packet
func CreateBlockCiphers() (cipher.Block, cipher.Block) {
	cOne, _ := aes.NewCipher([]byte{0x2A, 0x46, 0x2D, 0x4A, 0x61, 0x4E, 0x64, 0x52, 0x67, 0x55, 0x6A, 0x58, 0x6E, 0x32, 0x72, 0x35})
	cTwo, _ := aes.NewCipher([]byte{0x7A, 0x24, 0x43, 0x26, 0x46, 0x29, 0x4A, 0x40, 0x4E, 0x63, 0x52, 0x66, 0x55, 0x6A, 0x57, 0x6E})
	return cOne, cTwo
}

type LateralShift int8

const (
	LeftShift LateralShift = -1

	RightShift LateralShift = 1

	NoShiftIndicated LateralShift = 0
)

const IndicationThreshold = 1200

const (
	IndicationLeftActivationCount         = 5
	IndicationRightActivationCount        = 8
	IndicationNotSentActivationCount      = 8
	IndicationLeftReducedActivationCount  = 4
	IndicationRightReducedActivationCount = 4
	ReducedThresholdAllowance             = 6 * time.Second
)
