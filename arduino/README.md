# Arduino sketches

[Leggimi in italiano](README.it.md)

The board's only job is to read the sliders and send their positions over USB. deej plus does everything else on the PC.

## The protocol

The board sends one line per update over serial, with one value per slider, separated by `|` and ending in a newline:

```
512|1023|0|340|780|1023
```

- **Number of sliders**: the number of values in the line. deej plus counts them by itself, there's nothing to configure. A new count is accepted once two lines in a row agree on it, so a half line read while connecting is ignored.
- **Order**: the first value is slider `0` in `config.yaml`, the second is slider `1`, and so on, regardless of which pins they're wired to.
- **Range**: `0` at the bottom, `slider_max_value` (default `1023`) at the top.
- **Speed**: `baud_rate` in the config must match `Serial.begin()` in the sketch (`9600` in both sketches here).

Any sketch that follows this format works, including the ones from the original deej.

## Which sketch

| | [`deej-6-sliders-filtered`](deej-6-sliders-filtered/deej-6-sliders-filtered.ino) (recommended) | [`deej-5-sliders-vanilla`](deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino) |
|---|---|---|
| Sliders | 6 (A5…A0) | 5 (A0…A4) |
| Readings | average of 8 per slider, first reading after a channel switch discarded | 1 per slider |
| Sends | only when a slider really moves, plus once a second | every 10 ms, always |
| Ends of travel | small dead zones, exactly 0 and 1023 | raw values |
| Lost link detection | yes, with `serial_timeout: 3` | no (it never goes quiet) |

## `deej-6-sliders-filtered`, in detail

Every 10 ms the sketch reads each slider and decides whether there's something worth sending.

1. **Read**: for each slider it reads the pin once and throws that reading away (the ADC shares one capacitor between pins, so the first reading after switching can still carry the previous pin's voltage), then averages 8 readings.
2. **Scale**: readings at or below `RAW_MIN` become `0`, at or above `RAW_MAX` become `1023`, everything in between is stretched linearly. The dead zones make the ends reachable even with sliders that never quite touch 0 V or 5 V.
3. **Hysteresis**: a new value is reported only if it differs from the last reported one by at least `HYSTERESIS`, or if it's exactly at one end. A slider at rest doesn't send anything.
4. **Send**: a line goes out when any slider changed, and in any case once every `HEARTBEAT_INTERVAL_MS`. The heartbeat lets deej plus tell a quiet board from a disconnected one.

The settings at the top of the sketch:

| Setting | Default | Change it when |
|---|---|---|
| `NUM_SLIDERS` | `6` | you have a different number of sliders |
| `analogInputs` | `{A5, A4, A3, A2, A1, A0}` | your wiring is different: the first pin is slider 0 |
| `RAW_MIN` / `RAW_MAX` | `50` / `950` | a slider doesn't reach 0% or 100%: widen the dead zones (raise `RAW_MIN`, lower `RAW_MAX`) |
| `OVERSAMPLE` | `8` | rarely; more is steadier but slower |
| `HYSTERESIS` | `2` | values still flicker at rest: raise it to 3–4 |
| `SAMPLE_INTERVAL_MS` | `10` | rarely |
| `HEARTBEAT_INTERVAL_MS` | `1000` | rarely; keep it under `serial_timeout` |

Matching settings in deej plus (the window's Settings page, or `config.yaml`):

```yaml
baud_rate: 9600
slider_max_value: 1023
serial_timeout: 3   # the heartbeat makes 3 s of silence a lost link
```

## Uploading a sketch

Quit deej plus first (tray icon → Quit): while it runs it holds the serial port and the upload fails with "port busy".

**Arduino IDE**: open the `.ino` file, choose your board (Tools → Board) and port (Tools → Port), then Upload.

**arduino-cli**:

```bat
arduino-cli core install arduino:avr
arduino-cli compile -b arduino:avr:uno arduino/deej-6-sliders-filtered
arduino-cli upload -p COM3 -b arduino:avr:uno arduino/deej-6-sliders-filtered
```

Replace `arduino:avr:uno` with your board (`arduino:avr:nano`, `arduino:avr:mega`…) and `COM3` with its port (`arduino-cli board list` shows it). Then start deej plus again.

## Good to know

- **Which slider is which?** Open Mixer in the deej plus window and move one slider at a time: its card follows. If the order is wrong, reorder `analogInputs` in the sketch or the assignments in deej plus.
- **More than 6 sliders**: an Uno or Nano has 6 analog inputs (A0–A5; the Nano also has A6–A7). For more, use a board with more analog pins (e.g. a Mega) or an analog multiplexer.
- **Clone boards** with a CH340 USB chip may need its driver on older Windows versions. deej plus finds them on `com_port: auto` too.
- **Back up before replacing a sketch** if you don't have its source: `avrdude -p atmega328p -c arduino -P COM3 -b 115200 -U flash:r:backup.hex:i` reads the flash of an Uno without changing it.
