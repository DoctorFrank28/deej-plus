// deej - 6 sliders with on-board filtering
//
// Same serial protocol as the vanilla sketch ("v0|v1|v2|v3|v4|v5\r\n", values 0-1023),
// so it works with any deej build. On top of that it:
//  - averages several ADC reads per slider (discarding the first one after switching channel)
//  - applies a small hysteresis, so a slider sitting still doesn't flood the serial port
//  - maps a small dead zone at both ends of the travel straight to 0 and 1023
//  - re-sends the current values once per second even when nothing moved (a heartbeat
//    that lets deej tell a quiet board apart from a disconnected one)

const int NUM_SLIDERS = 6;

// slider index 0 is wired to A5, index 5 to A0 (same order as the previous firmware)
const int analogInputs[NUM_SLIDERS] = {A5, A4, A3, A2, A1, A0};

const int RAW_MIN = 50;    // raw readings at or below this are 0
const int RAW_MAX = 950;   // raw readings at or above this are full scale
const int OUT_MAX = 1023;  // full scale deej expects

const int OVERSAMPLE = 8;  // ADC reads averaged per slider and cycle
const int HYSTERESIS = 2;  // minimum change (in output units) worth reporting

const unsigned long SAMPLE_INTERVAL_MS = 10;
const unsigned long HEARTBEAT_INTERVAL_MS = 1000;

int reportedValues[NUM_SLIDERS];
unsigned long lastSample = 0;
unsigned long lastSend = 0;

void setup() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
    reportedValues[i] = -1;  // forces a first line right after boot
  }

  Serial.begin(9600);
}

void loop() {
  unsigned long now = millis();
  if (now - lastSample < SAMPLE_INTERVAL_MS) {
    return;
  }
  lastSample = now;

  bool changed = false;

  for (int i = 0; i < NUM_SLIDERS; i++) {
    int value = readSlider(analogInputs[i]);
    int reported = reportedValues[i];

    bool atEdge = (value == 0 || value == OUT_MAX) && value != reported;
    if (reported < 0 || abs(value - reported) >= HYSTERESIS || atEdge) {
      reportedValues[i] = value;
      changed = true;
    }
  }

  if (changed || now - lastSend >= HEARTBEAT_INTERVAL_MS) {
    sendSliderValues();
    lastSend = now;
  }
}

// returns the slider position scaled to 0..OUT_MAX
int readSlider(int pin) {
  // the ADC shares one sample-and-hold capacitor between channels: the first
  // reading after switching can still carry the previous channel's voltage
  analogRead(pin);

  long sum = 0;
  for (int n = 0; n < OVERSAMPLE; n++) {
    sum += analogRead(pin);
  }

  long scaled = (sum - (long)RAW_MIN * OVERSAMPLE) * OUT_MAX / ((long)(RAW_MAX - RAW_MIN) * OVERSAMPLE);
  return (int)constrain(scaled, 0L, (long)OUT_MAX);
}

void sendSliderValues() {
  for (int i = 0; i < NUM_SLIDERS; i++) {
    Serial.print(reportedValues[i]);
    if (i < NUM_SLIDERS - 1) {
      Serial.print('|');
    }
  }
  Serial.print("\r\n");
}
