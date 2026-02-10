<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getLogs, setLogDedupEnabled, isLogDedupEnabled, getLogPlatforms } from '../api/log'
import type { SimpleLog, LogQuery, LogLevel } from '../types'
import { getPlatformLabel, getPlatformColor, PLATFORMS } from '../types'

const logs = ref<SimpleLog[]>([])
const loading = ref(false)
const keyword = ref('')
const autoRefresh = ref(false)
const dedupEnabled = ref(true)
const selectedPlatform = ref('')
const selectedLevel = ref<LogLevel>('')
const availablePlatforms = ref<string[]>([])
let refreshTimer: number | null = null

// 日志级别选项
const levelOptions = [
  { label: '全部级别', value: '' },
  { label: '信息', value: 'info' },
  { label: '警告', value: 'warn' },
  { label: '错误', value: 'error' },
  { label: '成功', value: 'success' },
  { label: '调试', value: 'debug' },
]

// 平台选项（动态生成）
const platformOptions = computed(() => [
  { label: '全部平台', value: '' },
  ...availablePlatforms.value.map(p => ({
    label: getPlatformLabel(p),
    value: p
  }))
])

// 获取日志
async function fetchLogs() {
  loading.value = true
  const query: LogQuery = {
    keyword: keyword.value,
    limit: 200,
    platform: selectedPlatform.value,
    level: selectedLevel.value
  }
  logs.value = await getLogs(query)
  loading.value = false
}

// 获取可用的平台列表
async function fetchPlatforms() {
  availablePlatforms.value = await getLogPlatforms()
}

// 搜索日志
function handleSearch() {
  fetchLogs()
}

// 清空搜索
function clearSearch() {
  keyword.value = ''
  fetchLogs()
}

// 切换自动刷新
function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    refreshTimer = window.setInterval(() => {
      fetchLogs()
      fetchPlatforms()
    }, 3000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
}

// 切换日志归并
async function toggleDedup() {
  dedupEnabled.value = !dedupEnabled.value
  await setLogDedupEnabled(dedupEnabled.value)
}

// 格式化日志消息（高亮关键词）
function formatMessage(message: string) {
  if (!keyword.value) return message
  const regex = new RegExp(`(${keyword.value})`, 'gi')
  return message.replace(regex, '<mark>$1</mark>')
}

// 按日期分组日志
const groupedLogs = computed(() => {
  const groups: Record<string, SimpleLog[]> = {}
  logs.value.forEach(log => {
    if (!groups[log.date]) {
      groups[log.date] = []
    }
    groups[log.date].push(log)
  })
  return groups
})

// 获取日志级别样式
function getLogLevelClass(level: string): string {
  switch (level) {
    case 'error':
      return 'log-error'
    case 'warn':
      return 'log-warning'
    case 'success':
      return 'log-success'
    case 'info':
      return 'log-info'
    case 'debug':
      return 'log-debug'
    default:
      return 'log-default'
  }
}

// 获取日志级别标签
function getLevelLabel(level: string): string {
  const map: Record<string, string> = {
    'error': 'ERR',
    'warn': 'WRN',
    'info': 'INF',
    'success': 'SUC',
    'debug': 'DBG'
  }
  return map[level] || level.toUpperCase()
}

// 判断是否是归并日志
function isMergedLog(message: string): boolean {
  return message.includes('重复出现') || message.includes('↳')
}

onMounted(async () => {
  // 获取归并状态
  dedupEnabled.value = await isLogDedupEnabled()
  // 获取平台列表
  await fetchPlatforms()
  // 获取日志
  await fetchLogs()
})
</script>

<template>
  <div class="logs-page">
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">系统日志</h1>
        <p class="page-subtitle">查看应用运行日志</p>
      </div>
      <div class="header-actions">
        <!-- 平台筛选 -->
        <el-select
          v-model="selectedPlatform"
          placeholder="选择平台"
          class="filter-select"
          clearable
          @change="fetchLogs"
        >
          <el-option
            v-for="opt in platformOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 级别筛选 -->
        <el-select
          v-model="selectedLevel"
          placeholder="选择级别"
          class="filter-select"
          clearable
          @change="fetchLogs"
        >
          <el-option
            v-for="opt in levelOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>

        <!-- 关键词搜索 -->
        <el-input
          v-model="keyword"
          placeholder="搜索日志关键词..."
          class="search-input"
          clearable
          @keyup.enter="handleSearch"
          @clear="clearSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>

        <!-- 刷新按钮 -->
        <el-button type="primary" @click="fetchLogs" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>

        <!-- 自动刷新 -->
        <el-button
          :type="autoRefresh ? 'success' : 'default'"
          @click="toggleAutoRefresh"
        >
          <el-icon><Timer /></el-icon>
          {{ autoRefresh ? '停止刷新' : '自动刷新' }}
        </el-button>

        <!-- 归并开关 -->
        <el-button
          :type="dedupEnabled ? 'warning' : 'default'"
          @click="toggleDedup"
          :title="dedupEnabled ? '日志归并已启用，点击禁用' : '日志归并已禁用，点击启用'"
        >
          <el-icon><Filter /></el-icon>
          {{ dedupEnabled ? '归并:开' : '归并:关' }}
        </el-button>
      </div>
    </div>

    <div class="logs-container">
      <el-empty v-if="logs.length === 0 && !loading" description="暂无日志" />
      
      <div v-else class="logs-list">
        <div
          v-for="(groupLogs, date) in groupedLogs"
          :key="date"
          class="log-group"
        >
          <div class="log-date">
            <el-icon><Calendar /></el-icon>
            <span>{{ date }}</span>
          </div>
          <div class="log-items">
            <div
              v-for="(log, index) in groupLogs"
              :key="index"
              class="log-item"
              :class="[getLogLevelClass(log.level), { 'log-merged': isMergedLog(log.message) }]"
            >
              <span class="log-time">{{ log.time }}</span>
              
              <!-- 平台标签 -->
              <span 
                v-if="log.platform" 
                class="log-platform"
                :style="{ backgroundColor: getPlatformColor(log.platform) + '20', color: getPlatformColor(log.platform), borderColor: getPlatformColor(log.platform) + '40' }"
              >
                {{ getPlatformLabel(log.platform) }}
              </span>
              <span v-else class="log-platform log-platform-system">系统</span>
              
              <!-- 级别标签 -->
              <span class="log-level" :class="'level-' + log.level">
                {{ getLevelLabel(log.level) }}
              </span>
              
              <!-- 消息内容 -->
              <span class="log-message" v-html="formatMessage(log.message)"></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs-page {
  padding: var(--spacing-6);
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-6);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: var(--text-base);
  color: var(--text-secondary);
  margin: 0;
}

.header-actions {
  display: flex;
  gap: var(--spacing-3);
  align-items: center;
  flex-wrap: wrap;
}

.filter-select {
  width: 120px;
}

.search-input {
  width: 200px;
}

.logs-container {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  padding: var(--spacing-4);
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-6);
}

.log-group {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

.log-date {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
  padding-bottom: var(--spacing-2);
  border-bottom: 1px solid var(--border-color);
}

.log-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.log-item {
  display: flex;
  gap: var(--spacing-3);
  padding: var(--spacing-2) var(--spacing-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-family: var(--font-family-mono);
  transition: background-color var(--transition-fast);
  align-items: center;
}

.log-item:hover {
  background-color: var(--hover-bg);
}

.log-merged {
  opacity: 0.8;
  font-style: italic;
}

.log-time {
  color: var(--text-tertiary);
  flex-shrink: 0;
  min-width: 70px;
  font-size: var(--text-xs);
}

.log-platform {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  border: 1px solid;
  min-width: 50px;
  text-align: center;
}

.log-platform-system {
  background-color: var(--bg-tertiary);
  color: var(--text-tertiary);
  border-color: var(--border-color);
}

.log-level {
  flex-shrink: 0;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: var(--font-bold);
  min-width: 36px;
  text-align: center;
}

.level-info {
  background-color: var(--color-info-500);
  color: white;
}

.level-warn {
  background-color: var(--color-warning-500);
  color: white;
}

.level-error {
  background-color: var(--color-error-500);
  color: white;
}

.level-success {
  background-color: var(--color-success-500);
  color: white;
}

.level-debug {
  background-color: var(--text-tertiary);
  color: white;
}

.log-message {
  color: var(--text-primary);
  word-break: break-all;
  line-height: 1.5;
  flex: 1;
}

.log-message :deep(mark) {
  background-color: var(--color-primary-500);
  color: var(--text-inverse);
  padding: 0 4px;
  border-radius: var(--radius-sm);
}

/* 日志级别样式 - 行颜色 */
.log-error .log-message {
  color: var(--color-error-400);
}

.log-success .log-message {
  color: var(--color-success-400);
}

.log-warning .log-message {
  color: var(--color-warning-400);
}

.log-info .log-message {
  color: var(--color-info-400);
}

.log-debug .log-message {
  color: var(--text-tertiary);
}

/* 浅色主题适配 */
[data-theme="light"] .log-error .log-message {
  color: var(--color-error-600);
}

[data-theme="light"] .log-success .log-message {
  color: var(--color-success-600);
}

[data-theme="light"] .log-warning .log-message {
  color: var(--color-warning-600);
}

[data-theme="light"] .log-info .log-message {
  color: var(--color-info-600);
}

[data-theme="light"] .log-message :deep(mark) {
  background-color: var(--color-primary-200);
  color: var(--color-primary-800);
}
</style>
