#include <SoftwareSerial.h>
#include <Nextion.h>

SoftwareSerial nextion(2, 3); // Plug Nextion TX to pin 2 and RX to pin 3 of Arduino

Nextion myNextion(nextion, 9600); // Create a Nextion object named myNextion using the nextion serial port @ 9600bps

String inputString = "";

void setup() {
  Serial.begin(9600);
  myNextion.init();
  inputString.reserve(100);
}

/* Read commands from USB Serial port, then apply to LCD.
 * Command format: Type|Values$
 *   - First character determines the command type:
 *     + 0: Config
 *     + 1: CPU stats
 *     + 2: Memory stats
 *     + 3: GPU stats
 *     + 4: Network stats
 *     + z: Alert
 *   - Depends on command type, there may have one or many values.
 *     Values are separated by | character.
 *   - Command ends with $ character.
 *   - For alert command, first value dertermines the alert type,
 *     second value determines the alert status (0 = OFF, 1 = ON).
 * Examples:
 *  - 1|10|0$
 *  - 2|34|2951$
 *  - 4|13/227$
 */
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

  // Need bracket for each case otherwise we will face the "crosses initialization error" - Somewhat stupid :(
  switch (cmd.charAt(0)) {
    case '0':
      // TODO
      break;
    case '1': { // CPU
      String load = getValue(input, '|', 1);
      String temp = getValue(input, '|', 2);
      myNextion.setComponentText("cpu0", load + "%");
      myNextion.setComponentText("cpu1", temp + "*C");
      break;
    }
    case '2': { // MEM
      String load = getValue(input, '|', 1);
      String usage = getValue(input, '|', 2);
      myNextion.setComponentText("mem0", load + "%");
      myNextion.setComponentText("mem1", usage + "MB");
      break;
    }
    case '3': { // GPU
      String load = getValue(input, '|', 1);
      String usage = getValue(input, '|', 2);
      myNextion.setComponentText("gpu0", load + "%");
      myNextion.setComponentText("gpu1", usage + "MB");
      break;
    }
    case '4': { // NET
      String down = getValue(input, '|', 1);
      String up = getValue(input, '|', 2);
      myNextion.setComponentText("net0", down + "/" + up + "KBps");
      break;
    }
    case 'z': // Alert
      applyAlert(input);
      break;
    default:
      return;
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

void applyAlert(String cmd) {
  String alertType = getValue(cmd, '|', 1);
  String alertColor = "10730"; // Background color
  if (getValue(cmd, '|', 2).charAt(0) == '1') { // Alert status
    alertColor = "57798"; // Red
  }
  switch (alertType.charAt(0)) {
    case '1':
      myNextion.sendCommand(string2char("page0.cpu_alert.bco=" + alertColor));
      break;
    case '2':
      myNextion.sendCommand(string2char("page0.mem_alert.bco=" + alertColor));
      break;
    case '3':
      myNextion.sendCommand(string2char("page0.gpu_alert.bco=" + alertColor));
      break;
    case '4':
      myNextion.sendCommand(string2char("page0.net_alert.bco=" + alertColor));
      break;
    default:
      return;
  }
}

char* string2char(String cmd) {
    if (cmd.length() != 0) {
        char *p = const_cast<char*>(cmd.c_str());
        return p;
    }
}
