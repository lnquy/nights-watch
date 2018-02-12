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
  String cmd = getValue(input, '|', 0);
  if (cmd == "0") {
    // TODO: Config
  } else if (cmd == "1") {
    // CPU
    String load = getValue(input, '|', 1);
    String temp = getValue(input, '|', 2);
    myNextion.setComponentText("cpu0", String(load) + "%");
    myNextion.setComponentText("cpu1", String(temp) + "*C");
  } else if (cmd == "2") {
    // MEM
    String load = getValue(input, '|', 1);
    String usage = getValue(input, '|', 2);
    myNextion.setComponentText("ram0", String(load) + "%");
    myNextion.setComponentText("ram1", String(usage) + "MB");
  } else if (cmd == "3") {
    // GPU
  } else if (cmd == "4") {
    // NET
    String up = getValue(input, '|', 1);
    String down = getValue(input, '|', 2);
    // TODO
    myNextion.setComponentText("gpu0", String(up) + "KB/s");
    myNextion.setComponentText("gpu1", String(down) + "KB/s");
  }
}

// TODO: Improve this later
String getValue(String input, char separator, int index) {
  int found  = 0;
  int strIndex[] = {0, -1};
  int maxIndex = input.length() - 1;
  for (int i = 0; i <= maxIndex && found <= index; i++) {
    if (input.charAt(i) == separator || i == maxIndex) {
      found++;
      strIndex[0] = strIndex[1] + 1;
      strIndex[1] = (i == maxIndex) ? i+1: i;
    }
  }
  return found > index ? input.substring(strIndex[0], strIndex[1]): "";
}

