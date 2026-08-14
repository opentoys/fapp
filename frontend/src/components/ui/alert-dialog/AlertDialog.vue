<script setup lang="ts">
import {
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
} from 'reka-ui'

const props = withDefaults(defineProps<{ title?: string; description?: string; maxWidth?: string }>(), {
  title: '',
  description: '',
  maxWidth: 'md',
})

const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <AlertDialogRoot v-model:open="open">
    <slot name="trigger" />
    <AlertDialogPortal>
      <AlertDialogOverlay class="bg-black/80 fixed inset-0 z-50" />
      <AlertDialogContent class="bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-lg border p-6 shadow-lg duration-200 sm:max-w-md">
        <div class="flex flex-col gap-2 text-center sm:text-left">
          <AlertDialogTitle v-if="title" class="text-lg font-semibold">{{ title }}</AlertDialogTitle>
          <AlertDialogDescription v-if="description" class="text-muted-foreground text-sm" v-html="description" />
        </div>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <slot name="footer" />
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
