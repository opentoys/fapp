<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsList, type TabsListProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <TabsList
    v-bind="props"
    :class="cn('text-muted-foreground bg-muted inline-flex h-9 items-center justify-center rounded-lg p-1', props.class)"
  >
    <slot />
  </TabsList>
</template>
