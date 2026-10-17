<script setup lang="ts">
import {
  computed,
  onMounted,
  onBeforeUnmount,
  ref,
  watch,
  nextTick,
} from "vue";
import { Window, Events } from "@wailsio/runtime";
import {
  Power,
  Settings2,
  Plus,
  X,
  Minus,
  ArrowLeft,
  Check,
  ChevronRight,
  Trash2,
  RefreshCw,
  Sun,
  Moon,
  Monitor,
  Palette,
  GitBranch,
  Link2,
  ArrowDown,
  ArrowUp,
  ShieldCheck,
  Copy,
  ScrollText,
} from "lucide-vue-next";
import { backendApi as api } from "./api";
import type { AppConfig, RuntimeState } from "./types";
import ModeSelect from "./ModeSelect.vue";
import UpdatesPanel from "./UpdatesPanel.vue";
const draft = ref<AppConfig>();
const config = ref<AppConfig>(),
  state = ref<RuntimeState>();
const page = ref<"home" | "settings" | "servers" | "logs" | "updates">("home");
const removeNavigation = Events.On("main:navigate", async (event) => {
  await reload();
  if (event.data === "settings") openSettings();
  else page.value = event.data === "servers" ? "servers" : "home";
});
const importing = ref(false),
  link = ref(""),
  name = ref(""),
  busy = ref(false),
  loading = ref(true);
const error = ref(""),
  message = ref(""),
  theme = ref("auto"),
  domains = ref(""),
  split = ref(false),
  deleteId = ref("");
const active = computed(() =>
  config.value?.nodes.find((n) => n.id === config.value?.activeNodeId),
);
const selectedProtocol = computed(() => active.value?.protocol || "sudoku");
const language = computed(() => config.value?.ui.language === 'en' ? 'en' : 'ru');
const english: Record<string,string> = {
  "Свернуть": "Minimize",
  "Скрыть в трей": "Hide to tray",
  "Загрузка…": "Loading…",
  "Настройки": "Settings",
  "Протокол": "Protocol",
  "Пожалуйста, подождите": "Please wait",
  "Соединение активно": "Connected",
  "Проверка соединения": "Checking connection",
  "Вы отключены": "Disconnected",
  "Отключить VPN": "Disconnect VPN",
  "Подключить VPN": "Connect VPN",
  "Добавьте сервер, чтобы начать": "Add a server to get started",
  "Добавить соединение": "Add connection",
  "/с": "/s",
  "VPN · всё устройство": "VPN · whole device",
  "С исключениями": "With exclusions",
  "Весь трафик": "All traffic",
  "Назад": "Back",
  "НАСТРОЙКИ": "SETTINGS",
  "СОЕДИНЕНИЯ": "CONNECTIONS",
  "ЖУРНАЛ": "LOGS",
  "ВНЕШНИЙ ВИД": "APPEARANCE",
  "Система": "System",
  "Светлая": "Light",
  "Тёмная": "Dark",
  "ПОДКЛЮЧЕНИЕ": "CONNECTION",
  "Запускать вместе с Windows": "Launch with Windows",
  "Подключаться при запуске": "Connect on startup",
  "РАЗДЕЛЬНОЕ ТУННЕЛИРОВАНИЕ": "SPLIT TUNNELING",
  "Для Mist выберите Весь трафик.": "For Mist, select All traffic.",
  "Режим": "Mode",
  "Домены исключений": "Excluded domains",
  "Один домен на строку. Включая поддомены.": "One domain per line. Includes subdomains.",
  "Настройки подключения применятся с переподключением.": "Connection settings will reconnect the VPN.",
  "Сохранить настройки": "Save settings",
  "Журнал подключения": "Connection log",
  "Обновить подписку": "Refresh subscription",
  "Удалить сервер": "Delete server",
  "Удалить соединение?": "Delete connection?",
  "Отмена": "Cancel",
  "Удалить": "Delete",
  "Здесь появятся ваши соединения": "Your connections will appear here",
  "Последние события VPN-службы. Журнал может содержать адреса\n            серверов.": "Recent VPN service events. Logs may include server addresses.",
  "Событий пока нет": "No events yet",
  "Скопировать журнал": "Copy log",
  "НОВОЕ СОЕДИНЕНИЕ": "NEW CONNECTION",
  "Закрыть": "Close",
  "Вставьте ссылку сервера или конфигурацию Mist / AmneziaWG (.conf).": "Paste a server link or Mist / AmneziaWG (.conf) configuration.",
  "Ссылка соединения": "Connection link",
  "Название": "Name",
  "· необязательно": "· optional",
  "Мой сервер": "My server",
  "sudoku://, vless://, vmess://, trojan://, JSON Mist или .conf AmneziaWG": "sudoku://, vless://, vmess://, trojan:// Mist JSON or AmneziaWG .conf",
  "Sudoku и подписки Sudoku UI · Xray с xhttp · Mist JSON · AmneziaWG .conf": "Sudoku links and subscriptions · Xray with xhttp · Mist JSON · AmneziaWG .conf",
  "Добавление…": "Adding…",
  "Сервер добавлен": "Server added",
  "Подписка обновлена": "Subscription updated",
  "Настройки сохранены": "Settings saved",
  "Журнал скопирован": "Log copied",
  "Нет связи с VPN-службой": "VPN service unavailable",
  "Укажите домены без https://, путей и пробелов": "Enter domains without https://, paths or spaces",
  "Некорректный домен: ": "Invalid domain: ",
  "Не удалось загрузить приложение: ": "Could not load the app: ",
  "Последние события VPN-службы. Журнал может содержать адреса серверов.": "Recent VPN service events. Logs may include server addresses.",
  "Б": "B",
  "КБ": "KB",
  "МБ": "MB"
};
const tr = (value: string) => language.value === 'en' ? (english[value] || value) : value;
async function toggleLanguage() {
  await action(async () => {
    if (!config.value) return;
    const c = JSON.parse(JSON.stringify(config.value)) as AppConfig;
    c.ui.language = language.value === 'ru' ? 'en' : 'ru';
    await api.saveConfig(c);
    config.value = c;
    if (draft.value) draft.value.ui.language = c.ui.language;
  });
}
watch(language, value => { document.documentElement.lang = value; }, { immediate: true });
const running = computed(() => !!state.value?.running);
const healthy = computed(
  () =>
    running.value &&
    state.value?.coreRunning &&
    (!config.value?.tun.enabled || state.value?.tunRunning) &&
    !state.value?.routeSetupError,
);
const sample = computed(() => state.value?.traffic.recentBandwidth?.at(-1));
let timer: ReturnType<typeof setInterval> | undefined;
let polling = false;
const media = matchMedia("(prefers-color-scheme: dark)");
function applyTheme() {
  document.documentElement.dataset.theme =
    theme.value === "auto" ? (media.matches ? "dark" : "light") : theme.value;
}
function errorMessage(e: unknown) {
  const text = e instanceof Error ? e.message : String(e);
  try {
    const parsed = JSON.parse(text);
    if (typeof parsed.message === "string") return parsed.message;
  } catch {
    /* plain errors */
  }
  return text;
}
async function action(fn: () => Promise<unknown>) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  message.value = "";
  try {
    await fn();
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}
async function reload() {
  config.value = await api.getConfig();
  state.value = await api.getState();
}
async function toggle() {
  await action(async () => {
    if (running.value) await api.stopProxy();
    else await api.startProxy({ withTun: true });
    await reload();
  });
}
async function importLink() {
  await action(async () => {
    await api.importShortLink(name.value.trim(), link.value.trim());
    await reload();
    importing.value = false;
    link.value = "";
    name.value = "";
    message.value = "Сервер добавлен";
  });
}
async function readConnectionFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  if (file.size > 1048576) { error.value = language.value === 'en' ? 'Configuration exceeds 1 MB' : 'Конфигурация больше 1 МБ'; return; }
  try { link.value = await file.text(); if (!name.value) name.value = file.name.replace(/\.conf$/i, ''); }
  catch { error.value = language.value === 'en' ? 'Cannot read configuration' : 'Не удалось прочитать конфигурацию'; }
}
async function select(id: string) {
  await action(async () => {
    await api.switchNode(id);
    await reload();
    page.value = "home";
  });
}
async function remove(id: string) {
  await action(async () => {
    await api.deleteNode(id);
    deleteId.value = "";
    await reload();
  });
}
async function refresh(id: string) {
  const n = config.value?.nodes.find((n) => n.id === id);
  if (n?.sourceUrl)
    await action(async () => {
      await api.importShortLink(n.name, n.sourceUrl!);
      await reload();
      message.value = "Подписка обновлена";
    });
}
async function setTheme(value: string) {
  await action(async () => {
    if (!config.value) return;
    const c = JSON.parse(JSON.stringify(config.value)) as AppConfig;
    c.ui.theme = value;
    await api.saveConfig(c);
    config.value = c;
    theme.value = value;
    applyTheme();
  });
}
watch(page, async () => {
  await nextTick();
  window.scrollTo(0, 0);
});
watch(domains, () => {
  error.value = "";
});
function openSettings() {
  message.value = "";
  error.value = "";
  const c = config.value;
  if (!c) return;
  draft.value = JSON.parse(JSON.stringify(c));
  split.value = c.routing.proxyMode === "pac";
  domains.value = c.routing.customRules
    .split("\n")
    .filter((s) => s.startsWith("DOMAIN-SUFFIX,"))
    .map((s) => s.split(",")[1])
    .join("\n");
  page.value = "settings";
}
async function saveSettings() {
  await action(async () => {
    if (!draft.value) return;
    const clean = [
      ...new Set(
        domains.value
          .split("\n")
          .map((s) => s.trim().toLowerCase())
          .filter(Boolean)
          .map((s) => {
            if (/[\s/:,@*]/.test(s))
              throw new Error(tr("Укажите домены без https://, путей и пробелов"));
            const host = new URL("https://" + s).hostname;
            if (
              !host.includes(".") ||
              host.length > 253 ||
              !host
                .split(".")
                .every((p) => /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(p))
            )
              throw new Error(tr("Некорректный домен: ") + s);
            return host;
          }),
      ),
    ];
    const c = JSON.parse(JSON.stringify(draft.value)) as AppConfig;
    c.ui.theme = theme.value;
    c.tun.enabled = true;
    c.routing.proxyMode = split.value ? "pac" : "global";
    c.routing.ruleUrls = [];
    c.routing.customRulesEnabled = split.value;
    c.routing.customRules = clean
      .map((s) => "DOMAIN-SUFFIX," + s + ",DIRECT")
      .join("\n");
    const reconnect =
      running.value &&
      (c.tun.enabled !== config.value?.tun.enabled ||
        JSON.stringify(c.routing) !== JSON.stringify(config.value?.routing));
    if (reconnect) await api.stopProxy();
    try {
      await api.saveConfig(c);
    } catch (e) {
      if (reconnect) {
        try {
          await api.startProxy({ withTun: true });
        } catch {
          /* The original save error is shown; reload preserves current state. */
        }
      }
      throw e;
    } finally {
      await reload();
    }
    if (reconnect) {
      await api.startProxy({ withTun: true });
      await reload();
    }
    message.value = "Настройки сохранены";
    page.value = "home";
  });
}
function bytes(n = 0) {
  return n < 1024
    ? `${Math.round(n)} ${tr("Б")}`
    : n < 1048576
      ? `${(n / 1024).toFixed(1)} ${tr("КБ")}`
      : `${(n / 1048576).toFixed(1)} ${tr("МБ")}`;
}
async function copyLogs() {
  await action(async () => {
    const logs = await api.getLogs("", 150);
    await navigator.clipboard.writeText(
      logs.map((l) => `${l.timestamp} ${l.level} ${l.message}`).join("\n"),
    );
    message.value = "Журнал скопирован";
  });
}
onMounted(async () => {
  media.addEventListener("change", applyTheme);
  try {
    await reload();
    theme.value = config.value?.ui.theme || "auto";
    applyTheme();
  } catch (e) {
    error.value = tr("Не удалось загрузить приложение: ") + String(e);
  } finally {
    loading.value = false;
  }
  timer = setInterval(async () => {
    if (polling) return;
    polling = true;
    try {
      state.value = await api.getState();
    } catch {
      error.value = "Нет связи с VPN-службой";
    } finally {
      polling = false;
    }
  }, 1500);
});
onBeforeUnmount(() => {
  removeNavigation();
  clearInterval(timer);
  media.removeEventListener("change", applyTheme);
});
</script>
<template>
  <div class="app-shell">
    <header class="titlebar">
      <span class="wordmark">CONNECT<span class="wordmark-dot">.</span></span>
      <div class="window-actions">
        <button :aria-label="tr('Свернуть')" @click="Window.Minimise()">
          <Minus :size="15" /></button
        ><button :aria-label="tr('Скрыть в трей')" @click="Window.Hide()">
          <X :size="15" />
        </button>
      </div>
    </header>
    <div v-if="loading" class="loading"> {{ tr("Загрузка…") }} </div>
    <main v-else :class="'page-' + page">
      <div v-if="busy" class="ui-progress page-progress" role="progressbar" :aria-label="language === 'en' ? 'Working…' : 'Обработка…'"></div>
      <template v-if="page === 'home'">
        <div class="eyebrow-row">
          <button
            class="icon-button"
            :aria-label="tr('Настройки')"
            @click="openSettings"
          >
            <Settings2 :size="19" />
          </button>
        </div>
        <section class="connection">
          <div class="status">
            <span class="status-dot" :class="{ online: healthy }" />{{
              busy
                ? tr("Пожалуйста, подождите")
                : healthy
                  ? tr("Соединение активно")
                  : running
                    ? tr("Проверка соединения")
                    : tr("Вы отключены")
            }}
          </div>
          <button
            class="power-button"
            :class="{ connected: healthy, working: busy }"
            :disabled="busy || (!active && !running)"
            :aria-label="running ? tr('Отключить VPN') : tr('Подключить VPN')"
            @click="toggle"
          >
            <Power :size="43" :stroke-width="1.2" />
          </button>
          <p v-if="!active" class="muted"> {{ tr("Добавьте сервер, чтобы начать") }} </p>
        </section>
        <button v-if="active" class="server-card" @click="page = 'servers'">
          <ShieldCheck :size="23" :stroke-width="1.3" /><span
            class="server-details"
            ><strong>{{ active.name }}</strong
            ><small>{{ active.serverAddress }}</small></span
          ><ChevronRight :size="17" />
        </button>
        <button v-else class="server-card" @click="importing = true">
          <Plus :size="22" /><span class="server-details"
            ><strong> {{ tr("Добавить соединение") }} </strong
            ><small>Sudoku · Xray · Mist · AmneziaWG</small></span
          ><ChevronRight :size="17" />
        </button>
        <div class="traffic">
          <div>
            <ArrowDown :size="15" /><span
              >{{ bytes(sample?.rxBps) }}<small> {{ tr("/с") }} </small></span
            >
          </div>
          <span class="traffic-divider" />
          <div>
            <ArrowUp :size="15" /><span
              >{{ bytes(sample?.txBps) }}<small> {{ tr("/с") }} </small></span
            >
          </div>
        </div>
        <div class="mode-note">
          <span class="status-dot" /> {{ tr("VPN · всё устройство") }} <span>·</span
          >{{
            config?.routing.proxyMode === "pac"
              ? tr("С исключениями")
              : tr("Весь трафик")
          }}
        </div>
      </template>
      <template v-else>
        <div class="page-heading">
          <button class="icon-button" :aria-label="tr('Назад')" @click="page = page === 'updates' ? 'settings' : 'home'">
            <ArrowLeft :size="18" />
          </button>
          <h1>
            {{
              page === "settings"
                ? tr("НАСТРОЙКИ")
                : page === "updates" ? (language === "en" ? "UPDATES" : "ОБНОВЛЕНИЕ")
                : page === "servers"
                  ? tr("СОЕДИНЕНИЯ")
                  : tr("ЖУРНАЛ")
            }}
          </h1>
        </div>
        <UpdatesPanel v-if="page === 'updates'" :language="language" :running="running" />
        <template v-if="page === 'settings' && draft">
          <section class="settings-section">
            <div class="appearance-heading"><h2><Palette :size="17" /> {{ tr("ВНЕШНИЙ ВИД") }} </h2>
              <div class="language-picker" role="group" :aria-label="language === 'ru' ? 'Язык интерфейса' : 'Interface language'">
                <button v-for="code in ['ru','en']" :key="code" :aria-pressed="language === code" :disabled="busy" @click="language !== code && toggleLanguage()">{{code.toUpperCase()}}</button>
              </div>
            </div>
            <div class="theme-picker">
              <button
                v-for="t in [
                  { id: 'auto', label: tr('Система'), icon: Monitor },
                  { id: 'light', label: tr('Светлая'), icon: Sun },
                  { id: 'dark', label: tr('Тёмная'), icon: Moon },
                ]"
                :key="t.id"
                :class="{ selected: theme === t.id }"
                :aria-pressed="theme === t.id"
                :disabled="busy"
                @click="setTheme(t.id)"
              >
                <component :is="t.icon" :size="13" />{{ t.label }}
              </button>
            </div>
          </section>
          <section class="settings-section">
            <h2><Link2 :size="17" /> {{ tr("ПОДКЛЮЧЕНИЕ") }} </h2>
            <fieldset :disabled="busy">
              <label class="toggle-row"
                > {{ tr("Запускать вместе с Windows") }} <input
                  v-model="draft.ui.launchAtLogin"
                  type="checkbox"
                  role="switch" /><span class="switch" /></label
              ><label class="toggle-row"
                > {{ tr("Подключаться при запуске") }} <input
                  v-model="draft.core.autoStart"
                  type="checkbox"
                  role="switch" /><span class="switch"
              /></label>
            </fieldset>
          </section>
          <section class="settings-section">
            <h2><GitBranch :size="17" /> {{ tr("РАЗДЕЛЬНОЕ ТУННЕЛИРОВАНИЕ") }} </h2><p v-if="selectedProtocol === 'amneziawg'" class="hint">{{ language === 'en' ? 'For AmneziaWG, select All traffic.' : 'Для AmneziaWG выберите Весь трафик.' }}</p><p v-if="selectedProtocol === 'mist'" class="hint"> {{ tr("Для Mist выберите Весь трафик.") }} </p>
            <fieldset :disabled="busy">
              <label class="field-label" for="routing"> {{ tr("Режим") }} </label
              ><ModeSelect v-model="split" :disabled="busy" :language="language" /><textarea
                v-if="split"
                v-model="domains"
                :aria-label="tr('Домены исключений')"
                placeholder="hh.ru&#10;avito.ru&#10;vk.ru"
                rows="3"
              />
              <p v-if="split" class="hint"> {{ tr("Один домен на строку. Включая поддомены.") }} </p>
            </fieldset>
          </section>
          <p v-if="running" class="hint reconnect-hint"> {{ tr("Настройки подключения применятся с переподключением.") }} </p>
          <button class="primary" :disabled="busy" @click="saveSettings"> {{ tr("Сохранить настройки") }} </button
          ><div class="settings-links"><button class="text-row" @click="page = 'updates'"><RefreshCw :size="16" />{{ language === 'en' ? 'Updates' : 'Обновление' }}<ChevronRight :size="15" /></button><button class="text-row" :aria-label="tr('Журнал подключения')" @click="page = 'logs'">
            <ScrollText :size="16" /> {{ language === 'en' ? 'Log' : 'Журнал' }} <ChevronRight
              :size="15"
            />
          </button></div>
        </template>
        <template v-if="page === 'servers'"
          ><div class="server-list">
            <div v-for="n in config?.nodes" :key="n.id" class="node-row">
              <button
                class="node-select"
                :disabled="busy"
                @click="select(n.id)"
              >
                <span class="node-check"
                  ><Check
                    v-if="n.id === config?.activeNodeId"
                    :size="17" /></span
                ><span
                  ><strong>{{ n.name }}</strong
                  ><small>{{ (n.protocol || "sudoku").toUpperCase() }} · {{ n.serverAddress }}</small></span
                ></button
              ><button
                v-if="n.sourceUrl"
                class="icon-button"
                :disabled="busy || running"
                :aria-label="tr('Обновить подписку')"
                @click="refresh(n.id)"
              >
                <RefreshCw :size="15" /></button
              ><button
                class="icon-button"
                :disabled="busy || running"
                :aria-label="tr('Удалить сервер')"
                @click="deleteId = n.id"
              >
                <Trash2 :size="15" />
              </button>
              <div v-if="deleteId === n.id" class="delete-confirm">
                <span> {{ tr("Удалить соединение?") }} </span
                ><button @click="deleteId = ''"> {{ tr("Отмена") }} </button
                ><button :disabled="busy" @click="remove(n.id)"> {{ tr("Удалить") }} </button>
              </div>
            </div>
          </div>
          <p v-if="!config?.nodes.length" class="muted empty-text"> {{ tr("Здесь появятся ваши соединения") }} </p>
          <button
            class="primary"
            :disabled="busy || running"
            @click="importing = true"
          >
            <Plus :size="16" /> {{ tr("Добавить соединение") }} </button></template
        >
        <template v-if="page === 'logs'"
          ><p class="hint">
            Последние события VPN-службы. Журнал может содержать адреса
            серверов.
          </p>
          <div class="logs">
            <div
              v-for="entry in state?.recentLogs?.slice(-60).reverse()"
              :key="entry.id"
            >
              <small>{{ entry.timestamp }} · {{ entry.level }}</small>
              <p>{{ entry.message }}</p>
            </div>
            <p v-if="!state?.recentLogs?.length"> {{ tr("Событий пока нет") }} </p>
          </div>
          <button class="text-row" :disabled="busy" @click="copyLogs">
            <Copy :size="15" /> {{ tr("Скопировать журнал") }} </button></template
        >
      </template>
      <div
        v-if="error || (running && state?.lastError) || state?.routeSetupError"
        class="notice error"
        role="alert"
      >
        <span class="notice-symbol" aria-hidden="true">!</span><span>{{ error || state?.routeSetupError || state?.lastError }}</span>
      </div>
    </main>
    <footer>
      <span :role="message ? 'status' : undefined">{{
        tr(message) || "CONNECT VPN"
      }}</span>
    </footer>
    <div
      v-if="importing"
      class="modal-backdrop"
      @click.self="!busy && (importing = false)"
    >
      <section
        class="import-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="import-title"
      >
        <div class="page-heading">
          <h1 id="import-title"> {{ tr("НОВОЕ СОЕДИНЕНИЕ") }} </h1>
          <button
            class="icon-button"
            :disabled="busy"
            :aria-label="tr('Закрыть')"
            @click="importing = false"
          >
            <X :size="18" />
          </button>
        </div>
        <p class="muted"> {{ tr("Вставьте ссылку сервера или конфигурацию Mist / AmneziaWG (.conf).") }} </p>
        <form @submit.prevent="importLink">
          <label class="field-label" for="link"> {{ tr("Ссылка соединения") }} </label
          ><textarea
            id="link"
            v-model="link"
            autofocus
            :placeholder="tr('sudoku://, vless://, vmess://, trojan://, JSON Mist или .conf AmneziaWG')"
            rows="4"
            required
            :disabled="busy"
            spellcheck="false"
          /><label class="field-label file-picker">{{ language === 'en' ? 'Or open a .conf file' : 'Или открыть файл .conf' }}<input type="file" style="display:none" accept=".conf,text/plain" :disabled="busy" @change="readConnectionFile" /></label><label class="field-label" for="name"
            > {{ tr("Название") }} <span class="muted"> {{ tr("· необязательно") }} </span></label
          ><input
            id="name"
            v-model="name"
            :placeholder="tr('Мой сервер')"
            maxlength="80"
            :disabled="busy"
          />
          <p class="hint"> {{ tr("Sudoku и подписки Sudoku UI · Xray с xhttp · Mist JSON · AmneziaWG .conf") }} </p>
          <div v-if="error" class="notice error" role="alert"><span class="notice-symbol" aria-hidden="true">!</span><span>{{ error }}</span></div>
          <button class="primary" :disabled="busy || !link.trim()">
            {{ busy ? tr("Добавление…") : tr("Добавить соединение") }}
          </button>
        </form>
      </section>
    </div>
  </div>
</template>
