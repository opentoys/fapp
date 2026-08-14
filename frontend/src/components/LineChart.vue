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
    class="line-chart"
    :style="{ height: height + 'px' }"
    @mousemove="onMove"
    @mouseleave="onLeave"
  >
    <div v-if="n === 0" class="d-flex align-center justify-center h-100 text-medium-emphasis">
      {{ emptyText }}
    </div>
    <svg v-else-if="width > 0" :width="width" :height="height">
      <!-- Horizontal gridlines + y labels -->
      <g v-for="(g, i) in gridlines" :key="'g' + i">
        <line :x1="PAD.left" :x2="width - PAD.right" :y1="g.y" :y2="g.y" class="chart-grid" />
        <text :x="PAD.left - 6" :y="g.y + 3" class="chart-axis-label" text-anchor="end">{{ g.label }}</text>
      </g>
      <!-- X labels -->
      <g v-for="t in xTicks" :key="'x' + t.i">
        <text :x="x(t.i)" :y="height - 6" class="chart-axis-label" text-anchor="middle">{{ t.label }}</text>
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
        <line :x1="x(hover)" :x2="x(hover)" :y1="PAD.top" :y2="PAD.top + plotH" class="chart-guide" />
        <circle
          v-for="(s, si) in series"
          :key="'d' + si"
          :cx="x(hover)"
          :cy="y(s.values[hover] ?? 0)"
          r="3.5"
          stroke="rgb(var(--v-theme-surface))"
          stroke-width="1.5"
          :style="{ fill: s.color }"
        />
      </g>
    </svg>

    <!-- Tooltip -->
    <div v-if="hover !== null" class="chart-tooltip" :style="{ left: tooltipPos.left + 'px', top: tooltipPos.top + 'px' }">
      <div class="chart-tooltip-title">{{ dates[hover] }}</div>
      <div v-for="(s, si) in series" :key="'t' + si" class="chart-tooltip-row">
        <span class="chart-tooltip-dot" :style="{ background: s.color }" />
        <span class="chart-tooltip-name">{{ s.name }}</span>
        <b>{{ s.values[hover] ?? 0 }}</b>
      </div>
    </div>
  </div>
</template>

<style scoped>
.line-chart {
  position: relative;
  width: 100%;
  user-select: none;
}
.chart-grid {
  stroke: rgba(var(--v-theme-on-surface), 0.12);
  stroke-width: 1;
}
.chart-guide {
  stroke: rgba(var(--v-theme-on-surface), 0.35);
  stroke-width: 1;
  stroke-dasharray: 3 3;
}
.chart-axis-label {
  font-size: 11px;
  fill: rgba(var(--v-theme-on-surface), 0.6);
}
.chart-tooltip {
  position: absolute;
  width: 140px;
  padding: 6px 8px;
  border-radius: 6px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.18);
  pointer-events: none;
  font-size: 12px;
  z-index: 5;
}
.chart-tooltip-title {
  font-weight: 600;
  margin-bottom: 4px;
}
.chart-tooltip-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 2px;
}
.chart-tooltip-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.chart-tooltip-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
