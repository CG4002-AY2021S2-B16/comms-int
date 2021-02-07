void setup() {
  Serial.begin(115200);
  delay(200);
}


void loop(){
  receiveData();
  if (new_handshake_req) {
    handshakeResponse();
    resetTimeOffset();
    handshake_done = true;
    delay(200);
  } 
  else if (handshake_done) {
    dataResponse();
  }
  delay(20); // Seems to give 140 correct packets/sec (20 bytes of usable data each), we use this as baseline. Theoretical limit is around 350 packets/sec at 115200 bps
}
