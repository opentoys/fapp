<script setup lang="ts">
import { computed } from 'vue'

type Mode = 'public' | 'password' | 'expiry' | 'enabled' | 'taken_down' | 'live'

const props = defineProps<{
  mode: Mode
  label?: string
}>()

const colorVar = computed(() => {
  switch (props.mode) {
    case 'public':
    case 'enabled':
    case 'live':
      return 'var(--success)'
    case 'password':
      return 'var(--warning)'
    case 'expiry':
      return 'var(--warning)'
    case 'taken_down':
      return 'var(--danger)'
    default:
      return 'var(--text-mute)'
  }
})

const displayLabel = computed(() => {
  if (props.label) return props.label
  return props.mode.toUpperCase().replace('_', ' ')
})
</script>

<template>
  <span class="status-dot">
    <span class="dot" :style="{ color: colorVar }">●</span>
    <span class="label">{{ displayLabel }}</span>
  </span>
</template>

<style scoped>
.status-dot {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-mute);
}
.dot {
  font-size: 0.6rem;
  line-height: 1;
}
.label {
  color: var(--text-mute);
}
</style>
