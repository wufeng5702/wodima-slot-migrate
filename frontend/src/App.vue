<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { AutoFetchAndroidDB, DetectSteam, Migrate, PickAndroidDBManually, PickRemoteManually, ReadAndroidSlots } from '../wailsjs/go/main/App'
import type { android, migrate, steam } from '../wailsjs/go/models'
import AndroidPanel from './components/AndroidPanel.vue'
import MigratePanel from './components/MigratePanel.vue'
import SlotTable from './components/SlotTable.vue'
import SteamPanel from './components/SteamPanel.vue'

const steamInfo = ref<steam.Info | null>(null)
const selectedRemote = ref('')
const dbPath = ref('')
const slots = ref<android.SlotRow[]>([])
const selectedIds = ref<number[]>([])
const results = ref<migrate.MigrateResult[]>([])

const busy = ref(false)
const steamError = ref('')
const androidError = ref('')
const androidStatus = ref('')

const canMigrate = computed(
  () => selectedRemote.value !== '' && dbPath.value !== '' && selectedIds.value.length > 0
)

onMounted(() => {
  void detectSteam()
})

async function detectSteam() {
  busy.value = true
  steamError.value = ''
  try {
    const info = await DetectSteam()
    steamInfo.value = info
    if (info.users.length === 1) {
      selectedRemote.value = info.users[0].remotePath
    }
  } catch (e: any) {
    steamError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

async function pickRemote() {
  busy.value = true
  steamError.value = ''
  try {
    const dir = await PickRemoteManually()
    if (dir) selectedRemote.value = dir
  } catch (e: any) {
    steamError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

async function autoFetch() {
  busy.value = true
  androidError.value = ''
  androidStatus.value = '正在连接设备并拉取 game.db…'
  try {
    const path = await AutoFetchAndroidDB()
    dbPath.value = path
    androidStatus.value = '已获取存档文件。'
    await readSlots(path)
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
    androidStatus.value = ''
  } finally {
    busy.value = false
  }
}

async function pickAndroidDB() {
  busy.value = true
  androidError.value = ''
  androidStatus.value = ''
  try {
    const path = await PickAndroidDBManually()
    if (path) {
      dbPath.value = path
      await readSlots(path)
    }
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

async function readSlots(path: string) {
  slots.value = []
  selectedIds.value = []
  results.value = []
  const rows = await ReadAndroidSlots(path)
  slots.value = rows
}

function toggleSlot(id: number) {
  const i = selectedIds.value.indexOf(id)
  if (i >= 0) selectedIds.value = selectedIds.value.filter(x => x !== id)
  else selectedIds.value = [...selectedIds.value, id]
}

function toggleAll(on: boolean) {
  selectedIds.value = on ? slots.value.map(r => r.id) : []
}

async function doMigrate() {
  busy.value = true
  results.value = []
  try {
    const selections: migrate.SlotSelection[] = slots.value
      .filter(r => selectedIds.value.includes(r.id))
      .map(r => ({ id: r.id, slotIndex: r.slotIndex, jsonString: r.jsonString }))
    results.value = await Migrate(selectedRemote.value, selections)
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>我滴妈 · 存档迁移</h1>
      <p class="subtitle">Android → Steam (appid 3444020)</p>
    </header>
    <main class="app-main">
      <SteamPanel
        :info="steamInfo"
        :selected-remote="selectedRemote"
        :busy="busy"
        :error="steamError"
        @detect="detectSteam"
        @pick="pickRemote"
        @select="(r: string) => (selectedRemote = r)"
      />
      <AndroidPanel
        :db-path="dbPath"
        :busy="busy"
        :status="androidStatus"
        :error="androidError"
        @auto="autoFetch"
        @pick="pickAndroidDB"
      />
      <SlotTable
        :rows="slots"
        :selected-ids="new Set(selectedIds)"
        :busy="busy"
        @toggle="toggleSlot"
        @toggle-all="toggleAll"
      />
      <MigratePanel
        :results="results"
        :busy="busy"
        :can-migrate="canMigrate"
        @migrate="doMigrate"
      />
    </main>
  </div>
</template>
