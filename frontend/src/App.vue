<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { AutoFetchAndroidDB, CheckWifiUpload, DetectSteam, Migrate, PickAndroidDBManually, PickRemoteManually, ReadAndroidSlots, StartWifiServer, StopWifiServer } from '../wailsjs/go/main/App'
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

// Wi-Fi transfer state
const wifiUrl = ref('')
const wifiLocalUrl = ref('')
const wifiActive = ref(false)
const wifiWaiting = ref(false)
const wifiError = ref('')
const wifiStatus = ref('')

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
  await clearWifiState()
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
  await clearWifiState()
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

// Poll timer reference for cleanup
let wifiPollTimer: ReturnType<typeof setInterval> | null = null
let wifiPollTimeout: ReturnType<typeof setTimeout> | null = null

function stopWifiPolling() {
  if (wifiPollTimer) {
    clearInterval(wifiPollTimer)
    wifiPollTimer = null
  }
  if (wifiPollTimeout) {
    clearTimeout(wifiPollTimeout)
    wifiPollTimeout = null
  }
}

// Clear all Wi-Fi state and stop server if running
async function clearWifiState() {
  if (wifiActive.value) {
    stopWifiPolling()
    try {
      await StopWifiServer()
    } catch {
      // Ignore errors when stopping
    }
  }
  wifiActive.value = false
  wifiUrl.value = ''
  wifiLocalUrl.value = ''
  wifiWaiting.value = false
  wifiError.value = ''
  wifiStatus.value = ''
}

async function toggleWifi() {
  console.log('toggleWifi called, wifiActive:', wifiActive.value)
  if (wifiActive.value) {
    // Stop Wi-Fi server
    stopWifiPolling()
    try {
      await StopWifiServer()
    } catch (e: any) {
      wifiError.value = String(e?.message ?? e)
    }
    wifiActive.value = false
    wifiUrl.value = ''
    wifiLocalUrl.value = ''
    wifiWaiting.value = false
    wifiError.value = ''
  } else {
    // Start Wi-Fi server
    busy.value = true
    wifiError.value = ''
    wifiStatus.value = '正在启动 Wi-Fi 服务器…'
    try {
      console.log('Calling StartWifiServer...')
      const result = await StartWifiServer()
      console.log('StartWifiServer result:', result)
      wifiUrl.value = result.url
      wifiLocalUrl.value = result.localUrl
      wifiActive.value = true
      wifiWaiting.value = true
      wifiStatus.value = '等待手机上传存档文件…'
      // Poll for upload completion
      pollWifiUpload(result.token)
    } catch (e: any) {
      console.error('StartWifiServer failed:', e)
      wifiError.value = String(e?.message ?? e)
      wifiStatus.value = ''
    } finally {
      busy.value = false
    }
  }
}

function pollWifiUpload(token: string) {
  // Clear any existing polling
  stopWifiPolling()

  // Poll every 2 seconds for file upload completion
  wifiPollTimer = setInterval(async () => {
    try {
      const path = await CheckWifiUpload(token)
      if (path) {
        // File uploaded successfully
        stopWifiPolling()
        wifiWaiting.value = false
        wifiActive.value = false
        wifiUrl.value = ''
        wifiLocalUrl.value = ''
        wifiStatus.value = ''
        dbPath.value = path
        androidStatus.value = 'Wi-Fi 上传成功，已加载存档文件。'
        await readSlots(path)
      }
    } catch {
      // Still waiting, ignore errors
    }
  }, 2000)

  // Timeout after 10 minutes
  wifiPollTimeout = setTimeout(() => {
    stopWifiPolling()
    if (wifiActive.value && wifiWaiting.value) {
      wifiWaiting.value = false
      wifiStatus.value = '等待超时，请重新启动 Wi-Fi 传输。'
      wifiError.value = '上传超时（10 分钟）'
    }
  }, 10 * 60 * 1000)
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>我在地府打麻将 · 存档迁移</h1>
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
        :wifi-url="wifiUrl"
        :wifi-local-url="wifiLocalUrl"
        :wifi-active="wifiActive"
        :wifi-waiting="wifiWaiting"
        :wifi-error="wifiError"
        :wifi-status="wifiStatus"
        @auto="autoFetch"
        @pick="pickAndroidDB"
        @wifi="toggleWifi"
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
