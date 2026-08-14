<script setup lang="ts">
import { SelectContent, SelectPortal, SelectRoot, SelectTrigger, SelectValue } from 'reka-ui'
import { cn } from '../../../lib/utils'

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
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="bg-popover text-popover-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 relative z-50 max-h-96 min-w-32 origin-(--reka-select-content-transform-origin) overflow-y-auto rounded-md border shadow-md">
        <div class="p-1">
          <template v-for="item in props.items" :key="String(item.value)">
            <div
              class="data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer select-none rounded-sm px-2 py-1.5 text-sm outline-none"
              data-reka-select-item=""
              :data-value="String(item.value)"
              @click="emit('update:modelValue', fromStr(String(item.value)))"
            >
              {{ item.title }}
            </div>
          </template>
        </div>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
