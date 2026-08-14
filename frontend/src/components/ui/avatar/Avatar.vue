<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { AvatarFallback, AvatarImage, AvatarRoot } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ src?: string; fallback?: string; class?: HTMLAttributes['class'] }>(), {
  src: '',
  fallback: '',
})
</script>

<template>
  <AvatarRoot :class="cn('relative flex size-10 shrink-0 overflow-hidden rounded-full', props.class)">
    <AvatarImage v-if="props.src" :src="props.src" alt="" class="aspect-square size-full object-cover" />
    <AvatarFallback :delay-ms="600" class="bg-muted text-muted-foreground flex size-full items-center justify-center rounded-full text-xs font-medium uppercase">
      {{ props.fallback }}
    </AvatarFallback>
  </AvatarRoot>
</template>
