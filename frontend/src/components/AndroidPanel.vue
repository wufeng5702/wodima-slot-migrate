<script lang="ts" setup>
import { ref, watch } from 'vue'
import QRCode from 'qrcode'

const props = defineProps<{
  dbPath: string
  busy: boolean
  status: string
  error: string
  wifiUrl: string
  wifiLocalUrl: string
  wifiAllUrls: string[]
  wifiActive: boolean
  wifiWaiting: boolean
  wifiError: string
  wifiStatus: string
  wifiDebugInfo: string
}>()

const showDebugInfo = ref(false)

// Wi-Fi feature flag - disabled until fully implemented
const WIFI_ENABLED = true

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
        <button v-if="WIFI_ENABLED" class="btn" :disabled="props.busy || props.wifiActive" @click="emit('wifi')">
          {{ props.wifiActive ? '停止 Wi-Fi 传输' : (props.busy ? '启动中…' : '通过 Wi-Fi 获取') }}
        </button>
      </div>

      <!-- Status/Error messages -->
      <p v-if="props.status" class="hint">{{ props.status }}</p>
      <p v-if="props.error" class="error">{{ props.error }}</p>
      <p v-if="WIFI_ENABLED && props.wifiStatus" class="hint wifi-status">{{ props.wifiStatus }}</p>
      <p v-if="WIFI_ENABLED && props.wifiError" class="error">Wi-Fi 错误：{{ props.wifiError }}</p>

      <!-- Selected local file path -->
      <div class="row" v-if="props.dbPath">
        <span class="label">已加载文件：</span>
        <code class="path">{{ props.dbPath }}</code>
      </div>

      <!-- Wi-Fi transfer panel -->
      <div class="wifi-panel" v-if="WIFI_ENABLED && props.wifiActive">
        <div class="wifi-info" v-if="props.wifiUrl">
          <p>📱 在手机上扫描二维码，或在浏览器打开：</p>
          <code class="url">{{ props.wifiUrl }}</code>
          <p class="hint">请先将手机连入同一 Wi-Fi，在手机文件管理器中把 game.db 复制到「Download」文件夹，然后在打开的网页上选择文件并上传。</p>
        </div>

        <!-- All available URLs for troubleshooting -->
        <div class="all-urls" v-if="props.wifiAllUrls.length > 1">
          <p class="section-title">🔗 所有可用地址（逐个尝试）：</p>
          <ul class="url-list">
            <li v-for="(url, idx) in props.wifiAllUrls" :key="idx" :class="{ primary: url === props.wifiUrl }">
              <code class="url">{{ url }}</code>
              <span v-if="url === props.wifiUrl" class="badge">推荐</span>
              <span v-else-if="url === props.wifiLocalUrl" class="badge local">本机</span>
            </li>
          </ul>
        </div>

        <div class="local-test" v-if="props.wifiLocalUrl">
          <p>💻 本机测试地址：</p>
          <code class="url">{{ props.wifiLocalUrl }}</code>
          <p class="hint">请先在本机浏览器测试此地址是否可访问（应显示上传页面）。</p>
        </div>

        <div class="diagnose-section" v-if="props.wifiAllUrls.length > 0">
          <p class="section-title">🔧 诊断步骤：</p>
          <ol class="diagnose-steps">
            <li>在<strong>本机浏览器</strong>打开 <code>{{ props.wifiLocalUrl }}ping</code>，应显示「pong」</li>
            <li>在<strong>手机浏览器</strong>打开 <code>{{ props.wifiUrl }}ping</code>，应显示「pong」</li>
            <li>如果第二步失败，尝试上方列表中的其他 IP 地址</li>
            <li>如果所有地址都失败，请检查手机是否在同一 Wi-Fi 下</li>
          </ol>
        </div>

        <div class="qr-container" v-if="qrSvg" v-html="qrSvg"></div>
        <div class="firewall-hint">
          <p class="hint">⚠️ <strong>提示：</strong>如果手机浏览器显示「连接被中止」或「无法访问」，通常是网络隔离问题。请确保手机和电脑<strong>在同一局域网</strong>，且电脑的防火墙未阻止入站连接。</p>
        </div>
        <div v-if="props.wifiDebugInfo" class="debug-section">
          <button class="btn-debug" @click="showDebugInfo = !showDebugInfo">
            {{ showDebugInfo ? '隐藏' : '显示' }}网络接口信息
          </button>
          <pre v-if="showDebugInfo" class="debug-info">{{ props.wifiDebugInfo }}</pre>
        </div>
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
.wifi-info, .local-test {
  text-align: center;
}
.wifi-info .url, .local-test .url {
  display: block;
  margin: 8px 0;
  font-size: 13px;
  word-break: break-all;
}
.local-test {
  margin-top: 12px;
  padding: 8px;
  background: rgba(102, 126, 234, 0.1);
  border-radius: 6px;
}
.firewall-hint {
  margin-top: 12px;
  text-align: center;
}
.firewall-hint .hint {
  background: #fff3cd;
  color: #856404;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
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
.debug-section {
  margin-top: 12px;
  text-align: center;
}
.btn-debug {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  padding: 4px 12px;
  font-size: 12px;
  border-radius: 4px;
  cursor: pointer;
}
.btn-debug:hover {
  color: var(--color-text);
  border-color: var(--color-text-muted);
}
.debug-info {
  text-align: left;
  margin: 8px auto 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  max-width: 500px;
  overflow-x: auto;
}
.all-urls {
  margin-top: 16px;
  padding: 12px;
  background: rgba(102, 126, 234, 0.05);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}
.all-urls .section-title {
  font-weight: 600;
  margin: 0 0 8px;
  color: var(--color-text);
  font-size: 13px;
}
.url-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.url-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  margin-bottom: 4px;
  background: var(--color-bg);
  border: 1px solid transparent;
}
.url-list li.primary {
  border-color: var(--color-primary);
  background: rgba(102, 126, 234, 0.1);
}
.url-list li:last-child {
  margin-bottom: 0;
}
.url-list .url {
  flex: 1;
  font-size: 12px;
  word-break: break-all;
  color: var(--color-text);
}
.url-list .badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--color-primary);
  color: white;
  white-space: nowrap;
}
.url-list .badge.local {
  background: var(--color-text-muted);
}
.diagnose-section {
  margin-top: 16px;
  padding: 12px;
  background: rgba(255, 193, 7, 0.1);
  border-radius: 8px;
  border: 1px solid rgba(255, 193, 7, 0.3);
}
.diagnose-section .section-title {
  font-weight: 600;
  margin: 0 0 8px;
  font-size: 13px;
}
.diagnose-steps {
  margin: 0;
  padding-left: 20px;
  font-size: 12px;
  line-height: 1.6;
}
.diagnose-steps li {
  margin-bottom: 4px;
}
.diagnose-steps code {
  background: rgba(0, 0, 0, 0.1);
  padding: 1px 5px;
  border-radius: 3px;
  font-family: 'Consolas', 'Monaco', monospace;
}
</style>
