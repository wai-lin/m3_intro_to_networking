#include <LiquidCrystal_I2C.h>

LiquidCrystal_I2C lcd(32,16,2);

/// Constants

const int DATA_PIN = 2;
const int MS_PER_BIT = 100;
const String SOM = "/B";
const String EOM = "/E";
const String MESSAGE = "HELLO_WORLD";
const int MESSAGE_SIZE = MESSAGE.length();

String bitWindow = "";
uint8_t bitWindowCount = 0;

/// Helpers

void sendRawBit(bool b) {
  digitalWrite(DATA_PIN, b ? HIGH : LOW);
  delay(MS_PER_BIT);
}

const String LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ_,./()";
const int ALPHABET_SIZE = 32;
const int BITS_PER_CHAR = 5;

int indexOfChar(char c) {
  for (int i = 0; i < ALPHABET_SIZE; i++) {
    if (LETTERS[i] == c) return i;
  }
  return -1;
}

char indexToChar(int idx) {
  if (idx < 0 || idx >= ALPHABET_SIZE) return '?';
  return LETTERS[idx];
}

void emitBits(int idx, void (*sendBit)(bool)) {
  for (int i = BITS_PER_CHAR - 1; i >= 0; --i) {
    bool b = (idx >> i) & 1;
    sendBit(b);
  }
}


/// Program Functions

// SEND 8 BITS
void sendFramedChar(int idx) {
  // START
  digitalWrite(DATA_PIN, HIGH);
  delay(MS_PER_BIT);

  // 5 DATA BITS (MSB-first) — collect for display
  String dataBits = "";
  for (int i = BITS_PER_CHAR - 1; i >= 0; --i) {
    bool b = (idx >> i) & 1;
    dataBits += (b ? '1' : '0');
    sendRawBit(b);
  }

  // Show only the 5 message bits on line 2
  lcd.setCursor(0, 1);
  lcd.print("                ");
  lcd.setCursor(0, 1);
  lcd.print(dataBits);

  // END
  sendRawBit(false);
}


/// Program Functions
void sendSOMSignal() {
  for (int i = 0; i < SOM.length(); i++) {
    int idx = indexOfChar(SOM[i]);
    if (idx >= 0) {
      sendFramedChar(idx);
    }
  }
}

void sendEOMSignal() {
  for (int i = 0; i < EOM.length(); i++) {
    int idx = indexOfChar(EOM[i]);
    if (idx >= 0) {
      sendFramedChar(idx);
    }
  }
}

void sendMessage() {
  sendSOMSignal();
  for (int i = 0; i < MESSAGE_SIZE; i++) {
    int idx = indexOfChar(MESSAGE[i]);
    if (idx >= 0) {
      sendFramedChar(idx);
    }
  }
  sendEOMSignal();
}

/// Main Program

void setup() {
  pinMode(DATA_PIN, OUTPUT);

  lcd.begin(16, 2);
  lcd.init();
  lcd.backlight();
  lcd.print(MESSAGE);
  lcd.setCursor(0, 0);
  digitalWrite(DATA_PIN, HIGH);
}

void loop() {
  sendMessage();
}
