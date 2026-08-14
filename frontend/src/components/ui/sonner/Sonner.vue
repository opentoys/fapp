<script setup lang="ts">
import { Toaster as Sonner, type ToasterProps } from 'vue-sonner'

const props = defineProps<ToasterProps>()
</script>

<template>
  <Sonner
    :position="props.position ?? 'bottom-center'"
    :rich-colors="props.richColors ?? true"
    :theme="props.theme ?? 'system'"
    :toast-options="{ class: '!font-sans' }"
  />
</template>
