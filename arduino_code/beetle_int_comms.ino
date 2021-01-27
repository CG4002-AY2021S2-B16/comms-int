int val = 1234;

void setup() {
    Serial.begin(115200);               //initial the Serial
    pinMode(A0, INPUT);
}
 
void loop()
{
//  if (Serial.available()) { //send what has been received
//    Serial.write(Serial.read());
//  }
  
  delay(100);
  //Serial.write()
  
//  //val = analogRead(A0);
  char ptr1[16];

  Serial.write(itoa(val, ptr1, 10));
  val += 1;
}  