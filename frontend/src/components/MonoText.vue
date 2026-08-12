<script setup lang="ts">
defineProps<{
  muted?: boolean
}>()
</script>

<template>
  <span class="mono" :class="{ muted }"><slot /></span>
</template>

<style scoped>
.mono {
  font-family: var(--font-mono);
  font-feature-settings: 'tnum' 1;
}
.mono.muted {
  color: var(--text-mute);
}
</style>
