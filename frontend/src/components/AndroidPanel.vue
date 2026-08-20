<script lang="ts" setup>
const props = defineProps<{
  dbPath: string
  busy: boolean
  status: string
  error: string
}>()

const emit = defineEmits<{
  (e: 'auto'): void
  (e: 'pick'): void
}>()
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>2. 安卓存档文件 (game.db)</h2>
    </header>
    <div class="panel-body">
      <div class="row">
        <button class="btn" :disabled="props.busy" @click="emit('auto')">自动从手机获取</button>
        <button class="btn" :disabled="props.busy" @click="emit('pick')">手动选择 game.db</button>
      </div>
      <div class="row" v-if="props.dbPath">
        <span class="label">当前存档文件：</span>
        <code class="path">{{ props.dbPath }}</code>
      </div>
      <p v-if="props.status" class="hint">{{ props.status }}</p>
      <p v-if="props.error" class="error">{{ props.error }}</p>
    </div>
  </section>
</template>
