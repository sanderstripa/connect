<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue';
import { Events, Window } from '@wailsio/runtime';
import { ChevronRight, ArrowLeft, X } from 'lucide-vue-next';
import { backendApi as api } from './api';
import type { AppConfig, RuntimeState } from './types';
const config = ref<AppConfig>(), state = ref<RuntimeState>();
const profiles = ref(false), busy = ref(false), error = ref('');
const panel = ref<HTMLElement>();
let observer: ResizeObserver;
let lastHeight = 0;
let timer: ReturnType<typeof setInterval>;
let polling = false;
const media = matchMedia('(prefers-color-scheme: dark)');
function theme() {
  document.documentElement.dataset.theme = config.value?.ui.theme === 'dark' ||
    (config.value?.ui.theme !== 'light' && media.matches) ? 'dark' : 'light';
}
async function refresh() {
  if (polling) return;
  polling = true;
  try { [config.value, state.value] = await Promise.all([api.getConfig(), api.getState()]); theme(); }
  catch (e) { error.value = String(e); }
  finally { polling = false; }
}
async function action(fn: () => Promise<unknown>) {
  busy.value = true; error.value = '';
  try { await fn(); await refresh(); }
  catch(e) { error.value = String(e); }
  finally { busy.value = false; }
}
function open(page: string) { Events.Emit('tray:open', page); }
function toggleConnection() { return action(() => state.value?.running ? api.stopProxy() : api.startProxy({withTun: true})); }
function selectProfile(id: string) { return action(() => api.switchNode(id)); }
onMounted(() => {
  observer = new ResizeObserver(() => {
    const height = Math.ceil(panel.value?.getBoundingClientRect().height || 196);
    if (height !== lastHeight) { lastHeight = height; void Window.SetSize(220, height).catch(() => {}); }
  });
  if (panel.value) observer.observe(panel.value);
  void refresh(); timer = setInterval(refresh, 800); media.addEventListener('change', theme);
});
onBeforeUnmount(() => { observer?.disconnect(); clearInterval(timer); media.removeEventListener('change', theme); });
</script>

<template>
  <div ref="panel" class="tray-panel">
    <div class="tray-status"><span>{{ state?.running ? 'Состояние: подключено' : 'Состояние: отключено' }}</span><button aria-label="Закрыть меню" @click="Window.Hide()"><X :size="13"/></button></div>
    <template v-if="!profiles">
      <button class="tray-connect" :disabled="busy" @click="toggleConnection">{{ busy ? 'Подождите…' : state?.running ? 'Отключить' : 'Подключить' }}</button>
      <hr>
      <button @click="profiles = true">Выбор профиля <ChevronRight :size="14"/></button>
      <button @click="open('home')">Открыть приложение</button>
      <button @click="open('settings')">Настройки</button>
      <hr>
      <button @click="Events.Emit('tray:quit')">Выход</button>
    </template>
    <template v-else>
      <button @click="profiles = false"><ArrowLeft :size="14"/> Профили</button>
      <hr>
      <div class="tray-profiles">
        <button v-for="node in config?.nodes" :key="node.id" :disabled="busy" @click="selectProfile(node.id)"><span>{{ node.name || node.serverAddress }}</span><span v-if="node.id === config?.activeNodeId">✓</span></button>
        <button v-if="!config?.nodes.length" @click="open('servers')">Добавить подключение</button>
      </div>
    </template>
    <p v-if="error" role="alert" class="tray-error">{{error}}</p>
  </div>
</template>

<style scoped>
.tray-panel { padding: 8px 10px; background: var(--bg); border: 1px solid var(--line); border-radius: 10px; overflow: auto; font-size: 11px; }
.tray-status { display: flex; align-items: center; justify-content: space-between; color: var(--muted); font-size: 10px; height: 22px; }
.tray-status button { padding: 3px; width: auto; }
button { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 7px 5px; text-align: left; border-radius: 5px; min-height: 28px; }
button:hover { background: var(--hover); }
button:focus-visible { outline: 1px solid var(--muted); }
.tray-connect { font-weight: 600; }
hr { border: 0; border-top: 1px solid var(--line); margin: 4px 2px; }
.tray-profiles { max-height: 150px; overflow-y: auto; }
.tray-profiles span:first-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tray-error { color: var(--red); font-size: 10px; overflow-wrap: anywhere; }
</style>
