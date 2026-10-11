<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue";
import { ChevronDown, Check } from "lucide-vue-next";
const model = defineModel<boolean>({ required: true });
defineProps<{ disabled?: boolean }>();
const open = ref(false),
  root = ref<HTMLElement>();
const options = [
  { value: false, label: "Весь трафик через VPN" },
  { value: true, label: "Все, кроме списка" },
];
function choose(value: boolean) {
  model.value = value;
  open.value = false;
  root.value?.querySelector("button")?.focus();
}
function outside(e: PointerEvent) {
  if (!root.value?.contains(e.target as Node)) open.value = false;
}
function key(e: KeyboardEvent) {
  if (e.key === "Escape") {
    open.value = false;
    e.preventDefault();
  }
  if (e.key === "ArrowDown" || e.key === "ArrowUp") {
    e.preventDefault();
    if (!open.value) open.value = true;
    else model.value = !model.value;
  }
  if (e.key === "Tab") open.value = false;
}
onMounted(() => document.addEventListener("pointerdown", outside));
onBeforeUnmount(() => document.removeEventListener("pointerdown", outside));
</script>
<template>
  <div ref="root" class="mode-select" @keydown="key">
    <button
      id="routing"
      type="button"
      class="mode-trigger"
      role="combobox"
      aria-label="Режим"
      aria-haspopup="listbox"
      :aria-expanded="open && !disabled"
      aria-controls="routing-options"
      :disabled="disabled"
      @click="open = !open"
    >
      {{ options.find((o) => o.value === model)?.label
      }}<ChevronDown :size="15" :class="{ rotated: open }" />
    </button>
    <div
      v-if="open && !disabled"
      id="routing-options"
      class="mode-options"
      role="listbox"
      aria-label="Режим"
    >
      <button
        v-for="o in options"
        :key="String(o.value)"
        type="button"
        role="option"
        :aria-selected="model === o.value"
        @click="choose(o.value)"
      >
        {{ o.label }}<Check v-if="model === o.value" :size="14" />
      </button>
    </div>
  </div>
</template>
