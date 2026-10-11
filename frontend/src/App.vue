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
const draft = ref<AppConfig>();
const config = ref<AppConfig>(),
  state = ref<RuntimeState>();
const page = ref<"home" | "settings" | "servers" | "logs">("home");
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
const protocols = ["sudoku", "xray", "mist"];
const selectedProtocol = computed(() => active.value?.protocol || "sudoku");
async function chooseProtocol(protocol: string) { if(active.value && selectedProtocol.value===protocol) return; const node=config.value?.nodes.find(n=>(n.protocol || "sudoku")===protocol); if(node) await select(node.id); else { importing.value=true; link.value=""; name.value=""; } }
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
              throw new Error("Укажите домены без https://, путей и пробелов");
            const host = new URL("https://" + s).hostname;
            if (
              !host.includes(".") ||
              host.length > 253 ||
              !host
                .split(".")
                .every((p) => /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(p))
            )
              throw new Error("Некорректный домен: " + s);
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
    ? `${Math.round(n)} Б`
    : n < 1048576
      ? `${(n / 1024).toFixed(1)} КБ`
      : `${(n / 1048576).toFixed(1)} МБ`;
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
    error.value = "Не удалось загрузить приложение: " + String(e);
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
        <button aria-label="Свернуть" @click="Window.Minimise()">
          <Minus :size="15" /></button
        ><button aria-label="Скрыть в трей" @click="Window.Hide()">
          <X :size="15" />
        </button>
      </div>
    </header>
    <div v-if="loading" class="loading">Загрузка…</div>
    <main v-else :class="'page-' + page">
      <template v-if="page === 'home'">
        <div class="eyebrow-row">
          <button
            class="icon-button"
            aria-label="Настройки"
            @click="openSettings"
          >
            <Settings2 :size="19" />
          </button>
        </div>
        <nav class="protocol-picker" aria-label="Протокол"><button v-for="p in protocols" :key="p" :class="{selected: selectedProtocol===p}" :disabled="busy" @click="chooseProtocol(p)">{{ p==='xray' ? 'Xray' : p==='mist' ? 'Mist' : 'Sudoku' }}</button></nav>
        <section class="connection">
          <div class="status">
            <span class="status-dot" :class="{ online: healthy }" />{{
              busy
                ? "Пожалуйста, подождите"
                : healthy
                  ? "Соединение активно"
                  : running
                    ? "Проверка соединения"
                    : "Вы отключены"
            }}
          </div>
          <button
            class="power-button"
            :class="{ connected: healthy, working: busy }"
            :disabled="busy || (!active && !running)"
            :aria-label="running ? 'Отключить VPN' : 'Подключить VPN'"
            @click="toggle"
          >
            <Power :size="43" :stroke-width="1.2" />
          </button>
          <p v-if="!active" class="muted">Добавьте сервер, чтобы начать</p>
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
            ><strong>Добавить соединение</strong
            ><small>Sudoku · Xray · Mist</small></span
          ><ChevronRight :size="17" />
        </button>
        <div class="traffic">
          <div>
            <ArrowDown :size="15" /><span
              >{{ bytes(sample?.rxBps) }}<small>/с</small></span
            >
          </div>
          <span class="traffic-divider" />
          <div>
            <ArrowUp :size="15" /><span
              >{{ bytes(sample?.txBps) }}<small>/с</small></span
            >
          </div>
        </div>
        <div class="mode-note">
          <span class="status-dot" />VPN · всё устройство<span>·</span
          >{{
            config?.routing.proxyMode === "pac"
              ? "С исключениями"
              : "Весь трафик"
          }}
        </div>
      </template>
      <template v-else>
        <div class="page-heading">
          <button class="icon-button" aria-label="Назад" @click="page = 'home'">
            <ArrowLeft :size="18" />
          </button>
          <h1>
            {{
              page === "settings"
                ? "НАСТРОЙКИ"
                : page === "servers"
                  ? "СОЕДИНЕНИЯ"
                  : "ЖУРНАЛ"
            }}
          </h1>
        </div>
        <template v-if="page === 'settings' && draft">
          <section class="settings-section">
            <h2><Palette :size="17" />ВНЕШНИЙ ВИД</h2>
            <div class="theme-picker">
              <button
                v-for="t in [
                  { id: 'auto', label: 'Система', icon: Monitor },
                  { id: 'light', label: 'Светлая', icon: Sun },
                  { id: 'dark', label: 'Тёмная', icon: Moon },
                ]"
                :key="t.id"
                :class="{ selected: theme === t.id }"
                :disabled="busy"
                @click="setTheme(t.id)"
              >
                <component :is="t.icon" :size="13" />{{ t.label }}
              </button>
            </div>
          </section>
          <section class="settings-section">
            <h2><Link2 :size="17" />ПОДКЛЮЧЕНИЕ</h2>
            <fieldset :disabled="busy">
              <label class="toggle-row"
                >Запускать вместе с Windows<input
                  v-model="draft.ui.launchAtLogin"
                  type="checkbox"
                  role="switch" /><span class="switch" /></label
              ><label class="toggle-row"
                >Подключаться при запуске<input
                  v-model="draft.core.autoStart"
                  type="checkbox"
                  role="switch" /><span class="switch"
              /></label>
            </fieldset>
          </section>
          <section class="settings-section">
            <h2><GitBranch :size="17" />РАЗДЕЛЬНОЕ ТУННЕЛИРОВАНИЕ</h2><p v-if="selectedProtocol === 'mist'" class="hint">Для Mist выберите Весь трафик.</p>
            <fieldset :disabled="busy">
              <label class="field-label" for="routing">Режим</label
              ><ModeSelect v-model="split" :disabled="busy" /><textarea
                v-if="split"
                v-model="domains"
                aria-label="Домены исключений"
                placeholder="hh.ru&#10;avito.ru&#10;vk.ru"
                rows="3"
              />
              <p v-if="split" class="hint">
                Один домен на строку. Включая поддомены.
              </p>
            </fieldset>
          </section>
          <p v-if="running" class="hint reconnect-hint">
            Настройки подключения применятся с переподключением.
          </p>
          <button class="primary" :disabled="busy" @click="saveSettings">
            Сохранить настройки</button
          ><button class="text-row" @click="page = 'logs'">
            <ScrollText :size="16" />Журнал подключения<ChevronRight
              :size="15"
            />
          </button>
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
                aria-label="Обновить подписку"
                @click="refresh(n.id)"
              >
                <RefreshCw :size="15" /></button
              ><button
                class="icon-button"
                :disabled="busy || running"
                aria-label="Удалить сервер"
                @click="deleteId = n.id"
              >
                <Trash2 :size="15" />
              </button>
              <div v-if="deleteId === n.id" class="delete-confirm">
                <span>Удалить соединение?</span
                ><button @click="deleteId = ''">Отмена</button
                ><button :disabled="busy" @click="remove(n.id)">Удалить</button>
              </div>
            </div>
          </div>
          <p v-if="!config?.nodes.length" class="muted empty-text">
            Здесь появятся ваши соединения
          </p>
          <button
            class="primary"
            :disabled="busy || running"
            @click="importing = true"
          >
            <Plus :size="16" />Добавить соединение
          </button></template
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
            <p v-if="!state?.recentLogs?.length">Событий пока нет</p>
          </div>
          <button class="text-row" :disabled="busy" @click="copyLogs">
            <Copy :size="15" />Скопировать журнал
          </button></template
        >
      </template>
      <div
        v-if="error || state?.lastError || state?.routeSetupError"
        class="notice error"
        role="alert"
      >
        {{ error || state?.routeSetupError || state?.lastError }}
      </div>
    </main>
    <footer>
      <span :role="message ? 'status' : undefined">{{
        message || "CONNECT VPN"
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
          <h1 id="import-title">НОВОЕ СОЕДИНЕНИЕ</h1>
          <button
            class="icon-button"
            :disabled="busy"
            aria-label="Закрыть"
            @click="importing = false"
          >
            <X :size="18" />
          </button>
        </div>
        <p class="muted">
          Вставьте ссылку сервера или клиентскую конфигурацию Mist.
        </p>
        <form @submit.prevent="importLink">
          <label class="field-label" for="link">Ссылка соединения</label
          ><textarea
            id="link"
            v-model="link"
            autofocus
            placeholder="sudoku://, vless://, vmess://, trojan:// или JSON Mist"
            rows="4"
            required
            :disabled="busy"
            spellcheck="false"
          /><label class="field-label" for="name"
            >Название <span class="muted">· необязательно</span></label
          ><input
            id="name"
            v-model="name"
            placeholder="Мой сервер"
            maxlength="80"
            :disabled="busy"
          />
          <p class="hint">
            Sudoku и подписки Sudoku UI · Xray с xhttp · Mist JSON
          </p>
          <div v-if="error" class="notice error" role="alert">{{ error }}</div>
          <button class="primary" :disabled="busy || !link.trim()">
            {{ busy ? "Добавление…" : "Добавить соединение" }}
          </button>
        </form>
      </section>
    </div>
  </div>
</template>
