<script setup lang="ts">
defineProps<{
  label: string
  value: number | string
  emphasis?: boolean
}>()
</script>

<template>
  <div class="stat" :class="{ emphasis }">
    <div class="label">{{ label }}</div>
    <div class="value">{{ value }}</div>
  </div>
</template>

<style scoped>
.stat {
  border: 1px solid var(--border);
  background: var(--surface);
  padding: var(--sp-3) var(--sp-4);
}
.label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-1);
}
.value {
  font-family: var(--font-sans);
  font-size: 1.75rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text);
}
.emphasis .value {
  color: var(--accent);
}
</style>
