<script lang="ts" setup>
import {computed} from 'vue'
import type {SlotRow} from "../../bindings/wodima-slot-migrate/internal/android";

const props = defineProps<{
  rows: SlotRow[]
  selectedIds: Set<number>
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle', id: number): void
  (e: 'toggleAll', on: boolean): void
}>()

const allSelected = computed(
    () => props.rows.length > 0 && props.rows.every(r => props.selectedIds.has(r.id))
)

function formatSize(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(2) + ' MB'
}

// Detect slotIndex collisions across selected rows so we can warn the user:
// migrating two rows into the same Slot{X}.json would overwrite each other.
const slotCollision = computed(() => {
  const seen = new Map<number, number>()
  for (const r of props.rows) {
    if (props.selectedIds.has(r.id)) {
      if (seen.has(r.slotIndex)) return r.slotIndex
      seen.set(r.slotIndex, r.id)
    }
  }
  return -1
})
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>3. 选择要迁移的存档</h2>
      <label class="check-all" v-if="props.rows.length">
        <input
            type="checkbox"
            :checked="allSelected"
            :disabled="props.busy"
            @change="emit('toggleAll', !allSelected)"
        />
        全选
      </label>
    </header>
    <div class="panel-body">
      <p v-if="!props.rows.length" class="hint">请先选择安卓存档文件。</p>
      <p v-else-if="slotCollision >= 0" class="warn">
        注意：选中的行中有多条对应 Slot{{ slotCollision }}，迁移时后写入的会覆盖先写入的。
      </p>
      <table class="slot-table" v-if="props.rows.length">
        <thead>
        <tr>
          <th class="col-check"></th>
          <th class="col-slot">slotIndex</th>
          <th class="col-account">userAccount</th>
          <th class="col-size">JSON 大小</th>
          <th class="col-preview">JSON 预览</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="r in props.rows" :key="r.id">
          <td class="col-check">
            <input
                type="checkbox"
                :checked="props.selectedIds.has(r.id)"
                :disabled="props.busy"
                @change="emit('toggle', r.id)"
            />
          </td>
          <td class="col-slot">{{ r.slotIndex }}</td>
          <td class="col-account">{{ r.userAccount || '(空)' }}</td>
          <td class="col-size">{{ formatSize(r.jsonSize) }}</td>
          <td class="col-preview"><code>{{ r.jsonPreview }}</code></td>
        </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
