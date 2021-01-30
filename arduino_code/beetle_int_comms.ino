// Packet Specification
#define PACKET_SIZE 20

// 65280 in decimal (FF00) onwards is reserved for protocol
uint16_t val = 65535; //FFFE
bool handshake_done = false;

// Handshake constants
char HANDSHAKE_INIT = 'A';
char HANDSHAKE_RESPONSE = 'B';
char DATA_RESPONSE = 'C';

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
  memset(buf, '0', PACKET_SIZE);
  *buf = HANDSHAKE_RESPONSE;
  char* done = addDataToBuffer(++buf, val, val-1, val-2, val-3, val-4, val-5);

  // Checksum
  memset(done, '1', 1);
}


// PrepareDataPacket prepares the data to be sent out
void PrepareDataPacket(char* buf) {
  memset(buf, '0', PACKET_SIZE);
  *buf = DATA_RESPONSE;
  val -= 1;
  char* done = addDataToBuffer(++buf, val, val-1, val-2, val-3, val-4, val-5);

  // Checksum
  memset(done, '1', 1);
}



void setup() {
  handshake_done = false;
  Serial.begin(115200);
  sendBuffer[PACKET_SIZE] = '\0';
}


 
void loop(){
  // Handshake from laptop
  if (Serial.available() > 0) {
    char incomingByte = Serial.read();
    if (incomingByte == HANDSHAKE_INIT) {
      PrepareHandshakeAck(sendBuffer);
      Serial.write(sendBuffer, PACKET_SIZE);
      handshake_done = true;
    } else {
      // Clear input/output buffer
      Serial.flush();
      while (Serial.available() > 0) {
        Serial.read();
      }
    }
  }
  else if (handshake_done) {
    PrepareDataPacket(sendBuffer);
    Serial.write(sendBuffer, PACKET_SIZE);
    delay(7); // Seems to give 140 correct packets/sec (20 bytes of usable data each), we use this as baseline. Theoretical limit is around 350 packets/sec at 115200 bps
  }
}  