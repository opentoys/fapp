<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'

export interface LineChartSeries {
  name: string
  color: string
  values: number[]
}

const props = withDefaults(
  defineProps<{
    dates: string[]
    series: LineChartSeries[]
    height?: number
    emptyText?: string
  }>(),
  { height: 260, emptyText: '' }
)

// --- Responsive width ---
const container = ref<HTMLDivElement | null>(null)
const width = ref(0)
let ro: ResizeObserver | null = null
onMounted(() => {
  ro = new ResizeObserver(() => {
    if (container.value) width.value = container.value.clientWidth
  })
  if (container.value) {
    ro.observe(container.value)
    width.value = container.value.clientWidth
  }
})
onUnmounted(() => ro?.disconnect())

// --- Scales ---
const PAD = { top: 10, right: 12, bottom: 26, left: 36 }
const n = computed(() => props.dates.length)

const yMax = computed(() => {
  let m = 0
  for (const s of props.series) for (const v of s.values) if (v > m) m = v
  if (m <= 0) return 1
  const pow = Math.pow(10, Math.floor(Math.log10(m)))
  const d = m / pow
  const t = d <= 1 ? 1 : d <= 2 ? 2 : d <= 5 ? 5 : 10
  return t * pow
})

const plotW = computed(() => Math.max(0, width.value - PAD.left - PAD.right))
const plotH = computed(() => props.height - PAD.top - PAD.bottom)

function x(i: number): number {
  if (n.value <= 1) return PAD.left + plotW.value / 2
  return PAD.left + (i * plotW.value) / (n.value - 1)
}
function y(v: number): number {
  return PAD.top + plotH.value - (plotH.value * v) / yMax.value
}

// Horizontal gridlines + y-axis labels (5 divisions).
const gridlines = computed(() => {
  const lines: { y: number; label: string }[] = []
  for (let g = 0; g <= 4; g++) {
    const v = (yMax.value * g) / 4
    lines.push({ y: y(v), label: String(Math.round(v)) })
  }
  return lines
})

// X tick labels: up to ~7, evenly spaced, last one always shown.
const xTicks = computed(() => {
  const total = n.value
  if (!total) return []
  const step = Math.max(1, Math.ceil(total / 7))
  const ticks: { i: number; label: string }[] = []
  for (let i = 0; i < total; i += step) ticks.push({ i, label: props.dates[i] })
  if (ticks[ticks.length - 1]?.i !== total - 1) {
    ticks.push({ i: total - 1, label: props.dates[total - 1] })
  }
  return ticks
})

function points(values: number[]): string {
  return values.map((v, i) => `${x(i)},${y(v)}`).join(' ')
}

// --- Hover ---
const hover = ref<number | null>(null)
function onMove(e: MouseEvent) {
  if (!container.value || n.value === 0) return
  const rect = container.value.getBoundingClientRect()
  const px = e.clientX - rect.left
  const step = plotW.value / Math.max(1, n.value - 1)
  let idx = Math.round((px - PAD.left) / step)
  idx = Math.max(0, Math.min(n.value - 1, idx))
  hover.value = idx
}
function onLeave() {
  hover.value = null
}

const tooltipPos = computed(() => {
  if (hover.value === null) return { left: 0, top: 0 }
  let top = Infinity
  for (const s of props.series) {
    const yy = y(s.values[hover.value] ?? 0)
    if (yy < top) top = yy
  }
  const left = Math.max(4, Math.min(width.value - 156, x(hover.value) - 70))
  return { left, top: Math.max(4, top - 82) }
})
</script>

<template>
  <div
    ref="container"
    class="relative w-full select-none"
    :style="{ height: height + 'px' }"
    @mousemove="onMove"
    @mouseleave="onLeave"
  >
    <div v-if="n === 0" class="flex h-full items-center justify-center text-muted-foreground">
      {{ emptyText }}
    </div>
    <svg v-else-if="width > 0" :width="width" :height="height">
      <!-- Horizontal gridlines + y labels -->
      <g v-for="(g, i) in gridlines" :key="'g' + i">
        <line :x1="PAD.left" :x2="width - PAD.right" :y1="g.y" :y2="g.y" class="stroke-foreground/15" />
        <text :x="PAD.left - 6" :y="g.y + 3" class="fill-foreground/60 text-[11px]" text-anchor="end">{{ g.label }}</text>
      </g>
      <!-- X labels -->
      <g v-for="t in xTicks" :key="'x' + t.i">
        <text :x="x(t.i)" :y="height - 6" class="fill-foreground/60 text-[11px]" text-anchor="middle">{{ t.label }}</text>
      </g>
      <!-- Series lines -->
      <polyline
        v-for="(s, si) in series"
        :key="'l' + si"
        :points="points(s.values)"
        fill="none"
        stroke-width="2"
        stroke-linejoin="round"
        stroke-linecap="round"
        :style="{ stroke: s.color }"
      />
      <!-- Hover guide + dots -->
      <g v-if="hover !== null">
        <line :x1="x(hover)" :x2="x(hover)" :y1="PAD.top" :y2="PAD.top + plotH" class="stroke-foreground/35" stroke-dasharray="3 3" />
        <circle
          v-for="(s, si) in series"
          :key="'d' + si"
          :cx="x(hover)"
          :cy="y(s.values[hover] ?? 0)"
          r="3.5"
          class="stroke-background"
          stroke-width="1.5"
          :style="{ fill: s.color }"
        />
      </g>
    </svg>

    <!-- Tooltip -->
    <div
      v-if="hover !== null"
      class="pointer-events-none absolute z-5 w-[140px] rounded-md bg-background p-1.5 text-xs text-foreground shadow-lg"
      :style="{ left: tooltipPos.left + 'px', top: tooltipPos.top + 'px' }"
    >
      <div class="mb-1 font-semibold">{{ dates[hover] }}</div>
      <div v-for="(s, si) in series" :key="'t' + si" class="mt-0.5 flex items-center gap-1.5">
        <span class="size-2 shrink-0 rounded-full" :style="{ background: s.color }" />
        <span class="flex-1 truncate">{{ s.name }}</span>
        <b>{{ s.values[hover] ?? 0 }}</b>
      </div>
    </div>
  </div>
</template>
