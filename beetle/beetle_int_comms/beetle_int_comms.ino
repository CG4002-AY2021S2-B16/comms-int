// Packet Specification
#define PACKET_SIZE 20

// Define global variables
char receivedChar;
boolean new_handshake_req = false;
boolean handshake_done = false;

// Handshake constants
char HANDSHAKE_INIT = 'A';
char HANDSHAKE_RESPONSE = 'B';
char DATA_RESPONSE = 'C';

// Buffer used to write to bluetooth
char sendBuffer[PACKET_SIZE + 1];

// Dummy data value
uint16_t val = 65535; //FFFE


/* ---------------------------------
 * COMMS UTILITY FUNCTIONS ARE BELOW
 * ---------------------------------
 */


// clearSendBuffer clears the buffer to be used for populating a packet
void clearSendBuffer() {
  memset(sendBuffer, '0', PACKET_SIZE + 1);
  sendBuffer[PACKET_SIZE] = '\0';
}


// fillChecksum calculates and writes the checksum to the outgoing packet
void fillChecksum() {
  sendBuffer[19] = '1';
}
 

// addIntToBuffer writes an integer as 2 bytes to the buffer
// It uses big endian e.g. 0x0A0B -> 0A 0B
// returns next location after filling in 2 bytes
char* addIntToBuffer(char* start, uint16_t x) {
  *start = (x >> 8) & 0xFF;
  start++;
  *start = x & 0xFF;
  start++;
  return start;
}


// Expect fatigue level, etc. in the future
char* addDataToBuffer(char* next, uint16_t x, uint16_t y, uint16_t z, uint16_t yaw, uint16_t pitch, uint16_t roll) {
  next = addIntToBuffer(next, x);
  next = addIntToBuffer(next, y);
  next = addIntToBuffer(next, z);
  next = addIntToBuffer(next, yaw);
  next = addIntToBuffer(next, pitch);
  next = addIntToBuffer(next, roll);
  return next;
}


// handshakeResponse prepares the buffer to respond to an incoming handshake request
void handshakeResponse() {
  // Pre-process
  clearSendBuffer();
  char* buf = sendBuffer;
  
  // Process
  *buf = HANDSHAKE_RESPONSE;
  char* done = addDataToBuffer(++buf, val, val-1, val-2, val-3, val-4, val-5);

  // Checksum
  fillChecksum();
  
  // Send response out
  Serial.write(sendBuffer, PACKET_SIZE);
}


// dataResponse prepares the data to be sent out
void dataResponse() {
  // Pre-process
  val -= 1;
  clearSendBuffer();
  char* buf = sendBuffer;
  
  // Process
  *buf = DATA_RESPONSE;
  char* done = addDataToBuffer(++buf, val, val-1, val-2, val-3, val-4, val-5);

  // Checksum
  fillChecksum();

  // Send response out
   Serial.write(sendBuffer, PACKET_SIZE);
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
