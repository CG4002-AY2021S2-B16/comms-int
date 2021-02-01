void setup() {
  Serial.begin(115200);
}


void loop(){
  receiveData();
  if (new_handshake_req) {
    handshakeResponse();
    handshake_done = true;
  } 
  else if (handshake_done) {
    dataResponse();
  }
  delay(500); // Seems to give 140 correct packets/sec (20 bytes of usable data each), we use this as baseline. Theoretical limit is around 350 packets/sec at 115200 bps
}

//void setup() {
//    Serial.begin(115200);               //initial the Serial
//}
// 
//void loop()
//{
//    if(Serial.available())
//    {
//      Serial.print(Serial.read());
//    }
//    delay(100);
//}  
