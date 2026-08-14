<script setup lang="ts">
import { computed, ref } from 'vue'
import { Upload } from 'lucide-vue-next'
import { cn } from '../lib/utils'

const props = withDefaults(defineProps<{
  label?: string
  accept?: string
  multiple?: boolean
  dropZone?: boolean
  disabled?: boolean
}>(), { label: '', accept: '', multiple: false, dropZone: false })

const inputEl = ref<HTMLInputElement | null>(null)

const model = defineModel<File | File[] | null>({ default: null })
const emit = defineEmits<{ change: [File | File[]] }>()

const fileName = computed(() => {
  if (Array.isArray(model.value)) return model.value.map((f) => f.name).join(', ')
  return model.value?.name ?? ''
})

function onPick(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  const value = props.multiple ? Array.from(files) : files[0]
  model.value = value
  emit('change', value)
  if (inputEl.value) inputEl.value.value = ''
}

function open() {
  if (!props.disabled) inputEl.value?.click()
}
</script>

<template>
  <div>
    <input ref="inputEl" type="file" class="hidden" :accept="props.accept" :multiple="props.multiple" @change="onPick" />
    <button
      type="button"
      :disabled="props.disabled"
      @click="open"
      :class="cn(
        'border-input inline-flex items-center justify-center gap-2 rounded-md border text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
        props.dropZone ? 'border-dashed h-28 w-full flex-col px-4 py-6 text-center text-muted-foreground' : 'h-9 px-4 py-2',
      )"
    >
      <Upload class="size-4" />
      <span class="max-w-[220px] truncate">{{ fileName || props.label || 'Choose file' }}</span>
    </button>
  </div>
</template>
