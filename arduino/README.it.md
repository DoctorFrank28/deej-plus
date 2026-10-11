# Sketch Arduino

[Read in English](README.md)

La scheda ha un solo compito: leggere gli slider e mandarne la posizione via USB. Tutto il resto lo fa deej plus sul PC.

## Il protocollo

La scheda manda via seriale una riga per ogni aggiornamento, con un valore per slider separato da `|` e un a capo in fondo:

```
512|1023|0|340|780|1023
```

- **Numero di slider**: è il numero di valori nella riga. deej plus li conta da solo, non c'è niente da configurare. Un numero nuovo viene accettato quando due righe di fila sono d'accordo, così la mezza riga letta mentre si collega viene ignorata.
- **Ordine**: il primo valore è lo slider `0` del `config.yaml`, il secondo lo slider `1` e così via, qualunque siano i pin a cui sono collegati.
- **Scala**: `0` in basso, `slider_max_value` (predefinito `1023`) in alto.
- **Velocità**: `baud_rate` nel config deve coincidere con `Serial.begin()` nello sketch (`9600` in entrambi gli sketch qui).

Funziona qualsiasi sketch che segue questo formato, compresi quelli del deej originale.

## Quale sketch

| | [`deej-6-sliders-filtered`](deej-6-sliders-filtered/deej-6-sliders-filtered.ino) (consigliato) | [`deej-5-sliders-vanilla`](deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino) |
|---|---|---|
| Slider | 6 (A5…A0) | 5 (A0…A4) |
| Letture | media di 8 per slider, scartata la prima dopo il cambio di canale | 1 per slider |
| Invio | solo quando uno slider si muove davvero, più una volta al secondo | ogni 10 ms, sempre |
| Estremi della corsa | piccole zone morte, esattamente 0 e 1023 | valori grezzi |
| Rileva il collegamento perso | sì, con `serial_timeout: 3` | no (non resta mai in silenzio) |

## `deej-6-sliders-filtered` nel dettaglio

Ogni 10 ms lo sketch legge tutti gli slider e decide se c'è qualcosa da mandare.

1. **Lettura**: per ogni slider legge il pin una volta e butta via quella lettura (l'ADC usa un solo condensatore per tutti i pin, quindi la prima lettura dopo il cambio può portarsi dietro la tensione del pin precedente), poi fa la media di 8 letture.
2. **Scala**: le letture fino a `RAW_MIN` diventano `0`, da `RAW_MAX` in su diventano `1023`, quelle in mezzo vengono stirate in proporzione. Le zone morte fanno arrivare agli estremi anche gli slider che non toccano mai del tutto 0 V o 5 V.
3. **Isteresi**: un valore nuovo viene mandato solo se differisce da quello mandato prima di almeno `HYSTERESIS`, oppure se è esattamente a un estremo. Uno slider fermo non manda niente.
4. **Invio**: parte una riga quando uno slider è cambiato, e comunque ogni `HEARTBEAT_INTERVAL_MS`. Questo segnale periodico permette a deej plus di distinguere una scheda ferma da una scollegata.

Le impostazioni in cima allo sketch:

| Impostazione | Predefinito | Cambiala quando |
|---|---|---|
| `NUM_SLIDERS` | `6` | hai un numero diverso di slider |
| `analogInputs` | `{A5, A4, A3, A2, A1, A0}` | il cablaggio è diverso: il primo pin è lo slider 0 |
| `RAW_MIN` / `RAW_MAX` | `50` / `950` | uno slider non arriva a 0% o 100%: allarga le zone morte (alza `RAW_MIN`, abbassa `RAW_MAX`) |
| `OVERSAMPLE` | `8` | raramente; di più è più stabile ma più lento |
| `HYSTERESIS` | `2` | i valori tremolano ancora da fermi: portala a 3–4 |
| `SAMPLE_INTERVAL_MS` | `10` | raramente |
| `HEARTBEAT_INTERVAL_MS` | `1000` | raramente; tienilo sotto `serial_timeout` |

Le impostazioni corrispondenti in deej plus (pagina Impostazioni della finestra, oppure `config.yaml`):

```yaml
baud_rate: 9600
slider_max_value: 1023
serial_timeout: 3   # con il segnale periodico, 3 s di silenzio vogliono dire collegamento perso
```

## Caricare uno sketch

Prima chiudi deej plus (icona nel tray → Esci): mentre è aperto tiene occupata la porta seriale e il caricamento fallisce con "porta occupata".

**Arduino IDE**: apri il file `.ino`, scegli la scheda (Strumenti → Scheda) e la porta (Strumenti → Porta), poi Carica.

**arduino-cli**:

```bat
arduino-cli core install arduino:avr
arduino-cli compile -b arduino:avr:uno arduino/deej-6-sliders-filtered
arduino-cli upload -p COM3 -b arduino:avr:uno arduino/deej-6-sliders-filtered
```

Sostituisci `arduino:avr:uno` con la tua scheda (`arduino:avr:nano`, `arduino:avr:mega`…) e `COM3` con la sua porta (te la mostra `arduino-cli board list`). Poi riavvia deej plus.

## Buono a sapersi

- **Qual è quale slider?** Apri il Mixer nella finestra di deej plus e muovi uno slider alla volta: la sua card lo segue. Se l'ordine è sbagliato, riordina `analogInputs` nello sketch oppure le assegnazioni in deej plus.
- **Più di 6 slider**: Uno e Nano hanno 6 ingressi analogici (A0–A5; il Nano anche A6–A7). Per averne di più serve una scheda con più pin analogici (per esempio una Mega) o un multiplexer analogico.
- **Le schede clone** con il chip USB CH340 possono richiedere il suo driver sulle versioni più vecchie di Windows. deej plus le trova anche con `com_port: auto`.
- **Fai un backup prima di sostituire uno sketch** di cui non hai il sorgente: `avrdude -p atmega328p -c arduino -P COM3 -b 115200 -U flash:r:backup.hex:i` legge la flash di un Uno senza modificarla.
