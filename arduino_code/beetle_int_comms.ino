// Packet Specification
#define PACKET_SIZE 18

// 65280 in decimal (FF00) onwards is reserved for protocol
uint16_t val = 64000; //FFFE

// Handshake constants
byte HANDSHAKE_INIT = 'A';
byte HANDSHAKE_RESPONSE = 'B';

// Buffer used to write to bluetooth
char sendBuffer[PACKET_SIZE + 1]; 


// addIntToBuffer writes an integer as 2 bytes to the buffer
// It uses big endian e.g. 0x0A0B -> 0A 0B
// returns next location after filling in 2 bytes
char* addIntToBuffer(char * start, uint16_t x) {
  *start = (x >> 8) & 0xFF;
  start++;
  *start = x & 0xFF;
  start++;
  return start;
}


// addAsChar writes a byte to a char



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


// PrepareHandshakeAck prepares the buffer to respond to an incoming handshake request
void PrepareHandshakeAck(char* buf) {
  memset(buf, 0, PACKET_SIZE);
  buf[0] = HANDSHAKE_RESPONSE;
  char* done = addDataToBuffer(++buf, val, val-1, val-2, val-3, val-4, val-5);

  // Checksum
  memset(done, 1, 1);
}


void setup() {
  sendBuffer[PACKET_SIZE] = '\0';
  Serial.begin(115200);
}


 
void loop()
{
  if (Serial.available()) { //send what has been received
    // Handshake from laptop
    if (Serial.read() == HANDSHAKE_INIT) {
      PrepareHandshakeAck(sendBuffer);
      Serial.write(sendBuffer, PACKET_SIZE);
    } else {
      Serial.write(Serial.read()); 
    }
  }
}  