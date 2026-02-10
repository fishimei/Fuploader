<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useScheduleStore, useThemeStore, useScreenshotStore } from '../stores'
import { getAppVersion } from '../api'
import { getHeadlessConfig, setHeadlessConfig } from '../api/system'
import type { AppVersion } from '../types'

const scheduleStore = useScheduleStore()
const themeStore = useThemeStore()
const screenshotStore = useScreenshotStore()

const appVersion = ref<AppVersion | null>(null)
const timeInput = ref('')
const headlessMode = ref(false)
const headlessLoading = ref(false)

async function handleSaveSchedule() {
  try {
    if (scheduleStore.config) {
      await scheduleStore.updateConfig(scheduleStore.config)
      ElMessage.success('定时配置已保存')
    }
  } catch (error) {
    ElMessage.error('保存失败')
  }
}

function handleAddTime() {
  if (!timeInput.value) return
  
  scheduleStore.addTimeSlot(timeInput.value)
  timeInput.value = ''
}

function handleRemoveTime(index: number) {
  scheduleStore.removeTimeSlot(index)
}

async function handleSaveScreenshotConfig() {
  try {
    await screenshotStore.saveConfig()
    ElMessage.success('截图配置已保存')
  } catch (error) {
    ElMessage.error('保存截图配置失败')
  }
}

async function handleCleanOldScreenshots() {
  try {
    await ElMessageBox.confirm(
      '确定要清理旧截图吗？这将删除超出保留天数的截图。',
      '确认清理',
      {
        confirmButtonText: '清理',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const cleaned = await screenshotStore.cleanOld()
    ElMessage.success(`已清理 ${cleaned} 个旧截图`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('清理失败')
    }
  }
}

async function handleOpenScreenshotDir() {
  try {
    await screenshotStore.openDir()
  } catch (error) {
    ElMessage.error('打开目录失败')
  }
}

async function handleHeadlessModeChange() {
  headlessLoading.value = true
  try {
    await setHeadlessConfig(headlessMode.value)
    ElMessage.success(headlessMode.value ? '无头模式已启用' : '无头模式已禁用')
  } catch (error) {
    ElMessage.error('设置失败')
    headlessMode.value = !headlessMode.value
  } finally {
    headlessLoading.value = false
  }
}

onMounted(async () => {
  scheduleStore.fetchConfig()
  screenshotStore.fetchConfig()
  screenshotStore.fetchPlatformStats()
  try {
    appVersion.value = await getAppVersion()
    headlessMode.value = await getHeadlessConfig()
  } catch (error) {
    console.error('获取版本信息失败:', error)
  }
})
</script>

<template>
  <div class="settings-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">设置</h2>
        <p class="page-subtitle">配置应用参数</p>
      </div>
    </div>

    <div class="settings-container">
      <!-- 主题设置 -->
      <div class="settings-section">
        <h3 class="section-title">
          <el-icon><Brush /></el-icon>
          外观设置
        </h3>
        <div class="theme-setting">
          <span class="setting-label">主题模式</span>
          <el-radio-group v-model="themeStore.theme" @change="themeStore.applyTheme">
            <el-radio-button label="light">
              <el-icon><Sunny /></el-icon>
              浅色
            </el-radio-button>
            <el-radio-button label="dark">
              <el-icon><Moon /></el-icon>
              深色
            </el-radio-button>
          </el-radio-group>
        </div>
      </div>

      <!-- 浏览器设置 -->
      <div class="settings-section">
        <h3 class="section-title">
          <el-icon><Monitor /></el-icon>
          浏览器设置
        </h3>
        <div class="settings-form">
          <el-form label-position="top">
            <el-form-item>
              <template #label>
                <div class="form-label-with-tag">
                  <span>无头模式</span>
                  <el-tag v-if="headlessMode" type="success" size="small">已启用</el-tag>
                  <el-tag v-else type="info" size="small">已禁用</el-tag>
                </div>
              </template>
              <el-switch
                v-model="headlessMode"
                active-text="开启"
                inactive-text="关闭"
                :loading="headlessLoading"
                @change="handleHeadlessModeChange"
              />
              <div class="form-hint">
                开启后，上传视频时浏览器窗口将隐藏运行，不显示界面
              </div>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <!-- 截图设置 -->
      <div class="settings-section">
        <h3 class="section-title">
          <el-icon><Camera /></el-icon>
          截图设置
        </h3>
        
        <div class="settings-form">
          <el-form label-position="top">
            <el-form-item>
              <template #label>
                <div class="form-label-with-tag">
                  <span>启用截图</span>
                  <el-tag v-if="screenshotStore.config.enabled" type="success" size="small">已启用</el-tag>
                  <el-tag v-else type="info" size="small">已禁用</el-tag>
                </div>
              </template>
              <el-switch
                v-model="screenshotStore.config.enabled"
                active-text="开启"
                inactive-text="关闭"
                @change="handleSaveScreenshotConfig"
              />
              <div class="form-hint">
                开启后，上传过程中会自动截图记录关键步骤，便于排查问题
              </div>
            </el-form-item>

            <template v-if="screenshotStore.config.enabled">
              <el-form-item label="全局截图目录">
                <el-input v-model="screenshotStore.config.globalDir" disabled>
                  <template #append>
                    <el-button @click="handleOpenScreenshotDir">
                      <el-icon><FolderOpened /></el-icon>
                      打开
                    </el-button>
                  </template>
                </el-input>
              </el-form-item>

              <el-form-item label="自动清理">
                <el-switch
                  v-model="screenshotStore.config.autoClean"
                  active-text="开启"
                  inactive-text="关闭"
                />
                <div class="form-hint">
                  开启后，系统会自动清理超出保留天数的截图
                </div>
              </el-form-item>

              <el-form-item label="保留天数" v-if="screenshotStore.config.autoClean">
                <el-slider
                  v-model="screenshotStore.config.maxAgeDays"
                  :min="1"
                  :max="90"
                  show-stops
                  :marks="{7: '7天', 30: '30天', 60: '60天', 90: '90天'}"
                />
                <span class="slider-value">{{ screenshotStore.config.maxAgeDays }} 天</span>
              </el-form-item>

              <el-form-item label="存储限制">
                <el-slider
                  v-model="screenshotStore.config.maxSizeMB"
                  :min="100"
                  :max="2000"
                  :step="100"
                  show-stops
                />
                <span class="slider-value">{{ screenshotStore.config.maxSizeMB }} MB</span>
              </el-form-item>
            </template>
          </el-form>

          <div class="form-actions" v-if="screenshotStore.config.enabled">
            <el-button type="primary" @click="handleSaveScreenshotConfig" :loading="screenshotStore.saving">
              <el-icon><Check /></el-icon>
              保存配置
            </el-button>
            <el-button @click="handleCleanOldScreenshots" v-if="screenshotStore.config.autoClean">
              <el-icon><Delete /></el-icon>
              清理旧截图
            </el-button>
          </div>

          <!-- 平台截图统计 -->
          <div class="platform-stats" v-if="screenshotStore.platformStats.length > 0">
            <h4 class="stats-title">各平台截图统计</h4>
            <div class="stats-grid">
              <div 
                v-for="stat in screenshotStore.platformStats" 
                :key="stat.name"
                class="stat-item"
              >
                <div class="stat-name">{{ stat.name }}</div>
                <div class="stat-count">{{ stat.screenshotCount }} 张</div>
                <div class="stat-dir" :title="stat.dir">{{ stat.dir }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 定时发布配置 -->
      <div class="settings-section">
        <h3 class="section-title">
          <el-icon><Clock /></el-icon>
          定时发布配置
        </h3>
        
        <div class="settings-form" v-if="scheduleStore.config">
          <el-form label-position="top">
            <el-form-item label="每日发布数量">
              <el-slider
                v-model="scheduleStore.config.videosPerDay"
                :min="1"
                :max="10"
                show-stops
              />
              <span class="slider-value">{{ scheduleStore.config.videosPerDay }} 个/天</span>
            </el-form-item>

            <el-form-item label="发布时间">
              <div class="time-slots">
                <el-tag
                  v-for="(time, index) in scheduleStore.config.dailyTimes"
                  :key="time"
                  closable
                  @close="handleRemoveTime(index)"
                  class="time-tag"
                >
                  {{ time }}
                </el-tag>
                <el-time-picker
                  v-model="timeInput"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="添加时间"
                  size="small"
                  @change="handleAddTime"
                />
              </div>
            </el-form-item>

            <el-form-item label="时区">
              <el-select v-model="scheduleStore.config.timeZone" style="width: 100%">
                <el-option label="Asia/Shanghai (北京时间)" value="Asia/Shanghai" />
                <el-option label="Asia/Hong_Kong (香港时间)" value="Asia/Hong_Kong" />
                <el-option label="Asia/Tokyo (东京时间)" value="Asia/Tokyo" />
                <el-option label="America/New_York (纽约时间)" value="America/New_York" />
                <el-option label="Europe/London (伦敦时间)" value="Europe/London" />
              </el-select>
            </el-form-item>
          </el-form>

          <div class="form-actions">
            <el-button type="primary" @click="handleSaveSchedule">
              <el-icon><Check /></el-icon>
              保存配置
            </el-button>
          </div>
        </div>
      </div>

      <!-- 关于应用 -->
      <div class="settings-section">
        <h3 class="section-title">
          <el-icon><InfoFilled /></el-icon>
          关于应用
        </h3>
        
        <div class="about-content">
          <div class="about-logo">
            <el-icon :size="48" color="var(--primary-color)"><Upload /></el-icon>
            <h4>Fuploader</h4>
            <p>跨平台视频自动上传工具</p>
          </div>
          
          <el-descriptions :column="1" border v-if="appVersion">
            <el-descriptions-item label="版本">{{ appVersion.version }}</el-descriptions-item>
            <el-descriptions-item label="构建时间">{{ appVersion.buildTime }}</el-descriptions-item>
            <el-descriptions-item label="Go 版本">{{ appVersion.goVersion }}</el-descriptions-item>
            <el-descriptions-item label="Wails 版本">{{ appVersion.wailsVersion }}</el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  padding-bottom: var(--spacing-xl);
}

.page-header {
  margin-bottom: var(--spacing-xl);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.settings-container {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xl);
}

.settings-section {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-lg) 0;
}

.settings-form {
  max-width: 600px;
}

.form-label-with-tag {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.form-hint {
  margin-top: var(--spacing-xs);
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.slider-value {
  margin-left: var(--spacing-md);
  font-size: 13px;
  color: var(--text-secondary);
}

.time-slots {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
  align-items: center;
}

.time-tag {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

[data-theme="light"] .time-tag {
  background: #f5f5f5;
  color: #000000;
  border: 1px solid #e0e0e0;
}

[data-theme="dark"] .time-tag {
  background: #2a2a2a;
  color: #ffffff;
  border: 1px solid #404040;
}

.form-actions {
  margin-top: var(--spacing-lg);
  padding-top: var(--spacing-lg);
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: var(--spacing-md);
}

.platform-stats {
  margin-top: var(--spacing-xl);
  padding-top: var(--spacing-lg);
  border-top: 1px solid var(--border-color);
}

.stats-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-md) 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--spacing-md);
}

.stat-item {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}

.stat-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: var(--spacing-xs);
}

.stat-count {
  font-size: 20px;
  font-weight: 600;
  color: var(--primary-color);
  margin-bottom: var(--spacing-xs);
}

.stat-dir {
  font-size: 11px;
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.about-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xl);
}

.about-logo {
  text-align: center;
  padding: var(--spacing-xl);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
}

.about-logo h4 {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin: var(--spacing-md) 0 var(--spacing-xs) 0;
}

.about-logo p {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

:deep(.el-descriptions__label) {
  background: var(--bg-secondary);
  color: var(--text-primary) !important;
  font-weight: 500;
}

:deep(.el-descriptions__content) {
  background: var(--bg-card);
  color: var(--text-primary) !important;
}

.theme-setting {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
}

.setting-label {
  font-size: 14px;
  color: var(--text-secondary);
}
</style>
