void setup() {
    Serial.begin(115200);               //initial the Serial
}
 
void loop()
{
//  if (Serial.available()) { //send what has been received
//    Serial.write(Serial.read());
//  }
  delay(500);
  Serial.write("ABCD");    
}  
