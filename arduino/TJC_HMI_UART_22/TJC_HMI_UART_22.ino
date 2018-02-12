#include <SoftwareSerial.h>
#include <Nextion.h>

SoftwareSerial nextion(2, 3); // Plug Nextion TX to pin 2 and RX to pin 3 of Arduino

Nextion myNextion(nextion, 9600); // Create a Nextion object named myNextion using the nextion serial port @ 9600bps

String inputString = "";

void setup() {
  Serial.begin(9600);
  myNextion.init();
  inputString.reserve(200);
}

void loop() {
  while (Serial.available()) {
    char inChar = (char)Serial.read();
    if (inChar == '$'){
      outputToLCD(inputString);
      inputString = "";
    } else {
        inputString += inChar;
    }
  }
}

void outputToLCD(String input) {
  myNextion.setComponentText("cpu0", String(input) + "%");
  myNextion.setComponentText("cpu1", String(input) + "*C");
  myNextion.setComponentText("ram0", String(input) + "%");
  myNextion.setComponentText("ram1", String(input) + "MB");
  myNextion.setComponentText("gpu0", String(input) + "%");
  myNextion.setComponentText("gpu1", String(input) + "MB");
}
