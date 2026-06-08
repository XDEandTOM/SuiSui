<script setup lang="ts">
import { ref, computed } from "vue"
import { useNotesStore } from '@/stores/notes'

const store = useNotesStore()

const currentYear = ref(new Date().getFullYear())
const currentMonth = ref(new Date().getMonth())

const dayHeaders = ["日", "一", "二", "三", "四", "五", "六"]

function prevMonth() {
  if (currentMonth.value === 0) { currentMonth.value = 11; currentYear.value-- }
  else { currentMonth.value-- }
}

function nextMonth() {
  if (currentMonth.value === 11) { currentMonth.value = 0; currentYear.value++ }
  else { currentMonth.value++ }
}

const monthLabel = computed(() => currentYear.value + "年" + (currentMonth.value + 1) + "月")

const days = computed(() => {
  const year = currentYear.value, month = currentMonth.value
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const countMap = new Map<string, number>()
  for (const note of store.notes) {
    const d = new Date(note.createdAt)
    if (d.getFullYear() === year && d.getMonth() === month) {
      const key = d.getDate().toString()
      countMap.set(key, (countMap.get(key) || 0) + 1)
    }
  }
  const grid: { day: number; count: number }[][] = []
  let row: { day: number; count: number }[] = []
  for (let i = 0; i < firstDay.getDay(); i++) row.push({ day: 0, count: -1 })
  for (let d = 1; d <= lastDay.getDate(); d++) {
    const date = new Date(year, month, d)
    row.push({ day: d, count: countMap.get(d.toString()) || 0 })
    if (date.getDay() === 6) { grid.push(row); row = [] }
  }
  if (row.length > 0) {
    while (row.length < 7) row.push({ day: 0, count: -1 })
    grid.push(row)
  }
  return { grid }
})

function getColor(count: number): string {
  if (count <= 0) return "level-0"
  if (count === 1) return "level-1"
  if (count <= 3) return "level-2"
  if (count <= 6) return "level-3"
  return "level-4"
}

const hasTodayInMonth = computed(() => {
  const now = new Date()
  return currentYear.value === now.getFullYear() && currentMonth.value === now.getMonth()
})

function formatTooltip(day: number, count: number) {
  const d = currentYear.value + "-" + String(currentMonth.value + 1).padStart(2, "0") + "-" + String(day).padStart(2, "0")
  return d + ": " + count + " 条备忘"
}
</script>

<template>
  <v-card variant="outlined" class="rounded-xl pa-4 heatmap-card">
    <div class="d-flex align-center mb-3">
      <span class="text-subtitle-2 font-weight-medium">{{ monthLabel }}</span>
      <v-spacer />
      <div class="d-flex align-center ga-1">
        <v-btn icon="mdi-chevron-left" size="x-small" variant="text" @click="prevMonth" />
        <v-btn icon="mdi-chevron-right" size="x-small" variant="text" @click="nextMonth" />
      </div>
    </div>
    <div class="cal-grid">
      <div class="cal-row header-row">
        <div v-for="(h, hi) in dayHeaders" :key="hi" class="cal-cell header-cell">{{ h }}</div>
      </div>
      <div v-for="(row, ri) in days.grid" :key="ri" class="cal-row">
        <template v-for="(cell, ci) in row" :key="ci">
          <v-tooltip v-if="cell.day > 0" :text="formatTooltip(cell.day, cell.count)" location="top">
            <template #activator="{ props: tp }">
              <div v-bind="tp" class="cal-cell day-cell" :class="[
                getColor(cell.count),
                { today: hasTodayInMonth && cell.day === new Date().getDate() }
              ]">{{ cell.day }}</div>
            </template>
          </v-tooltip>
          <div v-else class="cal-cell day-cell empty-cell" />
        </template>
      </div>
    </div>
  </v-card>
</template>

<style scoped>
.heatmap-card { border: 1px solid rgba(var(--v-theme-on-surface), 0.08); }
.cal-grid { display: flex; flex-direction: column; gap: 2px; }
.cal-row { display: flex; gap: 2px; }
.cal-cell {
  flex: 1; aspect-ratio: 1; max-width: 40px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 6px; font-size: 0.75rem; font-weight: 500;
}
.header-cell { color: rgba(var(--v-theme-on-surface), 0.45); font-size: 0.7rem; font-weight: 400; }
.day-cell { cursor: pointer; transition: all 0.15s ease; }
.day-cell:hover { transform: scale(1.2); outline: 2px solid rgba(var(--v-theme-primary), 0.3); z-index: 1; }
.today { outline: 2px solid rgba(var(--v-theme-primary), 0.5); font-weight: 700; }
.empty-cell { visibility: hidden; }
.level-0 { background: rgba(var(--v-theme-on-surface), 0.04); color: rgba(var(--v-theme-on-surface), 0.5); }
.level-1 { background: rgba(var(--v-theme-primary), 0.25); }
.level-2 { background: rgba(var(--v-theme-primary), 0.45); }
.level-3 { background: rgba(var(--v-theme-primary), 0.7); color: #fff; }
.level-4 { background: rgb(var(--v-theme-primary)); color: #fff; }

@media (max-width: 768px) {
  .heatmap-card { padding: 8px !important; }
  .cal-cell { flex: 1; aspect-ratio: 1; font-size: 0.6rem; min-width: 0; }
}
</style>
