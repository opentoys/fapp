<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Circle } from 'lucide-vue-next'
import { RadioGroupIndicator, RadioGroupItem, type RadioGroupItemProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<RadioGroupItemProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <RadioGroupItem
    v-bind="props"
    :class="cn('border-input text-primary shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary size-4 shrink-0 rounded-full border', props.class)"
  >
    <RadioGroupIndicator class="relative flex items-center justify-center">
      <Circle class="fill-primary absolute top-1/2 left-1/2 size-2 -translate-x-1/2 -translate-y-1/2" />
    </RadioGroupIndicator>
  </RadioGroupItem>
</template>
