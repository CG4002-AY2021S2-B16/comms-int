// Packet Specification
#define PACKET_SIZE 18


uint16_t val = 541; //0000 0010 0001 1101 -> 02 1D

// Handshake constants
byte HANDSHAKE_INIT = 'A';
byte HANDSHAKE_RESPONSE = 'B';

// Buffer used to write to bluetooth
char sendBuffer[PACKET_SIZE]; 


// addIntToBuffer writes an integer as 2 bytes to the buffer
// It uses little endian e.g. 0x0A0B -> 0B 0A
// returns next location after filling in 2 bytes
char* addIntToBuffer(char * start, uint16_t x) {
  *start = x;
  start++;
  *start = x >> 8;
  start++;
  return start;
}


// Expect fatigue level, etc. in the future
void addDataToBuffer(char* next, uint16_t x, uint16_t y, uint16_t z, uint16_t yaw, uint16_t roll, uint16_t pitch) {
  next = addIntToBuffer(next, x);
  next = addIntToBuffer(next, y);
  next = addIntToBuffer(next, z);
  next = addIntToBuffer(next, yaw);
  next = addIntToBuffer(next, roll);
  next = addIntToBuffer(next, pitch);
}


// PrepareHandshakeAck prepares the buffer to respond to an incoming handshake request
void PrepareHandshakeAck(char* buf) {
  memset(buf, 0, PACKET_SIZE);
  buf[0] = HANDSHAKE_RESPONSE;
  addDataToBuffer(++buf, val, val+1, val+2, val+3, val+4, val+5);
}


void setup() {
    Serial.begin(115200);
    //pinMode(A0, INPUT);
}


 
void loop()
{
  if (Serial.available()) { //send what has been received
    // Handshake from laptop
    if (Serial.read() == HANDSHAKE_INIT) {
      PrepareHandshakeAck(sendBuffer);
      Serial.write(sendBuffer);
    }
    Serial.write(Serial.read());
  }
}  