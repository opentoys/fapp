<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ class?: HTMLAttributes['class']; variant?: 'default' | 'destructive' | 'info' | 'warning' | 'success' }>(), { variant: 'default' })

const variantClass = {
  default: 'bg-card text-card-foreground',
  destructive: 'border-destructive/50 text-destructive bg-destructive/5',
  info: 'border-info/50 text-foreground bg-info/5',
  warning: 'border-warning/50 text-foreground bg-warning/5',
  success: 'border-success/50 text-foreground bg-success/5',
}
</script>

<template>
  <div :class="cn('relative w-full rounded-lg border px-4 py-3 text-sm', variantClass[props.variant], props.class)" role="alert">
    <slot />
  </div>
</template>
