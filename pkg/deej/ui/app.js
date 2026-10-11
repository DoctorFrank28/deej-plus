/*
 * deej mini app: frontend statico (vanilla JS, nessuna build).
 * Servito dall'exe con go:embed. API sulla stessa origine:
 *   GET  /api/state, /api/sessions, /api/ports, /api/events (SSE)
 *   POST /api/settings, /api/mapping, /api/reconnect, /api/open  (header X-Deej: 1)
 * Modalità di prova: ?mock=1 [&cycle=1] [&fail=1] [&drop=1] [&lang=en]
 */
'use strict';

(function () {
  const qs = new URLSearchParams(location.search);
  const MOCK = qs.get('mock') === '1';
  const REDUCED = window.matchMedia('(prefers-reduced-motion: reduce)');
  const DARK = window.matchMedia('(prefers-color-scheme: dark)');

  /* =====================================================================
   * Traduzioni. Per aggiungere una lingua basta aggiungere un oggetto qui
   * (meta.name = nome della lingua scritto nella lingua stessa).
   * Chiavi mancanti: si usa l'inglese. Plurali: chiave.one / chiave.other
   * (scelti con Intl.PluralRules). Segnaposto: {nome}.
   * ===================================================================== */
  const I18N = {
    it: {
      'meta.name': 'Italiano',
      'skip': 'Vai al contenuto',
      'page.title': '{view} · deej plus',
      'nav.label': 'Sezioni',
      'nav.mixer': 'Mixer',
      'nav.assign': 'Assegnazioni',
      'nav.settings': 'Impostazioni',
      'brand.sub': '{version}',
      'device.label': 'Dispositivo',
      'device.none': 'Nessun dispositivo',
      'device.port': '{port} · {baud} baud',
      'link.down': 'deej plus non risponde, riprovo…',

      'state.loading': 'Collegamento a deej…',
      'state.connected': 'Connesso',
      'state.connectedPort': 'Connesso · {port}',
      'state.reconnecting': 'Riconnessione…',
      'state.disconnected': 'Scollegato',
      'state.busy': 'Porta occupata',

      'mixer.subtitle': 'Posizione degli slider in tempo reale.',
      'mixer.reconnect': 'Ricollega',
      'mixer.slidersLabel': 'Slider',
      'mixer.empty': 'Nessuno slider nel config. Aggiungili da',
      'mixer.legend': '= app che sta suonando adesso',
      'mixer.lastEvent': 'Ultimo evento: {text}',
      'mixer.chipsLabel': 'App comandate',
      'mixer.noTargets': 'Nessuna app',
      'mixer.live': '(suona adesso)',
      'mixer.more': 'e altre {n}: {list}',
      'slider.placeholder': 'Slider {n}',

      'banner.reconnTitle': 'Arduino scollegato',
      'banner.reconnTitleAt': 'Arduino scollegato alle {time}',
      'banner.reconnAuto.one': 'Riprovo da solo ogni secondo{attempt}. Gli slider restano dove li hai lasciati.',
      'banner.reconnAuto.other': 'Riprovo da solo ogni {n} secondi{attempt}. Gli slider restano dove li hai lasciati.',
      'banner.attempt': ' (tentativo {n})',
      'banner.reconnManual': 'Premi Ricollega per riprovare. Gli slider restano dove li hai lasciati.',
      'banner.offTitle': 'Nessun Arduino trovato',
      'banner.offTitlePort': 'Nessun Arduino su {port}',
      'banner.offAuto': "Collegalo: deej lo riprende appena compare. Oppure scegli un'altra porta.",
      'banner.offManual': "Collegalo e premi Ricollega. Oppure scegli un'altra porta.",
      'banner.choosePort': 'Scegli porta',
      'banner.busyTitle': 'Porta occupata',
      'banner.busyText': '{port} è aperta da un altro programma (es. Arduino IDE). Riprovo quando si libera.',
      'banner.busyTextNoPort': 'La porta è aperta da un altro programma (es. Arduino IDE). Riprovo quando si libera.',

      'toast.reconnectFail': 'Ricollega non riuscito: {error}',
      'toast.openFail': 'Impossibile aprire: {error}',
      'save.saving': 'Salvataggio…',
      'save.ok': 'Salvato in config.yaml',
      'save.error': 'Non salvato: {error}',
      'error.http': 'Errore {status}',
      'error.network': 'deej plus non risponde',
      'error.interval': 'Scegli un valore da 1 a 30 secondi.',

      'assign.subtitle': 'Quali app comanda ogni slider. Le modifiche si salvano da sole.',
      'assign.addSlider': 'Aggiungi slider',
      'assign.unassigned': 'Non assegnate',
      'assign.unassignedHint': 'App che fanno audio ma non sono su nessuno slider.',
      'assign.unassignedNote': "Le app non assegnate seguono lo slider con «Tutte le non assegnate», se c'è.",
      'assign.nameLabel': 'Nome dello slider {n}',
      'assign.add': 'Aggiungi',
      'assign.addTo': 'Aggiungi app allo slider {name}',
      'assign.remove': 'Togli {target} da {name}',
      'assign.assignTo': 'Assegna a…',
      'assign.assignLabel': 'Assegna {app} a',
      'assign.noneUnassigned': 'Tutte le app aperte sono già su uno slider.',
      'assign.playing': 'suona adesso',
      'assign.idle': 'aperta, in silenzio',
      'assign.option': '{n} · {name}',

      'pop.dialog': "Aggiungi un'app allo slider {name}",
      'pop.search': 'Cerca app',
      'pop.placeholder': 'Cerca o scrivi un .exe…',
      'pop.list': 'App disponibili',
      'pop.playing': 'Stanno suonando adesso',
      'pop.open': 'Altre app aperte',
      'pop.specials': 'Speciali',
      'pop.custom': 'Aggiungi «{text}»',
      'pop.manual': 'Scrivi a mano',
      'pop.emptyAssigned': 'Già assegnata a questo slider.',
      'pop.empty': 'Nessuna app da aggiungere.',

      'special.master': 'Volume generale',
      'special.mic': 'Microfono predefinito',
      'special.system': 'Suoni di sistema',
      'special.deej.current': 'App in primo piano',
      'special.deej.unmapped': 'Tutte le non assegnate',

      'settings.subtitle': 'Si applicano subito, senza riavviare deej.',
      'settings.conn': 'Connessione',
      'settings.connHint': "Come deej trova e tiene collegato l'Arduino.",
      'settings.port': 'Porta',
      'settings.baud': 'Velocità',
      'settings.portHint': 'In automatico cerca la scheda dal suo ID USB: se Windows le cambia porta, la ritrova lo stesso.',
      'settings.portAuto': 'Automatica',
      'settings.portAutoFound': 'Automatica · {desc} ({port})',
      'settings.portMissing': '{port} · non trovata',
      'settings.baudOption': '{n} baud',
      'settings.autoReconnect': 'Ricollega da solo',
      'settings.autoReconnectHint': 'Se il cavo si stacca o il PC va in sospensione.',
      'settings.notify': 'Avvisami quando si scollega',
      'settings.notifyHint': 'Notifica di Windows, una sola volta.',
      'settings.retryEvery': 'Riprova ogni',
      'settings.seconds': 'secondi',
      'settings.system': 'Sistema',
      'settings.startup': 'Avvia con Windows',
      'settings.startupHint': 'Parte in background all’accesso.',
      'settings.openUi': 'Apri questa finestra all’avvio',
      'settings.openUiHint': 'Altrimenti resta solo l’icona nel tray.',
      'settings.language': 'Lingua',
      'settings.configFile': 'File di configurazione',
      'settings.openFolder': 'Apri cartella',
      'settings.openLog': 'Log',
      'settings.sliders': 'Slider e filtro rumore',
      'settings.slidersHint': 'Decide quando un movimento è tuo e quando è solo rumore del potenziometro.',
      'settings.invert': 'Inverti slider',
      'settings.invertHint': 'In alto 0%, in basso 100%.',
      'settings.filter': 'Filtro',
      'lang.auto': 'Automatica',

      'preset.low': 'Basso',
      'preset.default': 'Normale',
      'preset.high': 'Alto',
      'preset.custom': 'Su misura',

      'filter.deadband': 'Zona morta',
      'filter.deadbandHint': 'Da fermo, sotto questa variazione non cambia niente.',
      'filter.smoothing': 'Levigatura',
      'filter.smoothingHint': 'Media mobile sui valori grezzi: più forte è più stabile ma un filo più lenta.',
      'filter.edgeSnap': 'Aggancio ai bordi',
      'filter.edgeSnapHint': 'Vicino agli estremi va dritto a 0% o 100%.',
      'filter.smoothOff': 'spenta',
      'filter.edgeOff': 'spento',
      'filter.light': 'leggera',
      'filter.medium': 'media',
      'filter.strong': 'forte',

      'chart.title': 'Anteprima dal vivo',
      'chart.raw': 'grezzo',
      'chart.value': 'applicato',
      'chart.caption': "Ultimi 10 secondi. La linea sottile è la lettura grezza dell'Arduino, quella spessa il volume applicato dopo il filtro: da fermo deve restare piatta.",
      'chart.waiting': 'In attesa di dati…',
      'chart.aria': 'Anteprima dello slider {name}: grezzo {raw}, applicato {value}.',
      'chart.ariaWaiting': 'Anteprima dello slider: in attesa di dati.',
      'chart.ago': '−10 s',
      'chart.now': 'adesso'
    },

    en: {
      'meta.name': 'English',
      'skip': 'Skip to content',
      'page.title': '{view} · deej plus',
      'nav.label': 'Sections',
      'nav.mixer': 'Mixer',
      'nav.assign': 'Assignments',
      'nav.settings': 'Settings',
      'brand.sub': '{version}',
      'device.label': 'Device',
      'device.none': 'No device',
      'device.port': '{port} · {baud} baud',
      'link.down': 'deej plus is not responding, retrying…',

      'state.loading': 'Connecting to deej…',
      'state.connected': 'Connected',
      'state.connectedPort': 'Connected · {port}',
      'state.reconnecting': 'Reconnecting…',
      'state.disconnected': 'Disconnected',
      'state.busy': 'Port busy',

      'mixer.subtitle': 'Live slider positions.',
      'mixer.reconnect': 'Reconnect',
      'mixer.slidersLabel': 'Sliders',
      'mixer.empty': 'No sliders in the config. Add them in',
      'mixer.legend': '= app playing right now',
      'mixer.lastEvent': 'Last event: {text}',
      'mixer.chipsLabel': 'Controlled apps',
      'mixer.noTargets': 'No apps',
      'mixer.live': '(playing now)',
      'mixer.more': 'and {n} more: {list}',
      'slider.placeholder': 'Slider {n}',

      'banner.reconnTitle': 'Arduino disconnected',
      'banner.reconnTitleAt': 'Arduino disconnected at {time}',
      'banner.reconnAuto.one': 'Retrying on my own every second{attempt}. Sliders stay where you left them.',
      'banner.reconnAuto.other': 'Retrying on my own every {n} seconds{attempt}. Sliders stay where you left them.',
      'banner.attempt': ' (attempt {n})',
      'banner.reconnManual': 'Press Reconnect to try again. Sliders stay where you left them.',
      'banner.offTitle': 'No Arduino found',
      'banner.offTitlePort': 'No Arduino on {port}',
      'banner.offAuto': 'Plug it in: deej picks it up as soon as it shows up. Or choose another port.',
      'banner.offManual': 'Plug it in and press Reconnect. Or choose another port.',
      'banner.choosePort': 'Choose Port',
      'banner.busyTitle': 'Port busy',
      'banner.busyText': '{port} is open in another program (e.g. Arduino IDE). I’ll retry when it’s free.',
      'banner.busyTextNoPort': 'The port is open in another program (e.g. Arduino IDE). I’ll retry when it’s free.',

      'toast.reconnectFail': 'Reconnect failed: {error}',
      'toast.openFail': 'Could not open: {error}',
      'save.saving': 'Saving…',
      'save.ok': 'Saved to config.yaml',
      'save.error': 'Not saved: {error}',
      'error.http': 'Error {status}',
      'error.network': 'deej plus is not responding',
      'error.interval': 'Pick a value from 1 to 30 seconds.',

      'assign.subtitle': 'Which apps each slider controls. Changes save automatically.',
      'assign.addSlider': 'Add Slider',
      'assign.unassigned': 'Unassigned',
      'assign.unassignedHint': 'Apps making sound that aren’t on any slider.',
      'assign.unassignedNote': 'Unassigned apps follow the slider with “All unassigned”, if there is one.',
      'assign.nameLabel': 'Name of slider {n}',
      'assign.add': 'Add',
      'assign.addTo': 'Add app to slider {name}',
      'assign.remove': 'Remove {target} from {name}',
      'assign.assignTo': 'Assign to…',
      'assign.assignLabel': 'Assign {app} to',
      'assign.noneUnassigned': 'Every open app is already on a slider.',
      'assign.playing': 'playing now',
      'assign.idle': 'open, silent',
      'assign.option': '{n} · {name}',

      'pop.dialog': 'Add an app to slider {name}',
      'pop.search': 'Search apps',
      'pop.placeholder': 'Search or type an .exe…',
      'pop.list': 'Available apps',
      'pop.playing': 'Playing now',
      'pop.open': 'Other open apps',
      'pop.specials': 'Special',
      'pop.custom': 'Add “{text}”',
      'pop.manual': 'Type it in',
      'pop.emptyAssigned': 'Already on this slider.',
      'pop.empty': 'No apps to add.',

      'special.master': 'Master volume',
      'special.mic': 'Default microphone',
      'special.system': 'System sounds',
      'special.deej.current': 'App in focus',
      'special.deej.unmapped': 'All unassigned',

      'settings.subtitle': 'They apply right away, no need to restart deej.',
      'settings.conn': 'Connection',
      'settings.connHint': 'How deej finds the Arduino and keeps it connected.',
      'settings.port': 'Port',
      'settings.baud': 'Speed',
      'settings.portHint': 'In automatic mode it finds the board by its USB ID: if Windows moves it to another port, it still finds it.',
      'settings.portAuto': 'Automatic',
      'settings.portAutoFound': 'Automatic · {desc} ({port})',
      'settings.portMissing': '{port} · not found',
      'settings.baudOption': '{n} baud',
      'settings.autoReconnect': 'Reconnect automatically',
      'settings.autoReconnectHint': 'If the cable comes loose or the PC goes to sleep.',
      'settings.notify': 'Notify me when it disconnects',
      'settings.notifyHint': 'One Windows notification, just once.',
      'settings.retryEvery': 'Retry every',
      'settings.seconds': 'seconds',
      'settings.system': 'System',
      'settings.startup': 'Start with Windows',
      'settings.startupHint': 'Runs in the background when you sign in.',
      'settings.openUi': 'Open this window at startup',
      'settings.openUiHint': 'Otherwise only the tray icon shows.',
      'settings.language': 'Language',
      'settings.configFile': 'Configuration file',
      'settings.openFolder': 'Open Folder',
      'settings.openLog': 'Log',
      'settings.sliders': 'Sliders and Noise Filter',
      'settings.slidersHint': 'Decides when a movement is yours and when it’s just potentiometer noise.',
      'settings.invert': 'Invert sliders',
      'settings.invertHint': 'Top is 0%, bottom is 100%.',
      'settings.filter': 'Filter',
      'lang.auto': 'Automatic',

      'preset.low': 'Low',
      'preset.default': 'Normal',
      'preset.high': 'High',
      'preset.custom': 'Custom',

      'filter.deadband': 'Dead zone',
      'filter.deadbandHint': 'When still, changes below this do nothing.',
      'filter.smoothing': 'Smoothing',
      'filter.smoothingHint': 'Moving average over raw values: stronger is steadier but a touch slower.',
      'filter.edgeSnap': 'Edge snap',
      'filter.edgeSnapHint': 'Near the ends it jumps straight to 0% or 100%.',
      'filter.smoothOff': 'off',
      'filter.edgeOff': 'off',
      'filter.light': 'light',
      'filter.medium': 'medium',
      'filter.strong': 'strong',

      'chart.title': 'Live Preview',
      'chart.raw': 'raw',
      'chart.value': 'applied',
      'chart.caption': 'Last 10 seconds. The thin line is the Arduino’s raw reading, the thick one the volume applied after the filter: it should stay flat when the slider is still.',
      'chart.waiting': 'Waiting for data…',
      'chart.aria': 'Preview of slider {name}: raw {raw}, applied {value}.',
      'chart.ariaWaiting': 'Slider preview: waiting for data.',
      'chart.ago': '−10 s',
      'chart.now': 'now'
    }
  };
  const FALLBACK_LANG = 'en';
  let LANG = FALLBACK_LANG;
  const fmtCache = new Map();

  function resolveLang(setting) {
    if (setting && setting !== 'auto' && I18N[setting]) return setting;
    const prefs = navigator.languages && navigator.languages.length ? navigator.languages : [navigator.language || ''];
    for (const l of prefs) {
      const base = String(l).toLowerCase().split('-')[0];
      if (I18N[base]) return base;
    }
    return FALLBACK_LANG;
  }
  function lookup(key) {
    const d = I18N[LANG];
    if (d && Object.prototype.hasOwnProperty.call(d, key)) return d[key];
    const f = I18N[FALLBACK_LANG];
    if (Object.prototype.hasOwnProperty.call(f, key)) return f[key];
    return null;
  }
  function t(key, vars) {
    let s = lookup(key);
    if (s == null) return key;
    if (vars) s = s.replace(/\{(\w+)\}/g, (m, k) => (vars[k] != null ? String(vars[k]) : m));
    return s;
  }
  function tp(key, n, vars) {
    let cat = 'other';
    try { cat = plural().select(n); } catch (e) { /* lingua sconosciuta */ }
    const k = lookup(`${key}.${cat}`) != null ? `${key}.${cat}` : `${key}.other`;
    return t(k, Object.assign({ n: fmtNum(n, 0) }, vars));
  }
  function cached(kind, opts, make) {
    const k = `${LANG}|${kind}|${JSON.stringify(opts || {})}`;
    if (!fmtCache.has(k)) fmtCache.set(k, make());
    return fmtCache.get(k);
  }
  const plural = () => cached('plural', null, () => new Intl.PluralRules(LANG));
  function fmtNum(v, maxFrac = 1, minFrac = 0) {
    const o = { maximumFractionDigits: maxFrac, minimumFractionDigits: minFrac };
    return cached('num', o, () => new Intl.NumberFormat(LANG, o)).format(v);
  }
  // v è una frazione (0.47 → 47%)
  function fmtPct(v, maxFrac = 0, minFrac = 0) {
    const o = { style: 'percent', maximumFractionDigits: maxFrac, minimumFractionDigits: minFrac };
    return cached('pct', o, () => new Intl.NumberFormat(LANG, o)).format(v);
  }
  function fmtTime(iso) {
    const d = new Date(iso);
    if (!iso || isNaN(d.getTime())) return '';
    return cached('time', null, () => new Intl.DateTimeFormat(LANG, { hour: '2-digit', minute: '2-digit' })).format(d);
  }

  /* ===================================================================== */
  const SPECIAL_KEYS = ['master', 'mic', 'system', 'deej.current', 'deej.unmapped'];
  const SPECIAL_MENU = ['deej.current', 'deej.unmapped', 'system', 'master', 'mic'];
  const SESSION_SPECIAL = new Set(['master', 'system', 'mic']);
  const PRESETS = {
    low: { deadband: 0.2, smoothing: 0, edgeSnap: 0.5 },
    default: { deadband: 0.4, smoothing: 0, edgeSnap: 1 },
    high: { deadband: 1.0, smoothing: 0.5, edgeSnap: 2 }
  };
  const VIEWS = { mixer: 'nav.mixer', assegnazioni: 'nav.assign', impostazioni: 'nav.settings' };
  const BAUDS = [9600, 19200, 38400, 57600, 115200];
  const HISTORY_MS = 10000;

  /* ---------- utilità DOM ---------- */
  const $ = (s, r) => (r || document).querySelector(s);
  const $$ = (s, r) => Array.from((r || document).querySelectorAll(s));
  const lc = (s) => String(s == null ? '' : s).toLowerCase();
  const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const clone = (o) => JSON.parse(JSON.stringify(o));

  function h(tag, attrs, ...kids) {
    const el = document.createElement(tag);
    if (attrs) {
      for (const k of Object.keys(attrs)) {
        const v = attrs[k];
        if (v == null || v === false) continue;
        if (k === 'class') el.className = v;
        else if (k.startsWith('on') && typeof v === 'function') el.addEventListener(k.slice(2), v);
        else el.setAttribute(k, v === true ? '' : String(v));
      }
    }
    for (const c of kids.flat(Infinity)) {
      if (c == null || c === false) continue;
      el.append(c instanceof Node ? c : String(c));
    }
    return el;
  }
  const ICONS = {
    x: '<path d="M18 6 6 18"/><path d="m6 6 12 12"/>',
    plus: '<path d="M12 5v14"/><path d="M5 12h14"/>',
    check: '<path d="M20 6 9 17l-5-5"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    loader: '<path d="M21 12a9 9 0 1 1-6.22-8.56"/>',
    alert: '<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>',
    plug: '<path d="M12 22v-5"/><path d="M9 8V2"/><path d="M15 8V2"/><path d="M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z"/>'
  };
  function icon(name, size, cls, sw) {
    const tpl = document.createElement('template');
    tpl.innerHTML = `<svg class="icon ${cls || ''}" aria-hidden="true" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${sw || 2}" stroke-linecap="round" stroke-linejoin="round">${ICONS[name]}</svg>`;
    return tpl.content.firstChild;
  }
  function setText(el, s) { if (el && el.textContent !== s) el.textContent = s; }
  function withFocus(container, fn) {
    const a = document.activeElement;
    const key = a && container.contains(a) ? a.getAttribute('data-focus-key') : null;
    fn();
    if (key) {
      const el = container.querySelector(`[data-focus-key="${CSS.escape(key)}"]`);
      if (el) el.focus({ preventScroll: true });
    }
  }

  /* ---------- dominio ---------- */
  const sliderLabel = (s) => (s && s.name) || t('slider.placeholder', { n: s ? s.index : '' });
  function targetInfo(target) {
    const k = lc(target);
    if (SPECIAL_KEYS.includes(k)) { const l = t('special.' + k); return { kind: 'special', label: l, short: l }; }
    if (/\.exe$/i.test(target) || !/\s/.test(target)) return { kind: 'process', label: target, short: target.replace(/\.exe$/i, '') };
    return { kind: 'device', label: target, short: target };
  }
  function prettyName(key) {
    const b = String(key).replace(/\.exe$/i, '');
    return b.charAt(0).toUpperCase() + b.slice(1);
  }
  function activeSet() {
    return new Set(S.sessions.filter((x) => x.active).map((x) => lc(x.key)));
  }
  function isLive(target, act) {
    const k = lc(target);
    if (k === 'deej.unmapped') return S.sessions.some((x) => x.active && !x.mapped && !SESSION_SPECIAL.has(lc(x.key)));
    if (SPECIAL_KEYS.includes(k)) return false;
    return act.has(k);
  }
  function effectiveFilter(set) {
    return PRESETS[set.noiseReduction] ? Object.assign({}, PRESETS[set.noiseReduction]) : Object.assign({}, PRESETS.default, set.filter || {});
  }

  /* =====================================================================
   * API: reale o mock
   * ===================================================================== */
  async function toError(r) {
    let msg = t('error.http', { status: r.status });
    try { const j = await r.json(); if (j && j.error) msg = j.error; } catch (e) { /* body non JSON */ }
    return new Error(msg);
  }
  const httpApi = {
    async get(path) {
      let r;
      try { r = await fetch(path, { cache: 'no-store' }); } catch (e) { throw new Error(t('error.network')); }
      if (!r.ok) throw await toError(r);
      return r.json();
    },
    async post(path, body) {
      let r;
      try {
        r = await fetch(path, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Deej': '1' },
          body: JSON.stringify(body == null ? {} : body)
        });
      } catch (e) { throw new Error(t('error.network')); }
      if (!r.ok) throw await toError(r);
      if (r.status === 204) return null;
      const txt = await r.text();
      return txt ? JSON.parse(txt) : null;
    },
    events() { return new EventSource('/api/events'); }
  };
  const api = MOCK ? createMock() : httpApi;

  /* =====================================================================
   * Stato
   * ===================================================================== */
  const S = { state: null, byIndex: new Map(), sessions: [], ports: [], view: 'mixer', linkDown: false };
  const hist = new Map(); // index -> [{t, raw, value}]

  function pushSample(index, raw, value, now) {
    if (!Number.isFinite(raw) || !Number.isFinite(value)) return;
    let arr = hist.get(index);
    if (!arr) { arr = []; hist.set(index, arr); }
    arr.push({ t: now, raw, value });
    while (arr.length > 2 && arr[1].t < now - HISTORY_MS - 1000) arr.shift();
  }

  function applyState(st) {
    if (!st || !st.settings || !st.connection) return;
    st.sliders = (st.sliders || []).slice().sort((a, b) => a.index - b.index);
    st.sliders.forEach((s) => { s.targets = s.targets || []; s.name = s.name || ''; });
    st.settings.filter = Object.assign({}, PRESETS.default, st.settings.filter || {});
    if (pendingSettings) mergeSettings(st.settings, pendingSettings);
    S.state = st;
    S.byIndex = new Map(st.sliders.map((s) => [s.index, s]));
    const now = performance.now();
    st.sliders.forEach((s) => pushSample(s.index, s.raw, s.value, now));
    const lang = resolveLang(st.settings.language);
    if (lang !== LANG) { setLanguage(lang); return; }
    renderAll();
  }
  function mergeSettings(target, patch) {
    for (const k of Object.keys(patch)) {
      if (k === 'filter') target.filter = Object.assign({}, target.filter, patch.filter);
      else target[k] = patch[k];
    }
  }

  function renderAll() {
    if (!S.state) return;
    renderChrome();
    renderConnection();
    renderMixer();
    renderAssign();
    renderSettings();
    renderChartSelect();
  }

  /* ---------- lingua ---------- */
  function setLanguage(lang) {
    LANG = lang;
    document.documentElement.lang = lang;
    $$('[data-i18n]').forEach((el) => setText(el, t(el.dataset.i18n)));
    $$('[data-i18n-attr]').forEach((el) => {
      el.dataset.i18nAttr.split(';').forEach((pair) => {
        const [attr, key] = pair.split(':').map((x) => x.trim());
        if (attr && key) el.setAttribute(attr, t(key));
      });
    });
    document.title = t('page.title', { view: t(VIEWS[S.view]) });
    // invalida le cache di rendering: tutto il testo dinamico va rifatto
    mixerKey = bannerKey = unKey = '';
    rowRefs.forEach((r) => { r.chipsKey = ''; });
    $$('select[data-key]').forEach((s) => { s.dataset.key = ''; });
    chart.colors = null;
    if (!S.state) {
      setText($('#conn-badge-text'), t('state.loading'));
      $('#chart').setAttribute('aria-label', t('chart.ariaWaiting'));
      return;
    }
    renderAll();
    if (pop) renderPopoverOptions();
  }

  /* =====================================================================
   * Sidebar e connessione
   * ===================================================================== */
  function renderChrome() {
    setText($('#brand-version'), t('brand.sub', { version: S.state.version || '' }));
  }

  let bannerKey = '';
  function renderConnection() {
    const c = S.state.connection || {};
    const set = S.state.settings;
    const st = ['connected', 'reconnecting', 'disconnected', 'busy'].includes(c.state) ? c.state : 'disconnected';

    const badge = $('#conn-badge');
    badge.dataset.state = st;
    setText($('#conn-badge-text'), st === 'connected' && c.port ? t('state.connectedPort', { port: c.port }) : t('state.' + st));

    const key = JSON.stringify([LANG, st, c.port, c.attempt, c.since, c.lastError, set.reconnectInterval, set.autoReconnect]);
    if (key !== bannerKey) {
      bannerKey = key;
      $('#conn-banner').replaceChildren(...buildBanner(st, c, set));
    }
    $('#mixer-grid').classList.toggle('is-dim', st !== 'connected');

    setText($('#device-name'), c.device || t('device.none'));
    setText($('#device-port'), t('device.port', { port: c.port || '—', baud: set.baudRate }));
    setText($('#last-event'), c.lastEvent ? t('mixer.lastEvent', { text: c.lastEvent }) : '');
  }

  function buildBanner(st, c, set) {
    const err = c.lastError ? h('span', { class: 'banner-err' }, c.lastError) : null;
    if (st === 'reconnecting') {
      const time = fmtTime(c.since);
      const attempt = c.attempt > 0 ? t('banner.attempt', { n: fmtNum(c.attempt, 0) }) : '';
      const n = Number(set.reconnectInterval) || 2;
      return [h('div', { class: 'banner banner-warn' },
        icon('loader', 18, 'spin'),
        h('div', { class: 'banner-body' },
          h('span', { class: 'banner-title' }, time ? t('banner.reconnTitleAt', { time }) : t('banner.reconnTitle')),
          h('span', { class: 'banner-text' }, set.autoReconnect ? tp('banner.reconnAuto', n, { attempt }) : t('banner.reconnManual')),
          err))];
    }
    if (st === 'disconnected') {
      return [h('div', { class: 'banner banner-danger' },
        h('div', { class: 'banner-body' },
          h('span', { class: 'banner-title' }, c.port ? t('banner.offTitlePort', { port: c.port }) : t('banner.offTitle')),
          h('span', { class: 'banner-text' }, set.autoReconnect ? t('banner.offAuto') : t('banner.offManual')),
          err),
        h('a', { href: '#impostazioni', class: 'btn btn-primary', onclick: () => { focusAfterRoute = '#set-port'; } }, t('banner.choosePort')))];
    }
    if (st === 'busy') {
      return [h('div', { class: 'banner banner-busy' },
        icon('plug', 18),
        h('div', { class: 'banner-body' },
          h('span', { class: 'banner-title' }, t('banner.busyTitle')),
          h('span', { class: 'banner-text' }, c.port ? t('banner.busyText', { port: c.port }) : t('banner.busyTextNoPort')),
          err))];
    }
    return [];
  }

  /* =====================================================================
   * Mixer
   * ===================================================================== */
  let mixerKey = '';
  const mixerRefs = new Map();
  let mixerRaf = 0;

  function renderMixer() {
    const sliders = S.state.sliders;
    const act = activeSet();
    const key = JSON.stringify([LANG, sliders.map((s) => [s.index, s.name, s.targets, s.targets.map((x) => isLive(x, act))])]);
    if (key !== mixerKey) {
      mixerKey = key;
      mixerRefs.clear();
      $('#mixer-grid').replaceChildren(...sliders.map((s) => buildSliderCard(s, act)));
    }
    $('#mixer-empty').hidden = sliders.length > 0;
    updateMixerValues();
  }

  function buildSliderCard(s, act) {
    const nameId = `sl-name-${s.index}`;
    const pct = h('p', { class: 'slider-pct mono' });
    const card = h('article', { class: 'slider-card', 'aria-labelledby': nameId },
      h('div', { class: 'slider-card-head' },
        h('h2', { class: 'slider-name' + (s.name ? '' : ' is-placeholder'), id: nameId, title: sliderLabel(s) }, sliderLabel(s)),
        h('span', { class: 'slider-idx mono', 'aria-hidden': 'true' }, '#' + s.index)),
      h('div', { class: 'fader', 'aria-hidden': 'true' },
        h('div', { class: 'fader-track' }, h('div', { class: 'fader-fill' })),
        h('div', { class: 'fader-thumb' })),
      pct,
      buildMixerChips(s, act));
    mixerRefs.set(s.index, { card, pct, last: null });
    return card;
  }

  function buildMixerChips(s, act) {
    const items = s.targets.map((tg, i) => ({ info: targetInfo(tg), live: isLive(tg, act), i }));
    items.sort((a, b) => (b.live - a.live) || (a.i - b.i)); // chi suona va davanti
    const shown = items.slice(0, 3);
    const rest = items.slice(3);
    const ul = h('ul', { class: 'chips', 'aria-label': t('mixer.chipsLabel') });
    for (const it of shown) {
      ul.append(h('li', { class: `chip chip-${it.info.kind}`, title: it.info.label },
        it.live ? h('span', { class: 'dot dot-live', 'aria-hidden': 'true' }) : null,
        h('span', { class: 'chip-text' }, it.info.short),
        it.live ? h('span', { class: 'sr-only' }, ' ' + t('mixer.live')) : null));
    }
    if (rest.length) {
      const names = rest.map((r) => r.info.label);
      ul.append(h('li', { class: 'chip chip-more', title: names.join('\n') },
        h('span', { 'aria-hidden': 'true' }, '+' + fmtNum(rest.length, 0)),
        h('span', { class: 'sr-only' }, t('mixer.more', { n: rest.length, list: names.join(', ') }))));
    }
    if (!items.length) ul.append(h('li', { class: 'chip-empty' }, t('mixer.noTargets')));
    return ul;
  }

  function updateMixerValues() {
    if (!S.state) return;
    for (const s of S.state.sliders) {
      const ref = mixerRefs.get(s.index);
      if (!ref) continue;
      const v = clamp(Number(s.value) || 0, 0, 1);
      const r = Math.round(v * 1000) / 1000;
      if (ref.last === r) continue;
      ref.last = r;
      ref.card.style.setProperty('--v', String(r));
      setText(ref.pct, fmtPct(v));
    }
  }
  function scheduleMixerValues() {
    if (mixerRaf || S.view !== 'mixer') return;
    mixerRaf = requestAnimationFrame(() => { mixerRaf = 0; updateMixerValues(); });
  }

  /* =====================================================================
   * Assegnazioni
   * ===================================================================== */
  const rowRefs = new Map();
  let unKey = '';
  let mapSeq = 0;

  function renderAssign() {
    const list = $('#assign-rows');
    const sliders = S.state.sliders;
    const indices = new Set(sliders.map((s) => s.index));
    for (const [i, r] of rowRefs) {
      if (!indices.has(i)) {
        if (pop && pop.index === i) closePopover(false);
        r.row.remove();
        rowRefs.delete(i);
      }
    }
    const act = activeSet();
    sliders.forEach((s, pos) => {
      let r = rowRefs.get(s.index);
      if (!r) { r = buildRow(s.index); rowRefs.set(s.index, r); }
      if (list.children[pos] !== r.row) list.insertBefore(r.row, list.children[pos] || null);
      updateRow(r, s, act);
    });
    renderUnassigned();
    if (pop) { renderPopoverOptions(); positionPopover(); }
  }

  function buildRow(index) {
    const id = `nome-${index}`;
    const label = h('label', { for: id, class: 'sr-only' });
    const input = h('input', { id, type: 'text', class: 'input input-name', autocomplete: 'off', spellcheck: 'false', maxlength: '40', name: `slider-${index}` });
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') { e.preventDefault(); commitName(index, input.value); }
      else if (e.key === 'Escape') { const s = S.byIndex.get(index); input.value = s ? s.name : ''; input.blur(); }
    });
    input.addEventListener('blur', () => commitName(index, input.value));
    const addBtn = h('button', { type: 'button', class: 'btn-add', 'aria-haspopup': 'dialog', 'aria-expanded': 'false', 'data-focus-key': `add:${index}` },
      icon('plus', 12, '', 2.5), h('span', null));
    addBtn.addEventListener('click', () => togglePopover(index));
    const targets = h('div', { class: 'targets' }, addBtn);
    const row = h('li', { class: 'assign-row', 'data-index': index },
      h('div', { class: 'assign-name' },
        h('span', { class: 'idx-badge mono', 'aria-hidden': 'true' }, String(index)),
        label, input),
      targets);
    return { row, input, label, targets, addBtn, chipsKey: '' };
  }

  function updateRow(r, s, act) {
    const name = sliderLabel(s);
    setText(r.label, t('assign.nameLabel', { n: s.index }));
    r.input.placeholder = t('slider.placeholder', { n: s.index });
    if (document.activeElement !== r.input) r.input.value = s.name;
    setText(r.addBtn.lastChild, t('assign.add'));
    r.addBtn.setAttribute('aria-label', t('assign.addTo', { name }));
    const key = JSON.stringify([LANG, name, s.targets, s.targets.map((x) => isLive(x, act))]);
    if (key === r.chipsKey) return;
    r.chipsKey = key;
    withFocus(r.targets, () => {
      while (r.targets.firstChild !== r.addBtn) r.targets.firstChild.remove();
      for (const tg of s.targets) r.targets.insertBefore(buildTargetChip(s, tg, act), r.addBtn);
    });
  }

  function buildTargetChip(s, tg, act) {
    const info = targetInfo(tg);
    const live = isLive(tg, act);
    return h('span', { class: `tchip tchip-${info.kind}` },
      live ? h('span', { class: 'dot dot-live', 'aria-hidden': 'true' }) : null,
      h('span', { class: 'tchip-text', title: info.label }, info.label),
      live ? h('span', { class: 'sr-only' }, ' ' + t('mixer.live')) : null,
      h('button', {
        type: 'button', class: 'tchip-x',
        'aria-label': t('assign.remove', { target: info.label, name: sliderLabel(s) }),
        'data-focus-key': `x:${s.index}:${lc(tg)}`,
        onclick: () => removeTarget(s.index, tg)
      }, icon('x', 12, '', 2.5)));
  }

  function commitName(index, raw) {
    const s = S.byIndex.get(index);
    if (!s) return;
    const name = String(raw).trim();
    if (name === s.name) return;
    saveMapping(() => { s.name = name; });
  }
  function addTarget(index, key, focusFn) {
    const s = S.byIndex.get(index);
    const k = String(key).trim();
    if (!s || !k || s.targets.some((x) => lc(x) === lc(k))) { if (focusFn) focusFn(); return; }
    saveMapping(() => { s.targets.push(k); }, focusFn);
  }
  function removeTarget(index, tg) {
    const s = S.byIndex.get(index);
    if (!s) return;
    const i = s.targets.findIndex((x) => lc(x) === lc(tg));
    if (i < 0) return;
    const next = s.targets[i + 1] != null ? s.targets[i + 1] : s.targets[i - 1];
    saveMapping(() => { s.targets.splice(i, 1); }, () => {
      const r = rowRefs.get(index);
      if (!r) return;
      const el = next != null ? r.targets.querySelector(`[data-focus-key="${CSS.escape(`x:${index}:${lc(next)}`)}"]`) : null;
      (el || r.addBtn).focus();
    });
  }
  function addSlider() {
    const sliders = S.state.sliders;
    const index = sliders.length ? Math.max(...sliders.map((s) => s.index)) + 1 : 0;
    saveMapping(() => {
      const s = { index, name: '', targets: [], raw: 0, value: 0 };
      sliders.push(s);
      S.byIndex.set(index, s);
    }, () => { const r = rowRefs.get(index); if (r) r.input.focus(); });
  }

  async function saveMapping(mutate, focusFn) {
    const before = clone(S.state.sliders);
    mutate();
    S.byIndex = new Map(S.state.sliders.map((s) => [s.index, s]));
    renderMixer(); renderAssign(); renderChartSelect();
    if (focusFn) focusFn();
    setSave('assign', 'saving');
    const seq = ++mapSeq;
    const body = { sliders: S.state.sliders.map((s) => ({ index: s.index, name: s.name, targets: s.targets.slice() })) };
    try {
      const st = await api.post('/api/mapping', body);
      if (seq !== mapSeq) return;
      setSave('assign', 'ok');
      if (st) applyState(st);
      loadSessions();
    } catch (e) {
      if (seq !== mapSeq) return;
      setSave('assign', 'error', e.message);
      S.state.sliders = before;
      S.byIndex = new Map(before.map((s) => [s.index, s]));
      renderAll();
    }
  }

  function renderUnassigned() {
    const items = S.sessions
      .filter((x) => !x.mapped && !SESSION_SPECIAL.has(lc(x.key)))
      .sort((a, b) => (b.active - a.active) || lc(a.key).localeCompare(lc(b.key)));
    const key = JSON.stringify([LANG, items.map((x) => [x.key, x.active]), S.state.sliders.map((s) => [s.index, sliderLabel(s)])]);
    if (key === unKey) return;
    unKey = key;
    const box = $('#unassigned-list');
    withFocus(box, () => {
      if (!items.length) {
        box.replaceChildren(h('p', { class: 'muted small' }, t('assign.noneUnassigned')));
        return;
      }
      box.replaceChildren(...items.map((x, i) => {
        const id = `un-${i}`;
        const title = prettyName(x.key);
        const sel = h('select', { id, class: 'select select-sm', 'data-focus-key': `un:${lc(x.key)}` },
          h('option', { value: '' }, t('assign.assignTo')),
          S.state.sliders.map((s) => h('option', { value: String(s.index) }, t('assign.option', { n: s.index, name: sliderLabel(s) }))));
        sel.addEventListener('change', () => {
          if (sel.value === '') return;
          const nextSel = sel.closest('.un-item').nextElementSibling;
          const nextKey = nextSel ? nextSel.querySelector('select').dataset.focusKey : null;
          x.mapped = true;
          addTarget(Number(sel.value), x.key, () => {
            const n = nextKey ? box.querySelector(`[data-focus-key="${CSS.escape(nextKey)}"]`) : null;
            (n || $('#h-unassigned')).focus();
          });
        });
        return h('div', { class: 'un-item' },
          h('span', { class: x.active ? 'dot dot-live' : 'dot dot-idle', 'aria-hidden': 'true' }),
          h('div', { class: 'un-text' },
            h('span', { class: 'un-title' }, title, h('span', { class: 'sr-only' }, ` (${x.active ? t('assign.playing') : t('assign.idle')})`)),
            h('span', { class: 'un-key' }, x.key)),
          h('label', { for: id, class: 'sr-only' }, t('assign.assignLabel', { app: title })),
          sel);
      }));
    });
  }

  /* ---------- popover "Aggiungi" ---------- */
  let pop = null;

  function togglePopover(index) {
    if (pop && pop.index === index) closePopover(true);
    else openPopover(index);
  }

  function openPopover(index) {
    if (pop) closePopover(false);
    const r = rowRefs.get(index);
    const s = S.byIndex.get(index);
    if (!r || !s) return;
    const input = h('input', {
      id: 'pop-search', type: 'text', class: 'pop-input', role: 'combobox',
      'aria-expanded': 'true', 'aria-controls': 'pop-list', 'aria-autocomplete': 'list',
      name: 'search-app', placeholder: t('pop.placeholder'), autocomplete: 'off', spellcheck: 'false'
    });
    const list = h('div', { id: 'pop-list', role: 'listbox', class: 'pop-list', 'aria-label': t('pop.list') });
    const el = h('div', { class: 'popover', role: 'dialog', 'aria-label': t('pop.dialog', { name: sliderLabel(s) }) },
      h('div', { class: 'pop-search' }, icon('search', 15), h('label', { for: 'pop-search', class: 'sr-only' }, t('pop.search')), input),
      list);
    r.row.append(el);
    pop = { index, el, input, list, r, activeKey: null, opts: [] };
    r.addBtn.setAttribute('aria-expanded', 'true');
    input.addEventListener('input', () => { pop.activeKey = null; renderPopoverOptions(); });
    input.addEventListener('keydown', onPopKey);
    el.addEventListener('focusout', (e) => {
      if (pop && e.relatedTarget && !el.contains(e.relatedTarget) && e.relatedTarget !== r.addBtn) closePopover(false);
    });
    document.addEventListener('pointerdown', onDocPointer, true);
    renderPopoverOptions();
    positionPopover();
    input.focus({ preventScroll: true });
    el.scrollIntoView({ block: 'nearest', behavior: REDUCED.matches ? 'auto' : 'smooth' });
  }

  function positionPopover() {
    if (!pop) return;
    const b = pop.r.addBtn;
    pop.el.style.top = `${b.offsetTop + b.offsetHeight + 6}px`;
  }

  function closePopover(refocus) {
    if (!pop) return;
    const p = pop;
    pop = null;
    document.removeEventListener('pointerdown', onDocPointer, true);
    p.el.remove();
    p.r.addBtn.setAttribute('aria-expanded', 'false');
    if (refocus) p.r.addBtn.focus();
  }

  function onDocPointer(e) {
    if (!pop) return;
    if (pop.el.contains(e.target) || pop.r.addBtn.contains(e.target)) return;
    closePopover(false);
  }

  function renderPopoverOptions() {
    const s = S.byIndex.get(pop.index);
    if (!s) { closePopover(false); return; }
    const q = pop.input.value.trim();
    const ql = lc(q);
    const have = new Set(s.targets.map(lc));
    const sess = S.sessions.filter((x) => !SESSION_SPECIAL.has(lc(x.key)));
    const groups = [
      { label: t('pop.playing'), items: sess.filter((x) => x.active).map((x) => ({ key: x.key, title: prettyName(x.key), live: true })) },
      { label: t('pop.open'), items: sess.filter((x) => !x.active).map((x) => ({ key: x.key, title: prettyName(x.key) })) },
      { label: t('pop.specials'), items: SPECIAL_MENU.map((k) => ({ key: k, title: t('special.' + k) })) }
    ];
    const exact = have.has(ql) || groups.some((g) => g.items.some((it) => lc(it.key) === ql));
    for (const g of groups) {
      g.items = g.items.filter((it) => !have.has(lc(it.key)) && (!ql || lc(it.key).includes(ql) || lc(it.title).includes(ql)));
    }
    if (q && !exact) groups.push({ label: null, aria: t('pop.manual'), items: [{ key: q, title: t('pop.custom', { text: q }), custom: true }] });

    const opts = [];
    const frag = [];
    groups.filter((g) => g.items.length).forEach((g, gi) => {
      if (gi > 0) frag.push(h('div', { class: 'pop-sep', role: 'presentation' }));
      const gid = `pop-g-${gi}`;
      const grp = h('div', { role: 'group', 'aria-labelledby': g.label ? gid : null, 'aria-label': g.label ? null : g.aria },
        g.label ? h('div', { id: gid, class: 'pop-group-label', role: 'presentation' }, g.label) : null);
      for (const it of g.items) {
        const i = opts.length;
        const o = h('div', { id: `pop-o-${i}`, role: 'option', class: 'pop-option', 'aria-selected': 'false' },
          it.live ? h('span', { class: 'dot dot-live', 'aria-hidden': 'true' }) : null,
          it.custom ? icon('plus', 14) : null,
          h('span', { class: 'pop-title' }, it.title),
          it.custom ? null : h('span', { class: 'pop-key mono' }, it.key));
        o.addEventListener('pointerdown', (e) => e.preventDefault());
        o.addEventListener('click', () => choose(it));
        o.addEventListener('pointermove', () => { if (pop && pop.activeIndex !== i) setActive(i); });
        opts.push({ el: o, it });
        grp.append(o);
      }
      frag.push(grp);
    });
    if (!opts.length) frag.push(h('p', { class: 'pop-empty' }, q ? t('pop.emptyAssigned') : t('pop.empty')));
    pop.list.replaceChildren(...frag);
    pop.opts = opts;
    const keep = pop.activeKey != null ? opts.findIndex((o) => o.it.key === pop.activeKey) : -1;
    setActive(keep >= 0 ? keep : 0);
  }

  function setActive(i) {
    if (!pop) return;
    pop.opts.forEach((o, j) => {
      o.el.classList.toggle('is-active', j === i);
      o.el.setAttribute('aria-selected', j === i ? 'true' : 'false');
    });
    const o = pop.opts[i];
    pop.activeIndex = o ? i : -1;
    pop.activeKey = o ? o.it.key : null;
    if (o) {
      pop.input.setAttribute('aria-activedescendant', o.el.id);
      o.el.scrollIntoView({ block: 'nearest' });
    } else pop.input.removeAttribute('aria-activedescendant');
  }

  function onPopKey(e) {
    if (!pop) return;
    const n = pop.opts.length;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (!n) return;
      const d = e.key === 'ArrowDown' ? 1 : -1;
      setActive((pop.activeIndex + d + n) % n);
    } else if (e.key === 'PageDown' || e.key === 'PageUp') {
      e.preventDefault();
      if (n) setActive(e.key === 'PageDown' ? n - 1 : 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const o = pop.opts[pop.activeIndex];
      if (o) choose(o.it);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      closePopover(true);
    } else if (e.key === 'Tab') {
      closePopover(false);
    }
  }

  function choose(it) {
    if (!pop) return;
    const index = pop.index;
    closePopover(true);
    addTarget(index, it.key, () => { const r = rowRefs.get(index); if (r) r.addBtn.focus(); });
  }

  /* =====================================================================
   * Impostazioni
   * ===================================================================== */
  let pendingSettings = null;
  let settingsTimer = 0;
  let settingsSeq = 0;

  function renderSettings() {
    const set = S.state.settings;
    const active = document.activeElement;

    $$('[role="switch"][data-setting]').forEach((sw) => sw.setAttribute('aria-checked', String(!!set[sw.dataset.setting])));
    renderPortOptions();

    const baud = $('#set-baud');
    const bauds = BAUDS.includes(Number(set.baudRate)) ? BAUDS : BAUDS.concat(Number(set.baudRate)).sort((a, b) => a - b);
    fillSelect(baud, bauds.map((b) => [String(b), t('settings.baudOption', { n: b })]));
    if (active !== baud) baud.value = String(set.baudRate);

    const langSel = $('#set-language');
    fillSelect(langSel, [['auto', t('lang.auto')]].concat(Object.keys(I18N).map((k) => [k, I18N[k]['meta.name'] || k])));
    if (active !== langSel) langSel.value = I18N[set.language] ? set.language : 'auto';

    const iv = $('#set-interval');
    if (active !== iv) { iv.value = String(set.reconnectInterval); iv.removeAttribute('aria-invalid'); }
    iv.disabled = !set.autoReconnect;

    setText($('#config-path'), set.configPath || '');

    renderPresetRadios();
    const f = effectiveFilter(set);
    $$('.range[data-filter]').forEach((r) => {
      if (active !== r) r.value = String(f[r.dataset.filter]);
      updateRangeUI(r);
    });
  }

  function fillSelect(sel, opts) {
    const key = JSON.stringify(opts);
    if (sel.dataset.key === key) return;
    if (document.activeElement === sel && sel.dataset.key) return; // non toccare un select aperto
    const v = sel.value;
    sel.replaceChildren(...opts.map(([val, label]) => h('option', { value: val }, label)));
    sel.dataset.key = key;
    if (v && opts.some((o) => o[0] === v)) sel.value = v;
  }

  function renderPortOptions() {
    const sel = $('#set-port');
    const cur = S.state.settings.comPort || 'auto';
    const ard = S.ports.find((p) => p.isArduino);
    const opts = [['auto', ard ? t('settings.portAutoFound', { desc: ard.description || 'Arduino', port: ard.name }) : t('settings.portAuto')]];
    S.ports.forEach((p) => opts.push([p.name, p.description ? `${p.name} · ${p.description}` : p.name]));
    let curVal = cur;
    if (cur !== 'auto') {
      const m = S.ports.find((p) => lc(p.name) === lc(cur));
      if (m) curVal = m.name;
      else opts.push([cur, t('settings.portMissing', { port: cur })]);
    }
    fillSelect(sel, opts);
    if (document.activeElement !== sel) sel.value = curVal;
  }

  function renderPresetRadios() {
    const cur = PRESETS[S.state.settings.noiseReduction] ? S.state.settings.noiseReduction : 'custom';
    $$('#preset-group [role="radio"]').forEach((b) => {
      const on = b.dataset.preset === cur;
      b.setAttribute('aria-checked', on ? 'true' : 'false');
      b.tabIndex = on ? 0 : -1;
    });
  }

  function rangeText(kind, v) {
    if (kind === 'deadband') return fmtPct(v / 100, 1, 1);
    if (kind === 'edgeSnap') return v === 0 ? t('filter.edgeOff') : fmtPct(v / 100, 1, 0);
    if (v === 0) return t('filter.smoothOff');
    const word = v <= 0.3 ? t('filter.light') : v <= 0.6 ? t('filter.medium') : t('filter.strong');
    return `${fmtNum(v, 1, 1)} · ${word}`;
  }
  function updateRangeUI(r) {
    const v = Number(r.value);
    const min = Number(r.min), max = Number(r.max);
    r.style.setProperty('--fill', `${((v - min) / (max - min)) * 100}%`);
    const txt = rangeText(r.dataset.filter, v);
    setText($(`#v-${r.dataset.filter}`), txt);
    r.setAttribute('aria-valuetext', txt);
  }
  function currentRangeFilter() {
    const f = {};
    $$('.range[data-filter]').forEach((r) => { f[r.dataset.filter] = Number(r.value); });
    return f;
  }

  function queueSettings(patch) {
    mergeSettings(S.state.settings, patch);
    pendingSettings = Object.assign(pendingSettings || {}, patch);
    clearTimeout(settingsTimer);
    settingsTimer = setTimeout(flushSettings, 400);
  }
  function saveSettingsNow(patch) {
    mergeSettings(S.state.settings, patch);
    pendingSettings = Object.assign(pendingSettings || {}, patch);
    flushSettings();
  }
  async function flushSettings() {
    clearTimeout(settingsTimer);
    settingsTimer = 0;
    const patch = pendingSettings;
    pendingSettings = null;
    if (!patch) return;
    const seq = ++settingsSeq;
    setSave('settings', 'saving');
    try {
      const st = await api.post('/api/settings', patch);
      if (seq !== settingsSeq) return;
      setSave('settings', 'ok');
      if (st) applyState(st);
    } catch (e) {
      if (seq !== settingsSeq) return;
      setSave('settings', 'error', e.message);
      try { applyState(await api.get('/api/state')); } catch (e2) { /* resta l'ultimo stato */ }
    }
  }

  function setSave(which, kind, msg) {
    const el = $(`#save-${which}`);
    el.dataset.kind = kind;
    if (kind === 'saving') el.replaceChildren(icon('loader', 14, 'spin'), t('save.saving'));
    else if (kind === 'ok') el.replaceChildren(icon('check', 14, '', 2.5), t('save.ok'));
    else el.replaceChildren(icon('alert', 14), t('save.error', { error: msg || '' }));
  }

  function bindSettings() {
    $$('[role="switch"][data-setting]').forEach((sw) => {
      sw.addEventListener('click', () => {
        if (!S.state) return;
        const on = sw.getAttribute('aria-checked') !== 'true';
        sw.setAttribute('aria-checked', String(on));
        saveSettingsNow({ [sw.dataset.setting]: on });
        if (sw.dataset.setting === 'autoReconnect') { $('#set-interval').disabled = !on; renderConnection(); }
      });
    });

    $('#set-port').addEventListener('change', (e) => { if (S.state) saveSettingsNow({ comPort: e.target.value }); });
    $('#set-baud').addEventListener('change', (e) => { if (S.state) saveSettingsNow({ baudRate: Number(e.target.value) }); });
    $('#set-language').addEventListener('change', (e) => {
      if (!S.state) return;
      const v = e.target.value;
      saveSettingsNow({ language: v });
      setLanguage(resolveLang(v));
    });

    const iv = $('#set-interval');
    const validInterval = () => {
      const n = Number(iv.value);
      return Number.isInteger(n) && n >= 1 && n <= 30 ? n : null;
    };
    iv.addEventListener('input', () => {
      if (!S.state) return;
      const n = validInterval();
      if (n == null) { iv.setAttribute('aria-invalid', 'true'); return; }
      iv.removeAttribute('aria-invalid');
      if (n !== S.state.settings.reconnectInterval || pendingSettings) queueSettings({ reconnectInterval: n });
    });
    iv.addEventListener('blur', () => {
      if (!S.state) return;
      if (validInterval() == null) {
        setSave('settings', 'error', t('error.interval'));
        iv.value = String(S.state.settings.reconnectInterval);
        iv.removeAttribute('aria-invalid');
      } else if (pendingSettings) flushSettings();
    });
    iv.addEventListener('keydown', (e) => { if (e.key === 'Enter' && validInterval() != null) flushSettings(); });

    const group = $('#preset-group');
    const radios = $$('[role="radio"]', group);
    const pick = (key) => {
      if (!S.state) return;
      if (key === 'custom') {
        const f = currentRangeFilter();
        saveSettingsNow({ noiseReduction: 'custom', filter: f });
      } else {
        if (pendingSettings) delete pendingSettings.filter;
        saveSettingsNow({ noiseReduction: key });
      }
      renderSettings();
    };
    radios.forEach((b) => b.addEventListener('click', () => pick(b.dataset.preset)));
    group.addEventListener('keydown', (e) => {
      const keys = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 };
      if (!(e.key in keys)) return;
      e.preventDefault();
      const i = radios.indexOf(document.activeElement);
      const next = radios[(i + keys[e.key] + radios.length) % radios.length];
      next.focus();
      pick(next.dataset.preset);
    });

    $$('.range[data-filter]').forEach((r) => {
      r.addEventListener('input', () => {
        if (!S.state) return;
        updateRangeUI(r);
        queueSettings({ noiseReduction: 'custom', filter: currentRangeFilter() });
        renderPresetRadios();
      });
      r.addEventListener('change', () => { if (pendingSettings) flushSettings(); });
    });

    $$('[data-open]').forEach((b) => {
      b.addEventListener('click', async () => {
        try { await api.post('/api/open', { what: b.dataset.open }); } catch (e) { toast(t('toast.openFail', { error: e.message }), true); }
      });
    });

    $('#chart-slider').addEventListener('change', (e) => {
      chart.index = Number(e.target.value);
      chart.init = false;
      chart.lastAria = 0;
    });
  }

  /* =====================================================================
   * Grafico "Anteprima dal vivo"
   * ===================================================================== */
  const chart = { index: 0, raf: 0, lo: 0, hi: 1, init: false, colors: null, lastAria: 0 };

  function renderChartSelect() {
    const sel = $('#chart-slider');
    const sliders = S.state.sliders;
    fillSelect(sel, sliders.map((s) => [String(s.index), t('assign.option', { n: s.index, name: sliderLabel(s) })]));
    sel.disabled = !sliders.length;
    if (!S.byIndex.has(chart.index) && sliders.length) { chart.index = sliders[0].index; chart.init = false; }
    if (document.activeElement !== sel) sel.value = String(chart.index);
  }

  function chartColors() {
    const cs = getComputedStyle(document.documentElement);
    const v = (n) => cs.getPropertyValue(n).trim();
    return { raw: v('--chart-raw'), value: v('--primary'), grid: v('--chart-grid'), text: v('--muted-foreground'), font: v('--font-mono') };
  }

  function startChart() {
    if (chart.raf || S.view !== 'impostazioni' || document.hidden) return;
    const loop = () => {
      chart.raf = 0;
      if (S.view !== 'impostazioni' || document.hidden) return;
      drawChart();
      chart.raf = requestAnimationFrame(loop);
    };
    chart.raf = requestAnimationFrame(loop);
  }

  function drawChart() {
    const cv = $('#chart');
    const w = cv.clientWidth, hh = cv.clientHeight;
    if (!w || !hh) return;
    const dpr = window.devicePixelRatio || 1;
    if (cv.width !== Math.round(w * dpr) || cv.height !== Math.round(hh * dpr)) {
      cv.width = Math.round(w * dpr);
      cv.height = Math.round(hh * dpr);
    }
    const ctx = cv.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, hh);
    const col = chart.colors || (chart.colors = chartColors());

    const now = performance.now();
    const t0 = now - HISTORY_MS;
    const arr = hist.get(chart.index) || [];
    let start = 0;
    while (start < arr.length - 1 && arr[start + 1].t <= t0) start++;
    const pts = arr.slice(start);

    const L = 54, R = 8, T = 8, B = 20;
    const pw = w - L - R, ph = hh - T - B;

    if (pts.length) {
      let mn = Infinity, mx = -Infinity;
      for (const p of pts) { mn = Math.min(mn, p.raw, p.value); mx = Math.max(mx, p.raw, p.value); }
      const range = Math.max(0.04, (mx - mn) * 1.3);
      const mid = (mn + mx) / 2;
      let lo = mid - range / 2, hi = mid + range / 2;
      if (range >= 1) { lo = 0; hi = 1; } else if (lo < 0) { hi -= lo; lo = 0; } else if (hi > 1) { lo -= hi - 1; hi = 1; }
      if (!chart.init || REDUCED.matches) { chart.lo = lo; chart.hi = hi; chart.init = true; }
      else { chart.lo += (lo - chart.lo) * 0.12; chart.hi += (hi - chart.hi) * 0.12; }
    }
    const lo = chart.lo, hi = chart.hi;

    ctx.font = `11px ${col.font}`;
    ctx.lineWidth = 1;
    ctx.strokeStyle = col.grid;
    ctx.fillStyle = col.text;
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';
    for (let i = 0; i <= 2; i++) {
      const y = Math.round(T + (ph * i) / 2) + 0.5;
      ctx.beginPath(); ctx.moveTo(L, y); ctx.lineTo(w - R, y); ctx.stroke();
      if (pts.length) ctx.fillText(fmtPct(hi - ((hi - lo) * i) / 2, 1, 1), L - 8, y);
    }
    ctx.textBaseline = 'alphabetic';
    ctx.textAlign = 'left'; ctx.fillText(t('chart.ago'), L, hh - 4);
    ctx.textAlign = 'right'; ctx.fillText(t('chart.now'), w - R, hh - 4);

    if (!pts.length) {
      ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
      ctx.font = `12px ${getComputedStyle(document.body).fontFamily}`;
      ctx.fillText(t('chart.waiting'), L + pw / 2, T + ph / 2);
      updateChartAria(null, now);
      return;
    }

    const X = (tt) => L + ((Math.max(tt, t0) - t0) / HISTORY_MS) * pw;
    const Y = (v) => T + ((hi - v) / (hi - lo)) * ph;
    const last = pts[pts.length - 1];
    ctx.save();
    ctx.beginPath(); ctx.rect(L, T - 2, pw, ph + 4); ctx.clip();
    ctx.lineJoin = 'round';

    ctx.beginPath();
    pts.forEach((p, i) => { if (i) ctx.lineTo(X(p.t), Y(p.raw)); else ctx.moveTo(X(p.t), Y(p.raw)); });
    ctx.lineTo(X(now), Y(last.raw));
    ctx.strokeStyle = col.raw; ctx.lineWidth = 1.25; ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(X(pts[0].t), Y(pts[0].value));
    for (let i = 1; i < pts.length; i++) {
      const x = X(pts[i].t);
      ctx.lineTo(x, Y(pts[i - 1].value));
      ctx.lineTo(x, Y(pts[i].value));
    }
    ctx.lineTo(X(now), Y(last.value));
    ctx.strokeStyle = col.value; ctx.lineWidth = 2; ctx.stroke();
    ctx.restore();

    updateChartAria(last, now);
  }

  function updateChartAria(last, now) {
    if (now - chart.lastAria < 1000) return;
    chart.lastAria = now;
    const s = S.byIndex.get(chart.index);
    $('#chart').setAttribute('aria-label', last && s
      ? t('chart.aria', { name: sliderLabel(s), raw: fmtPct(last.raw, 1, 1), value: fmtPct(last.value, 1, 1) })
      : t('chart.ariaWaiting'));
  }

  /* =====================================================================
   * Caricamenti, SSE, polling
   * ===================================================================== */
  async function loadState() {
    let delay = 1000;
    for (;;) {
      try {
        applyState(await api.get('/api/state'));
        setLinkDown(false);
        return;
      } catch (e) {
        setLinkDown(true);
        await sleep(delay);
        delay = Math.min(delay * 2, 15000);
      }
    }
  }

  async function loadSessions() {
    try {
      const r = await api.get('/api/sessions');
      S.sessions = (r && r.sessions) || [];
      if (S.state) { renderMixer(); renderAssign(); }
    } catch (e) { /* riproverà al prossimo giro */ }
  }

  async function loadPorts() {
    try {
      const r = await api.get('/api/ports');
      S.ports = (r && r.ports) || [];
      if (S.state) renderPortOptions();
    } catch (e) { /* lascia la lista precedente */ }
  }

  let sessTimer = 0, sessGen = 0;
  function startSessionPolling() {
    const gen = ++sessGen;
    clearTimeout(sessTimer);
    if (S.view === 'impostazioni' || document.hidden || !S.state) return;
    const tick = async () => {
      if (gen !== sessGen) return;
      await loadSessions();
      if (gen === sessGen) sessTimer = setTimeout(tick, 3000);
    };
    tick();
  }

  let es = null, sseRetry = 0, sseTimer = 0, sseGrace = 0;
  function connectSSE() {
    clearTimeout(sseTimer);
    if (es) es.close();
    es = api.events();
    es.addEventListener('open', () => {
      sseRetry = 0;
      clearTimeout(sseGrace); sseGrace = 0;
      if (S.linkDown) {
        setLinkDown(false);
        api.get('/api/state').then(applyState).catch(() => {});
        loadSessions();
      }
    });
    es.addEventListener('error', () => {
      if (es.readyState === 2) { // CLOSED: EventSource non riprova da solo
        const delay = Math.min(15000, 1000 * Math.pow(2, sseRetry++));
        sseTimer = setTimeout(connectSSE, delay);
      }
      if (!sseGrace && !S.linkDown) sseGrace = setTimeout(() => { sseGrace = 0; if (es.readyState !== 1) setLinkDown(true); }, 1500);
    });
    es.addEventListener('sliders', (e) => { try { onSliders(JSON.parse(e.data)); } catch (err) { /* evento malformato */ } });
    es.addEventListener('connection', (e) => { try { onConnection(JSON.parse(e.data)); } catch (err) { /* evento malformato */ } });
    es.addEventListener('state', (e) => { try { applyState(JSON.parse(e.data)); } catch (err) { /* evento malformato */ } });
  }

  function onSliders(data) {
    if (!S.state || !data) return;
    const now = performance.now();
    for (const u of data.sliders || []) {
      const s = S.byIndex.get(u.index);
      if (!s) continue;
      if (u.raw != null) s.raw = u.raw;
      if (u.value != null) s.value = u.value;
      pushSample(u.index, Number(s.raw), Number(s.value), now);
    }
    scheduleMixerValues();
  }

  function onConnection(c) {
    if (!S.state || !c) return;
    const portChanged = S.state.connection.port !== c.port || S.state.connection.state !== c.state;
    S.state.connection = c;
    renderConnection();
    if (portChanged) loadPorts();
  }

  function setLinkDown(down) {
    S.linkDown = down;
    $('#link-banner').hidden = !down;
  }

  let toastTimer = 0;
  function toast(msg, isErr) {
    const el = $('#toast');
    el.dataset.kind = isErr ? 'error' : '';
    el.textContent = msg;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { el.textContent = ''; }, 5000);
  }

  /* =====================================================================
   * Navigazione (hash)
   * ===================================================================== */
  let focusAfterRoute = null;
  function route(fromUser) {
    const v = location.hash.slice(1);
    const view = VIEWS[v] ? v : 'mixer';
    S.view = view;
    $$('.view').forEach((sec) => { sec.hidden = sec.dataset.view !== view; });
    $$('[data-nav]').forEach((a) => {
      if (a.dataset.nav === view) a.setAttribute('aria-current', 'page');
      else a.removeAttribute('aria-current');
    });
    document.title = t('page.title', { view: t(VIEWS[view]) });
    if (pop) closePopover(false);
    if (view === 'mixer') updateMixerValues();
    if (view === 'impostazioni') { loadPorts(); chart.lastAria = 0; startChart(); }
    startSessionPolling();
    if (fromUser) {
      const target = (focusAfterRoute && $(focusAfterRoute)) || $(`#h-${view}`);
      focusAfterRoute = null;
      window.scrollTo(0, 0);
      if (target) target.focus({ preventScroll: true });
    }
  }

  /* =====================================================================
   * Avvio
   * ===================================================================== */
  async function init() {
    setLanguage(resolveLang(MOCK && qs.get('lang') ? qs.get('lang') : 'auto'));
    bindSettings();
    $('#btn-add-slider').addEventListener('click', () => { if (S.state) addSlider(); });
    const rb = $('#btn-reconnect');
    rb.addEventListener('click', async () => {
      if (rb.getAttribute('aria-disabled') === 'true') return;
      rb.setAttribute('aria-disabled', 'true');
      try { await api.post('/api/reconnect'); } catch (e) { toast(t('toast.reconnectFail', { error: e.message }), true); }
      setTimeout(() => rb.removeAttribute('aria-disabled'), 800);
    });
    window.addEventListener('hashchange', () => route(true));
    $('#skip-link').addEventListener('click', (e) => { e.preventDefault(); $('#contenuto').focus(); });
    document.addEventListener('visibilitychange', () => { startSessionPolling(); startChart(); });
    DARK.addEventListener('change', () => { chart.colors = null; });
    window.addEventListener('resize', () => { if (pop) positionPopover(); });
    route(false);

    await loadState();
    connectSSE();
    startSessionPolling();
    loadPorts();
  }

  /* =====================================================================
   * Mock (?mock=1): dati finti coerenti con l'API, tutto in memoria.
   *   &cycle=1  lo stato di connessione cicla ogni 6 s
   *   &fail=1   i salvataggi rispondono con errore
   *   &drop=1   lo stream SSE cade dopo 5 s e torna dopo altri 6
   *   &lang=xx  forza settings.language
   * ===================================================================== */
  function createMock() {
    const cycle = qs.get('cycle') === '1';
    const fail = qs.get('fail') === '1';
    const drop = qs.get('drop') === '1';
    const langParam = qs.get('lang');
    const hhmm = (ms) => new Date(ms).toLocaleTimeString(langParam === 'en' ? 'en-GB' : 'it-IT', { hour: '2-digit', minute: '2-digit' });
    const iso = (ms) => new Date(ms).toISOString();

    const names = langParam === 'en' ? ['Master', 'Browser', 'Music', 'Discord', 'Games', 'Mic'] : ['Master', 'Browser', 'Musica', 'Discord', 'Giochi', 'Microfono'];
    const targets = [
      ['master'],
      ['chrome.exe', 'msedge.exe', 'firefox.exe'],
      ['spotify.exe', 'vlc.exe'],
      ['discord.exe'],
      ['deej.current', 'cs2.exe', 'RocketLeague.exe', 'eldenring.exe', 'javaw.exe', 'Minecraft.Windows.exe'],
      ['mic']
    ];
    const start = [0.72, 0.45, 0.60, 0.80, 1.0, 0.90];
    const t0 = Date.now() - 6 * 60000;
    const st = {
      version: 'v1.0.0',
      connection: { state: 'connected', port: 'COM3', device: 'Arduino Uno', attempt: 0, since: iso(t0), lastError: '', lastEvent: `${langParam === 'en' ? 'Reconnected at' : 'Ricollegato alle'} ${hhmm(t0)} · ${langParam === 'en' ? '6 sliders detected' : '6 slider rilevati'}` },
      sliders: names.map((name, i) => ({ index: i, name, raw: start[i], value: start[i], targets: targets[i].slice() })),
      settings: {
        comPort: 'auto', baudRate: 9600, invertSliders: false,
        autoReconnect: true, reconnectInterval: 2, notifyOnDisconnect: true,
        noiseReduction: 'default', filter: { deadband: 0.4, smoothing: 0, edgeSnap: 1 },
        sliderMaxValue: 1023, startWithWindows: true, openUiOnStart: false,
        language: langParam || 'auto',
        configPath: 'C:\\Users\\you\\AppData\\Local\\Programs\\deej-plus\\config.yaml'
      }
    };
    const phys = new Map(st.sliders.map((s) => [s.index, { pos: s.raw, goal: s.raw, ema: s.raw }]));
    const sessions = [
      { key: 'chrome.exe', active: true }, { key: 'spotify.exe', active: true }, { key: 'discord.exe', active: true },
      { key: 'steam.exe', active: true }, { key: 'obs64.exe', active: false }, { key: 'zoom.exe', active: false },
      { key: 'firefox.exe', active: false }, { key: 'master', active: true }, { key: 'system', active: false }, { key: 'mic', active: true }
    ];
    const sources = new Set();
    const emit = (type, data) => sources.forEach((s) => s._emit(type, data));
    const isMapped = (key) => {
      const k = lc(key);
      return SESSION_SPECIAL.has(k) || st.sliders.some((s) => s.targets.some((x) => lc(x) === k));
    };

    let nextMove = performance.now() + 2500;
    setInterval(() => {
      if (st.connection.state !== 'connected') return;
      const now = performance.now();
      if (now > nextMove) {
        const i = [1, 2, 3][Math.floor(Math.random() * 3)];
        const p = phys.get(i);
        if (p) p.goal = 0.15 + Math.random() * 0.8;
        nextMove = now + 5000 + Math.random() * 4000;
      }
      const f = effectiveFilter(st.settings);
      const out = [];
      for (const s of st.sliders) {
        let p = phys.get(s.index);
        if (!p) { p = { pos: 0.5, goal: 0.5, ema: 0.5 }; phys.set(s.index, p); }
        const d = p.goal - p.pos;
        p.pos += Math.sign(d) * Math.min(Math.abs(d), 0.02);
        let raw = clamp(p.pos + (Math.random() * 2 - 1) * 0.001, 0, 1);
        if (st.settings.invertSliders) raw = 1 - raw;
        raw = Math.round(raw * 1023) / 1023;
        p.ema = f.smoothing > 0 ? p.ema * f.smoothing + raw * (1 - f.smoothing) : raw;
        const e = f.edgeSnap / 100;
        let v = s.value;
        if (p.ema <= e) v = 0;
        else if (p.ema >= 1 - e) v = 1;
        else if (Math.abs(p.ema - v) >= f.deadband / 100) v = Math.round(p.ema * 100) / 100;
        s.raw = raw; s.value = v;
        out.push({ index: s.index, raw: Number(raw.toFixed(4)), value: v });
      }
      emit('sliders', { sliders: out });
    }, 50);

    function setConn(state) {
      const c = st.connection, now = Date.now();
      const en = langParam === 'en';
      if (state === 'connected') Object.assign(c, { state, port: 'COM3', device: 'Arduino Uno', attempt: 0, since: iso(now), lastError: '', lastEvent: `${en ? 'Reconnected at' : 'Ricollegato alle'} ${hhmm(now)} · ${st.sliders.length} ${en ? 'sliders detected' : 'slider rilevati'}` });
      else if (state === 'reconnecting') Object.assign(c, { state, attempt: 1, since: iso(now), lastError: '', lastEvent: `${en ? 'Disconnected at' : 'Scollegato alle'} ${hhmm(now)}` });
      else if (state === 'disconnected') Object.assign(c, { state, port: 'COM3', device: '', attempt: 0, lastError: '', lastEvent: `${en ? 'No Arduino found at' : 'Nessun Arduino alle'} ${hhmm(now)}` });
      else if (state === 'busy') Object.assign(c, { state, port: 'COM3', device: 'Arduino Uno', attempt: 0, lastError: 'open COM3: Access is denied.', lastEvent: `${en ? 'COM3 busy at' : 'COM3 occupata alle'} ${hhmm(now)}` });
      emit('connection', clone(c));
    }
    if (cycle) {
      const seq = ['connected', 'reconnecting', 'disconnected', 'busy'];
      let k = 0;
      setInterval(() => { k = (k + 1) % seq.length; setConn(seq[k]); }, 6000);
      setInterval(() => {
        if (st.connection.state === 'reconnecting') { st.connection.attempt++; emit('connection', clone(st.connection)); }
      }, 2000);
    }

    return {
      async get(path) {
        await sleep(60);
        if (path === '/api/state') return clone(st);
        if (path === '/api/sessions') return { sessions: sessions.map((s) => Object.assign({}, s, { mapped: isMapped(s.key) })) };
        if (path === '/api/ports') return { ports: [{ name: 'COM3', description: 'Arduino Uno', isArduino: true }, { name: 'COM4', description: 'Collegamento Bluetooth standard', isArduino: false }] };
        throw new Error('404');
      },
      async post(path, body) {
        await sleep(150 + Math.random() * 100);
        if (fail && (path === '/api/settings' || path === '/api/mapping')) throw new Error('impossibile scrivere config.yaml: accesso negato');
        if (path === '/api/settings') {
          const b = body || {};
          if (b.reconnectInterval != null && !(b.reconnectInterval >= 1 && b.reconnectInterval <= 30)) throw new Error('reconnectInterval fuori intervallo (1-30)');
          for (const k of Object.keys(b)) {
            if (k === 'filter') st.settings.filter = Object.assign({}, st.settings.filter, b.filter);
            else if (k in st.settings && k !== 'configPath') st.settings[k] = b[k];
          }
          if (PRESETS[st.settings.noiseReduction]) st.settings.filter = Object.assign({}, PRESETS[st.settings.noiseReduction]);
          setTimeout(() => emit('state', clone(st)), 30);
          return clone(st);
        }
        if (path === '/api/mapping') {
          const old = new Map(st.sliders.map((s) => [s.index, s]));
          st.sliders = ((body && body.sliders) || []).map((x) => {
            const o = old.get(x.index);
            return { index: x.index, name: x.name || '', targets: (x.targets || []).slice(), raw: o ? o.raw : 0.5, value: o ? o.value : 0.5 };
          }).sort((a, b) => a.index - b.index);
          setTimeout(() => emit('state', clone(st)), 30);
          return clone(st);
        }
        if (path === '/api/reconnect') {
          setConn('reconnecting');
          setTimeout(() => setConn('connected'), 1500);
          return null;
        }
        if (path === '/api/open') { console.info('[mock] apri', body); return null; }
        throw new Error('404');
      },
      events() {
        const listeners = {};
        const fire = (type, ev) => (listeners[type] || []).forEach((fn) => fn(ev || {}));
        const src = {
          readyState: 0,
          addEventListener(type, fn) { (listeners[type] = listeners[type] || []).push(fn); },
          close() { src.readyState = 2; sources.delete(src); },
          _emit(type, data) { if (src.readyState === 1) fire(type, { data: JSON.stringify(data) }); }
        };
        sources.add(src);
        setTimeout(() => { if (src.readyState !== 2) { src.readyState = 1; fire('open'); } }, 50);
        if (drop) {
          setTimeout(() => {
            if (src.readyState === 2) return;
            src.readyState = 0; fire('error');
            setTimeout(() => { if (src.readyState !== 2) { src.readyState = 1; fire('open'); } }, 6000);
          }, 5000);
        }
        return src;
      }
    };
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
