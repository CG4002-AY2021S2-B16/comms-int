#define EMG_SENSOR_MODE false

// Dynamically changing dummy data value
int16_t dummy_val = 32767;
int16_t neg_dummy_val = -32768;

float dummy_f_val = 314.26;
float neg_dummy_f_val = -527.984231;


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
      delay(7);
    } else if (EMG_SENSOR_MODE) {
      EMGdataResponse(0.00, dummy_f_val, neg_dummy_f_val);
      delay(128); // There should not be a delay on integrated code, because delay comes exclusively from EMG sampling
    } else {
      IMUdataResponse(0, -0, dummy_val, neg_dummy_val, dummy_val + neg_dummy_val, -500);
      delay(7);
    }
    dummy_val--;
    neg_dummy_val++;
    dummy_f_val--;
    neg_dummy_f_val++;
  }
}
