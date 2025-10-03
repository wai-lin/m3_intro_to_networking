#include <LiquidCrystal_I2C.h>

LiquidCrystal_I2C lcd(32,16,2);

/// Constants

const int DATA_PIN = 2;
const int MS_PER_BIT = 100;

const String LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ_,./()";
const int ALPHABET_SIZE = 32;
const int BITS_PER_CHAR = 5;

bool isCollectingMsg = false;
char last1 = '\0', last2 = '\0', hold = '\0';

String messageWindow = "";
String bitWindow = "";
uint8_t bitWindowCount = 0;


/// Helpers

int ledPin = 13;
bool shouldBlinkLED = false;
int blinkState = LOW;
int blinkFor = 0;

void blinkLED() {
  if (shouldBlinkLED) {
    blinkState = blinkState ? LOW : HIGH;
    digitalWrite(ledPin, blinkState);
    blinkFor += MS_PER_BIT;

    if (blinkFor >= 1000) {
      blinkFor = 0;
      blinkState = LOW;
      shouldBlinkLED = false;
    }
  }
}

int sampleBit() {
  // delay(MS_PER_BIT);
  return digitalRead(DATA_PIN);
}

int bitStrToDecimal(const String &bitStr) {
  if (bitStr.length() <= 0) return -1;
  for (int i = 0; i < bitStr.length(); i++) {
    if (bitStr[i] != '0' && bitStr[i] != '1') return -1;
  }
  return (int)strtol(bitStr.c_str(), nullptr, 2);
}


/// Program Functions

void displayDataBits(const String &bits) {
  lcd.setCursor(0, 1);
  lcd.print("                ");
  lcd.setCursor(0, 1);
  lcd.print(bits);
}

void receiveMessage() {
  // Wait for START Bit
  while(sampleBit() == LOW) { delay(1); };
  delay(MS_PER_BIT * 1.5);
  
  // READ 5 MSG Bits
  String dataBits = "";
  for (int i = 0; i < 5; i++) {
    blinkLED();
    dataBits += sampleBit() ? '1' : '0';
    delay(MS_PER_BIT);
  }
  displayDataBits(dataBits);
  
  // Bits Operation
  int letterIndex = bitStrToDecimal(dataBits);
  char ch = '?';
  if (letterIndex >= 0 && letterIndex < ALPHABET_SIZE) {
    ch = LETTERS[letterIndex];
  }

  // Read message
  last1 = last2;
  last2 = ch;
  if (last1 == '/' && last2 == 'B') {
    hold = '\0';
    messageWindow = "";
    isCollectingMsg = true;
  } else if (last1 == '/' && last2 == 'E') {
    hold = '\0';
    messageWindow = "";
    shouldBlinkLED = true;
    isCollectingMsg = false;
    lcd.clear();
  } else {
    if (isCollectingMsg) {
      if (hold != '\0') {
        messageWindow += hold;
      }
      hold = ch;
    }
  }

  // Keep waiting until we get a STOP Bit
  while(sampleBit() == HIGH) { delay(1); };
}


/// Main Program

void setup() {
  pinMode(DATA_PIN, INPUT);
  lcd.init();
  lcd.backlight();
}

void loop() {
  lcd.setCursor(0, 0);
  lcd.print(messageWindow);
  receiveMessage();
}
