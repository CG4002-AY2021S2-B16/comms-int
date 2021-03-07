// Dynamically changing dummy data value
int16_t dummy_val = 32767;
int16_t neg_dummy_val = -32768;


void setup() {
  // // Initialize the i2c wire connection
  // Wire.begin();

  prepareAES();
  Serial.begin(115200);
  delay(1000);

  
  //  imuSensor.initialize();
  //  if (!imuSensor.testConnection())
  //  {
  //    Serial.println("MPU6050 connection failed!");
  //  }
}


void loop(){
  receiveData();
  if (new_handshake_req) {
    handshakeResponse();
    resetTimeOffset();

    // If for some reason, Beetle is out of sync with laptop
    // Skip a single handshake, because issue resolves with time
    if (handshake_done) {
      Serial.flush();
      handshake_done = false;
    } else {
      handshake_done = true;
    }
    delay(300);
  } 
  else if (handshake_done) {
    if (checkLivenessPacketRequired()) {
      livenessResponse();
    } else {
      // if data available
      dataResponse(0, -0, dummy_val, neg_dummy_val, dummy_val + neg_dummy_val, -500);
    }
    dummy_val--;
    neg_dummy_val++;
  }
  delay(7);
}
