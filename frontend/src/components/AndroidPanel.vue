<script lang="ts" setup>
import { ref, watch } from 'vue'
import QRCode from 'qrcode'

const props = defineProps<{
  dbPath: string
  busy: boolean
  status: string
  error: string
  wifiUrl: string
  wifiActive: boolean
  wifiWaiting: boolean
  wifiError: string
  wifiStatus: string
}>()

const emit = defineEmits<{
  (e: 'auto'): void
  (e: 'pick'): void
  (e: 'wifi'): void
}>()

// Android device path (constant, shown for reference)
const ANDROID_DB_PATH = '/sdcard/Android/data/com.itaotuo.wodima/files/game.db'

// QR code SVG generated from wifiUrl
const qrSvg = ref('')

// Generate QR code when wifiUrl changes
watch(() => props.wifiUrl, async (url: string) => {
  if (url) {
    try {
      qrSvg.value = await QRCode.toString(url, { type: 'svg', width: 200 })
    } catch {
      qrSvg.value = ''
    }
  } else {
    qrSvg.value = ''
  }
}, { immediate: true })
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>2. 安卓存档文件 (game.db)</h2>
    </header>
    <div class="panel-body">
      <!-- Android path reference -->
      <div class="row">
        <span class="label">手机端路径：</span>
        <code class="path">{{ ANDROID_DB_PATH }}</code>
      </div>

      <!-- Action buttons -->
      <div class="row">
        <button class="btn" :disabled="props.busy" @click="emit('auto')">自动从手机获取</button>
        <button class="btn" :disabled="props.busy" @click="emit('pick')">手动选择 game.db</button>
        <button class="btn" :disabled="props.busy || props.wifiActive" @click="emit('wifi')">
          {{ props.wifiActive ? '停止 Wi-Fi 传输' : (props.busy ? '启动中…' : '通过 Wi-Fi 获取') }}
        </button>
      </div>

      <!-- Status/Error messages -->
      <p v-if="props.status" class="hint">{{ props.status }}</p>
      <p v-if="props.error" class="error">{{ props.error }}</p>
      <p v-if="props.wifiStatus" class="hint wifi-status">{{ props.wifiStatus }}</p>
      <p v-if="props.wifiError" class="error">Wi-Fi 错误：{{ props.wifiError }}</p>

      <!-- Selected local file path -->
      <div class="row" v-if="props.dbPath">
        <span class="label">已加载文件：</span>
        <code class="path">{{ props.dbPath }}</code>
      </div>

      <!-- Wi-Fi transfer panel -->
      <div class="wifi-panel" v-if="props.wifiActive">
        <div class="wifi-info" v-if="props.wifiUrl">
          <p>📱 在手机上扫描二维码，或在浏览器打开：</p>
          <code class="url">{{ props.wifiUrl }}</code>
          <p class="hint">请先将手机连入同一 Wi-Fi，在手机文件管理器中把 game.db 复制到「Download」文件夹，然后在打开的网页上选择文件并上传。</p>
        </div>
        <div class="qr-container" v-if="qrSvg" v-html="qrSvg"></div>
        <button class="btn btn-stop" :disabled="props.busy" @click="emit('wifi')">停止 Wi-Fi 传输</button>
        <p v-if="props.wifiWaiting" class="hint">等待手机上传文件中…</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.wifi-status {
  color: #667eea;
  font-weight: 500;
}
.wifi-panel {
  margin-top: 16px;
  padding: 16px;
  background: var(--color-bg-elevated);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}
.wifi-info {
  text-align: center;
}
.wifi-info .url {
  display: block;
  margin: 8px 0;
  font-size: 13px;
  word-break: break-all;
}
.qr-container {
  display: flex;
  justify-content: center;
  margin: 16px 0;
}
.qr-container svg {
  width: 200px;
  height: 200px;
}
.btn-stop {
  display: block;
  margin: 0 auto;
  background: var(--color-danger);
}
.btn-stop:hover {
  opacity: 0.9;
}
</style>
