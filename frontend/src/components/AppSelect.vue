<script setup lang="ts">
import { Check, ChevronDown } from 'lucide-vue-next'
import { SelectContent, SelectItem, SelectPortal, SelectRoot, SelectTrigger, SelectValue, SelectViewport } from 'reka-ui'
import { cn } from '../lib/utils'

export interface SelectOption {
  title: string
  value: string | number
}

const props = withDefaults(defineProps<{
  items: SelectOption[]
  modelValue?: string | number | null
  placeholder?: string
  class?: string
  disabled?: boolean
}>(), { placeholder: 'Select…' })

const emit = defineEmits<{ 'update:modelValue': [value: string | number | null] }>()

function toStr(v: string | number | null | undefined): string | undefined {
  if (v === null || v === undefined) return undefined
  return String(v)
}
function fromStr(s: string): string | number | null {
  return props.items.find((i) => String(i.value) === s)?.value ?? null
}
</script>

<template>
  <SelectRoot
    :model-value="toStr(props.modelValue)"
    :disabled="props.disabled"
    @update:model-value="(s: string) => emit('update:modelValue', fromStr(s))"
  >
    <SelectTrigger
      :class="cn('border-input bg-transparent shadow-xs data-[placeholder]:text-muted-foreground flex h-10 w-full items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50', props.class)"
    >
      <SelectValue :placeholder="props.placeholder" />
      <ChevronDown class="size-4 shrink-0 opacity-50" />
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="bg-popover text-popover-foreground z-50 max-h-96 min-w-32 overflow-y-auto rounded-md border shadow-md">
        <SelectViewport class="p-1">
          <SelectItem
            v-for="item in props.items"
            :key="String(item.value)"
            :value="String(item.value)"
            class="group data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer rounded-sm px-2 py-1.5 text-sm outline-none"
          >
            <span class="flex items-center justify-between gap-2">
              <span>{{ item.title }}</span>
              <Check class="size-4 opacity-0 group-data-[state=checked]:opacity-100" />
            </span>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
