<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle, DialogTrigger } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ title?: string; class?: string; maxWidth?: string }>(), {
  title: '',
  class: '',
  maxWidth: 'lg',
})

const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DialogTrigger>
    <DialogPortal>
      <DialogOverlay class="bg-black/80 fixed inset-0 z-50" />
      <DialogContent
        :class="cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-lg border p-6 shadow-lg duration-200', maxWidth === 'md' ? 'sm:max-w-md' : 'sm:max-w-lg', props.class)"
      >
        <div class="flex items-start justify-between gap-4">
          <DialogTitle v-if="title" class="text-lg font-semibold leading-none">{{ title }}</DialogTitle>
          <span v-else />
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-none"
            @click="open = false"
          >
            <X class="size-4" />
          </button>
        </div>
        <slot />
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
