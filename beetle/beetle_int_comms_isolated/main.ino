int16_t accelX, accelY, accelZ;
int16_t gyroX, gyroY, gyroZ;


void setup() {
  // // Initialize the i2c wire connection
  // Wire.begin();
  
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
    handshake_done = true;
    delay(500);
  } 
  else if (handshake_done) {
    // imuSensor.getMotion6(&accelX, &accelY, &accelZ, &gyroX, &gyroY, &gyroZ);
    dataResponse(accelX, accelY, accelZ, gyroX, gyroY, gyroZ);
  }
  delay(20);
}
