<script setup lang="ts">
import type { HTMLAttributes } from 'vue'

const props = defineProps<{ class?: HTMLAttributes['class']; colspan?: number }>()
</script>

<template>
  <tr :class="props.class">
    <td :colspan="props.colspan ?? 1" class="text-muted-foreground p-6 text-center">
      <slot />
    </td>
  </tr>
</template>
