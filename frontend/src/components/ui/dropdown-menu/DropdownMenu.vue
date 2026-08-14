<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { DropdownMenuContent, DropdownMenuItem, DropdownMenuPortal, DropdownMenuRoot, DropdownMenuTrigger } from 'reka-ui'
import { cn } from '../../../lib/utils'

export interface DropdownItem {
  label: string
  value?: string
  danger?: boolean
  divider?: boolean
}

const props = withDefaults(defineProps<{
  items: DropdownItem[]
  selected?: string
  class?: string
}>(), { items: () => [], selected: '' })

const emit = defineEmits<{ select: [index: number] }>()
</script>

<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DropdownMenuTrigger>
    <DropdownMenuPortal>
      <DropdownMenuContent
        :class="cn('bg-popover text-popover-foreground z-50 min-w-32 overflow-hidden rounded-md border p-1 shadow-md', props.class)"
        :side-offset="4"
        align="end"
      >
        <template v-for="(item, i) in props.items" :key="i">
          <div v-if="item.divider" class="bg-border my-1 h-px" />
          <DropdownMenuItem
            v-else
            :class="cn(
              'focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer rounded-sm px-2 py-1.5 text-sm outline-none',
              item.danger && 'text-destructive data-[highlighted]:bg-destructive data-[highlighted]:text-white',
            )"
            @select="emit('select', i)"
          >
            <span class="flex w-full items-center justify-between gap-2">
              <span>{{ item.label }}</span>
              <Check v-if="item.value && item.value === props.selected" class="size-4" />
            </span>
          </DropdownMenuItem>
        </template>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>
