<script setup lang="ts">
import { TabsRoot, type TabsRootEmits, type TabsRootProps } from 'reka-ui'

const props = withDefaults(defineProps<TabsRootProps>(), {})
const emits = defineEmits<TabsRootEmits>()
</script>

<template>
  <TabsRoot v-bind="props" @update:model-value="emits('update:modelValue', $event)">
    <slot />
  </TabsRoot>
</template>
