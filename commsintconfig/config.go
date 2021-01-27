package commsintconfig

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
