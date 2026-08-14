<script setup lang="ts">
import { AvatarFallback, AvatarImage, AvatarRoot } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ src?: string; fallback?: string; class?: string }>(), {
  src: '',
  fallback: '',
})
</script>

<template>
  <AvatarRoot :class="cn('relative flex shrink-0 overflow-hidden rounded-full', props.class)">
    <AvatarImage v-if="props.src" :src="props.src" alt="" class="aspect-square size-full object-cover" />
    <AvatarFallback class="bg-muted text-muted-foreground flex size-full items-center justify-center rounded-full text-xs font-medium uppercase">
      {{ props.fallback }}
    </AvatarFallback>
  </AvatarRoot>
</template>
