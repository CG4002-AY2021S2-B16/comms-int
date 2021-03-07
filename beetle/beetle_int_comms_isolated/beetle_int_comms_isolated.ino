#include <Crypto.h>
#include <AES.h>

// Packet Specification
#define PACKET_SIZE 19
#define MUSCLE_SENSOR_INVALID_VAL 1023

// Packet masks
#define DESIGNATE_DATA_PACKET_MASK 0x0C // 0b0000 1100, To be ORed with
#define DESIGNATE_DATA_PACKET_WITH_MUSCLE_SENSOR_MASK 0x08 // 0b0000 1000, To be ORed with

#define DESIGNATE_ACK_PACKET_MASK 0xF3  // 0b1111 0011, To be ANDed with
#define DESIGNATE_LIVENESS_PACKET_MASK 0x04 // To be ORed with

// Handshake constants
#define HANDSHAKE_INIT 'A'

// Liveness constants
#define LIVENESS_TIMEOUT 800 // Just below half of receiver-side timeout, so that 2 attempts can be made before reconnection happens

// Encryption
#define AES_BLOCK_SIZE 16

AESTiny128 aesTinyOne;
BlockCipher *cipherOne = &aesTinyOne;

AESTiny128 aesTinyTwo;
BlockCipher *cipherTwo = &aesTinyTwo;

const byte AESKeyStageOne[AES_BLOCK_SIZE] = {0x2A, 0x46, 0x2D, 0x4A, 0x61, 0x4E, 0x64, 0x52, 0x67, 0x55, 0x6A, 0x58, 0x6E, 0x32, 0x72, 0x35};
const byte AESKeyStageTwo[AES_BLOCK_SIZE] = {0x7A, 0x24, 0x43, 0x26, 0x46, 0x29, 0x4A, 0x40, 0x4E, 0x63, 0x52, 0x66, 0x55, 0x6A, 0x57, 0x6E};


// Define global variables
uint8_t receivedChar;
boolean new_handshake_req = false;
boolean handshake_done = false;
boolean muscle_sensor_active = false;

// Time
uint32_t start_time = millis();
uint32_t last_packet_sent_at = millis();

// Buffer used to write to bluetooth
uint8_t sendBuffer[PACKET_SIZE];


/* ---------------------------------
 * TIME FUNCTIONS
 * ---------------------------------
 */
 
void resetTimeOffset() {
  start_time = millis();
}

uint32_t calculateTimestamp() {
  return millis() - start_time;
}

void updateLastPacketSent() {
  last_packet_sent_at = millis();
}

bool checkLivenessPacketRequired() {
  return (millis() - last_packet_sent_at) >= LIVENESS_TIMEOUT;
}


/* ---------------------------------
 *  ENCRYPTION FUNCTIONS
 * ---------------------------------
 */

// prepareAES sets the keys for each cipher
void prepareAES() {
  crypto_feed_watchdog();
  cipherOne->setKey(AESKeyStageOne, AES_BLOCK_SIZE);
  cipherTwo->setKey(AESKeyStageTwo, AES_BLOCK_SIZE);
}

// encryptAES encrypts sixteen bytes using AES-ECB
// It encrypts the sixteen bytes found from the start pointer
void encryptAES(uint8_t* start) {
  // Perform first encryption
  crypto_feed_watchdog();
  cipherOne->encryptBlock(start, start);

  // Perform second encryption (currently, bytes 2 to 17 inclusive)
  start += 2;
  crypto_feed_watchdog();
  cipherTwo->encryptBlock(start, start);
}


/* ---------------------------------
 *  BUFFER FUNCTIONS
 * ---------------------------------
 */

// clearSendBuffer clears the buffer to be used for populating a packet
void clearSendBuffer() {
  memset(sendBuffer, 0, PACKET_SIZE);
}


// setChecksum calculates and writes the checksum to the outgoing packet
// based on the existing bytes within the sendBuffer
void setChecksum() {
  uint8_t checksum_val = 0;
  
  for (int i = 0; i < (PACKET_SIZE - 1); i++) {
    checksum_val ^= sendBuffer[i];
  }

  sendBuffer[PACKET_SIZE - 1] = checksum_val;
}


// addIntToBuffer writes an integer as 2 bytes to the buffer
// It uses little endian e.g. 0x1A0B -> 0B 1A
// returns next location after filling in 2 bytes
uint8_t* addIntToBuffer(uint8_t* start, uint16_t x) {
  for (int i = 0; i < 2; i++) {
    start[i] = (x >> (i * 8)) & 0xFF;
  }
  return start + 2;
}


// addLongToBuffer writes a 32-bit integer as 4 bytes to the buffer
// It uses little endian e.g. 0x0A1B2C3D -> 3D 2C 1B 0A
// returns next location after filling in 4 bytes
uint8_t* addLongToBuffer(uint8_t* start, uint32_t x) {
  for (int i = 0; i < 4; i++) {
    start[i] = (x >> (i * 8)) & 0xFF; 
  }
  return start + 4;
}


// Adds 16-bit IMU data to the buffer 
uint8_t* addIMUDataToBuffer(uint8_t* next, uint16_t x, uint16_t y, uint16_t z, uint16_t pitch, uint16_t roll, uint16_t yaw) {
  next = addIntToBuffer(next, x);
  next = addIntToBuffer(next, y);
  next = addIntToBuffer(next, z);
  next = addIntToBuffer(next, pitch);
  next = addIntToBuffer(next, roll);
  next = addIntToBuffer(next, yaw);
  return next;
}


//addMuscleSensorDataToBuffer adds 10-bit Muscle Sensor data to the buffer
uint8_t* addMuscleSensorDataToBuffer(uint8_t* next, uint16_t ms_val) {  
  next[0] = ms_val & 0xFF;
  next[1] = (ms_val >> 8) & 0xFF;
  return next + 1; // returns the partially filled byte in this case
}


// setDataPacketTypeToBuffer adds 2-bit packet type data to the buffer to designate as data packet
uint8_t* setDataPacketTypeToBuffer(uint8_t* next) {
  if (muscle_sensor_active) {
    next[0] |= DESIGNATE_DATA_PACKET_WITH_MUSCLE_SENSOR_MASK;
  } else {
    next[0] |= DESIGNATE_DATA_PACKET_MASK;
  }
  return next + 1;
}


// setAckPacketTypeToBuffer adds 2-bit packet type data to the buffer to designate as data packet
uint8_t* setAckPacketTypeToBuffer(uint8_t* next) {
  next[0] &= DESIGNATE_ACK_PACKET_MASK;
  return next + 1;
}


// setLivenessPacketTypeToBuffer adds 2-bit packet type data to the buffer to designate as liveness packet
uint8_t* setLivenessPacketTypeToBuffer(uint8_t* next) {
  next[0] |= DESIGNATE_LIVENESS_PACKET_MASK;
  return next + 1;
}



/* ---------------------------------
 *  COMMS FUNCTIONS
 * ---------------------------------
 */

// handshakeResponse prepares the buffer to respond to an incoming handshake request
void handshakeResponse() {
  // Pre-process
  clearSendBuffer();
  uint8_t* buf = sendBuffer;
  
  // Fill different sections of the buffer
  buf = addLongToBuffer(buf, calculateTimestamp());
  buf = addIMUDataToBuffer(buf, 0, -0, 32767, -32768, 100, -100); // Arbitrary values to verify signed transmission integrity

  uint8_t* partial = addMuscleSensorDataToBuffer(buf, MUSCLE_SENSOR_INVALID_VAL);
  buf = setAckPacketTypeToBuffer(partial);

  // Perform encryption
  encryptAES(sendBuffer);

  // Calculate and fill in checksum
  setChecksum();
  
  // Send response out
  Serial.write(sendBuffer, PACKET_SIZE);
  updateLastPacketSent();
}


// livenessResponse prepares the buffer to send out a liveness packet, so that the receiver can be aware of the Bluno's connection
void livenessResponse() {
  // Pre-process
  clearSendBuffer();
  uint8_t* buf = sendBuffer;

  // Fill different sections of the buffer
  buf = addLongToBuffer(buf, calculateTimestamp());
  buf = addIMUDataToBuffer(buf, 0, -0, 32767, -32768, 100, -100); // Arbitrary values to verify signed transmission integrity

  uint8_t* partial = addMuscleSensorDataToBuffer(buf, MUSCLE_SENSOR_INVALID_VAL);
  buf = setLivenessPacketTypeToBuffer(partial);

  // Perform encryption
  encryptAES(sendBuffer);

  // Calculate and fill in checksum
  setChecksum();
  
  // Send response out
  Serial.write(sendBuffer, PACKET_SIZE);
  updateLastPacketSent();
}


// dataResponse prepares the data to be sent out
// Rotation about X axis = pitch
// Rotation about Y axis = roll
// Rotation about Z axis = yaw
void dataResponse(int16_t x, int16_t y, int16_t z, int16_t pitch, int16_t roll, int16_t yaw) {
  // Pre-process
  clearSendBuffer();
  uint8_t* buf = sendBuffer;
  
  // Fill different sections of the buffer
  buf = addLongToBuffer(buf, calculateTimestamp());
  buf = addIMUDataToBuffer(buf, x, y, z, pitch, roll, yaw);

  uint8_t* partial = (muscle_sensor_active)? addMuscleSensorDataToBuffer(buf, MUSCLE_SENSOR_INVALID_VAL) : addMuscleSensorDataToBuffer(buf, MUSCLE_SENSOR_INVALID_VAL);
  buf = setDataPacketTypeToBuffer(partial);

  // Perform encryption
  encryptAES(sendBuffer);

  // Calculate and fill in checksum
  setChecksum();

  // Send response out
  Serial.write(sendBuffer, PACKET_SIZE);
  updateLastPacketSent();
}



void receiveData() {
  new_handshake_req = false;
  
  while (Serial.available() > 0) {
      receivedChar = Serial.read();
      
      if (receivedChar == HANDSHAKE_INIT) {
        new_handshake_req = true;
        break;
      }
  }
}
