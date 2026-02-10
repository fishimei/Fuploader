<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useAccountStore, useVideoStore, useTaskStore, useScheduleStore } from '../stores'
import { PLATFORM_CONFIG, PLATFORM_PUBLISH_FIELDS, PLATFORM_TAGS_LIMIT } from '../types'
import type { PlatformType, PlatformField, UploadCompleteEvent, UploadErrorEvent } from '../types'
import { EventsOn, EventsOff } from '../../wailsjs/runtime'

const router = useRouter()
const accountStore = useAccountStore()
const videoStore = useVideoStore()
const taskStore = useTaskStore()
const scheduleStore = useScheduleStore()

const selectedVideo = ref<number | null>(null)
const selectedAccounts = ref<number[]>([])
const publishMode = ref<'immediate' | 'scheduled'>('immediate')
const scheduledTime = ref<string>('')

// 通用表单数据
const commonForm = ref({
  title: '',
  description: ''
})

// 平台特定表单数据
const platformForms = ref<Record<string, Record<string, any>>>({})

const availableAccounts = computed(() => {
  return accountStore.validAccounts
})

// 获取已选择的平台列表
const selectedPlatforms = computed(() => {
  const platforms = new Set<string>()
  selectedAccounts.value.forEach(accountId => {
    const account = accountStore.accounts.find(a => a.id === accountId)
    if (account) {
      platforms.add(account.platform)
    }
  })
  return Array.from(platforms) as PlatformType[]
})

// 获取已选择平台的名称
const selectedPlatformNames = computed(() => {
  return selectedPlatforms.value.map(p => PLATFORM_CONFIG[p]?.name || p).join('、')
})

const canPublish = computed(() => {
  return selectedVideo.value && selectedAccounts.value.length > 0 &&
    (publishMode.value === 'immediate' || scheduledTime.value)
})

// 获取当前选中视频的标签
const selectedVideoTags = computed(() => {
  if (!selectedVideo.value) return []
  const video = videoStore.videos.find(v => v.id === selectedVideo.value)
  return video?.tags || []
})

// 获取当前选中视频的标题
const selectedVideoTitle = computed(() => {
  if (!selectedVideo.value) return ''
  const video = videoStore.videos.find(v => v.id === selectedVideo.value)
  return video?.title || ''
})

// 获取当前选中视频的描述
const selectedVideoDescription = computed(() => {
  if (!selectedVideo.value) return ''
  const video = videoStore.videos.find(v => v.id === selectedVideo.value)
  return video?.description || ''
})

// 获取当前选中视频的封面
const selectedVideoThumbnail = computed(() => {
  if (!selectedVideo.value) return ''
  const video = videoStore.videos.find(v => v.id === selectedVideo.value)
  return video?.thumbnail || ''
})

// 同步视频标签到平台表单
function syncVideoTagsToPlatform(platform: PlatformType) {
  const fields = getPlatformFields(platform)
  const hasTagsField = fields.some(f => f.key === 'tags')

  if (!hasTagsField) return

  const tags = selectedVideoTags.value
  if (tags.length === 0) return

  // 获取平台标签限制
  const limit = PLATFORM_TAGS_LIMIT[platform]

  // 如果有限制，截断标签
  const syncTags = limit ? tags.slice(0, limit) : [...tags]

  // 设置到平台表单
  if (!platformForms.value[platform]) {
    platformForms.value[platform] = {}
  }

  // 只有当标签字段为空或未被用户修改时才同步
  const currentTags = platformForms.value[platform]?.tags
  if (!currentTags || currentTags.length === 0) {
    platformForms.value[platform].tags = syncTags
  }
}

// 同步视频封面到平台表单
function syncVideoThumbnailToPlatform(platform: PlatformType) {
  const fields = getPlatformFields(platform)
  const hasThumbnailField = fields.some(f => f.key === 'thumbnail' && f.type === 'image')

  if (!hasThumbnailField) return

  const thumbnail = selectedVideoThumbnail.value
  if (!thumbnail) return

  // 设置到平台表单
  if (!platformForms.value[platform]) {
    platformForms.value[platform] = {}
  }

  // 只有当封面字段为空时才同步（避免覆盖用户手动设置的封面）
  const currentThumbnail = platformForms.value[platform]?.thumbnail
  if (!currentThumbnail) {
    platformForms.value[platform].thumbnail = thumbnail
  }
}

const groupedAccounts = computed(() => {
  const grouped: Record<string, typeof accountStore.accounts> = {}
  availableAccounts.value.forEach(account => {
    if (!grouped[account.platform]) {
      grouped[account.platform] = []
    }
    grouped[account.platform].push(account)
  })
  return grouped
})

function getPlatformColor(platform: string): string {
  return PLATFORM_CONFIG[platform as PlatformType]?.color || 'var(--primary-color)'
}

function getPlatformName(platform: string): string {
  return PLATFORM_CONFIG[platform as PlatformType]?.name || platform
}

function getPlatformIcon(platform: string): string {
  return PLATFORM_CONFIG[platform as PlatformType]?.icon || 'Platform'
}

function getPlatformFields(platform: PlatformType): PlatformField[] {
  return PLATFORM_PUBLISH_FIELDS[platform] || []
}

// 初始化平台表单数据
function initializePlatformForm(platform: PlatformType) {
  if (!platformForms.value[platform]) {
    const fields = getPlatformFields(platform)
    const initialData: Record<string, any> = {}
    fields.forEach(field => {
      initialData[field.key] = field.defaultValue !== undefined ? field.defaultValue : ''
    })
    platformForms.value[platform] = initialData
  }
}

// 监听选择的账号变化，自动初始化对应平台的数据
watch(selectedPlatforms, (platforms) => {
  platforms.forEach(platform => {
    initializePlatformForm(platform)
  })
}, { immediate: true })

// 监听选中的视频变化，自动同步标题、描述、标签和封面到各平台
watch(selectedVideo, (newVideoId) => {
  if (!newVideoId) return

  // 延迟执行，确保平台表单已初始化
  nextTick(() => {
    // 同步标题（如果通用标题为空）
    if (!commonForm.value.title && selectedVideoTitle.value) {
      commonForm.value.title = selectedVideoTitle.value
    }

    // 同步描述（如果通用描述为空）
    if (!commonForm.value.description && selectedVideoDescription.value) {
      commonForm.value.description = selectedVideoDescription.value
    }

    // 同步标签到各平台
    selectedPlatforms.value.forEach(platform => {
      syncVideoTagsToPlatform(platform)
    })

    // 同步封面到各平台
    selectedPlatforms.value.forEach(platform => {
      syncVideoThumbnailToPlatform(platform)
    })
  })
})

// 监听平台选择变化，同步标签和封面到新选择的平台
watch(selectedPlatforms, (platforms, oldPlatforms) => {
  const oldPlatformsList = oldPlatforms || []
  const newPlatforms = platforms.filter(p => !oldPlatformsList.includes(p))
  newPlatforms.forEach(platform => {
    syncVideoTagsToPlatform(platform)
    syncVideoThumbnailToPlatform(platform)
  })
})

const createdTaskIds = ref<number[]>([])
const publishResults = ref<Map<number, { success: boolean; message: string }>>(new Map())

async function handlePublish() {
  if (!canPublish.value) return

  try {
    const scheduleTime = publishMode.value === 'scheduled' ? scheduledTime.value : null

    // 构建平台特定的数据
    const platformData = selectedPlatforms.value.map(platform => ({
      platform,
      accounts: selectedAccounts.value.filter(id => {
        const account = accountStore.accounts.find(a => a.id === id)
        return account?.platform === platform
      }),
      fields: platformForms.value[platform] || {}
    }))

    const newTasks = await taskStore.createTask({
      videoId: selectedVideo.value!,
      platformData,
      commonData: commonForm.value,
      scheduleTime
    })

    // 记录创建的任务ID
    createdTaskIds.value = newTasks.map(t => t.id)

    if (publishMode.value === 'immediate') {
      // 立即发布模式：监听发布结果
      ElMessage.info('发布任务已创建，正在执行发布...')

      // 设置发布结果监听
      setupPublishListeners()

      // 延迟跳转到任务页面，让用户看到发布进度
      setTimeout(() => {
        router.push('/tasks')
      }, 2000)
    } else {
      // 定时发布模式：直接跳转
      ElMessage.success('定时任务已创建')
      router.push('/tasks')
    }
  } catch (error: any) {
    const errorMsg = error?.message || '创建任务失败'
    ElMessage.error(`创建任务失败: ${errorMsg}`)
  }
}

// 设置发布结果监听
function setupPublishListeners() {
  // 监听发布完成事件
  EventsOn('upload:complete', (event: UploadCompleteEvent) => {
    if (createdTaskIds.value.includes(event.taskId)) {
      publishResults.value.set(event.taskId, {
        success: true,
        message: `发布成功: ${event.platform}`
      })
      ElMessage.success(`${PLATFORM_CONFIG[event.platform as PlatformType]?.name || event.platform} 发布成功`)
    }
  })

  // 监听发布错误事件
  EventsOn('upload:error', (event: UploadErrorEvent) => {
    if (createdTaskIds.value.includes(event.taskId)) {
      publishResults.value.set(event.taskId, {
        success: false,
        message: `发布失败: ${event.platform} - ${event.error}`
      })
      ElMessage.error(`${PLATFORM_CONFIG[event.platform as PlatformType]?.name || event.platform} 发布失败: ${event.error}`)
    }
  })
}

// 清理事件监听
onUnmounted(() => {
  EventsOff('upload:complete')
  EventsOff('upload:error')
})

function toggleAccount(accountId: number) {
  const index = selectedAccounts.value.indexOf(accountId)
  if (index > -1) {
    selectedAccounts.value.splice(index, 1)
  } else {
    selectedAccounts.value.push(accountId)
  }
}

// ============================================
// 字段显示控制 - 根据条件判断是否显示字段
// ============================================
function shouldShowField(field: PlatformField, formData: Record<string, any>): boolean {
  if (!field.showWhen) return true
  return field.showWhen(formData)
}

// ============================================
// 合集选择相关
// ============================================
const loadingCollections = ref<Record<string, boolean>>({})
const collectionOptions = ref<Record<string, { label: string; value: string }[]>>({})

async function loadCollections(platform: string, field: PlatformField) {
  // 如果已经有数据，不再重复加载
  if (collectionOptions.value[platform]?.length > 0) return

  loadingCollections.value[platform] = true
  try {
    // 调用 taskStore 获取合集列表
    const options = await taskStore.getCollections(platform)
    collectionOptions.value[platform] = options
  } catch (error) {
    console.error('加载合集列表失败:', error)
    ElMessage.error('加载合集列表失败')
  } finally {
    loadingCollections.value[platform] = false
  }
}

// ============================================
// 自动生成短标题 - 监听标题变化
// ============================================
watch(() => commonForm.value.title, (newTitle) => {
  if (!newTitle) return
  
  selectedPlatforms.value.forEach(platform => {
    const fields = getPlatformFields(platform)
    const shortTitleField = fields.find(f => f.key === 'shortTitle' && f.autoGenerate)
    if (shortTitleField && !platformForms.value[platform]?.shortTitle) {
      // 自动生成短标题：6-16字符
      let shortTitle = newTitle.slice(0, 16)
      if (shortTitle.length < 6) {
        shortTitle = shortTitle.padEnd(6, ' ')
      }
      if (!platformForms.value[platform]) {
        platformForms.value[platform] = {}
      }
      platformForms.value[platform].shortTitle = shortTitle
    }
  })
})

// ============================================
// 图片选择处理
// ============================================
async function handleSelectImage(platform: string, fieldKey: string) {
  if (!selectedVideo.value) {
    ElMessage.warning('请先选择视频')
    return
  }

  try {
    // 调用后端API选择图片文件并保存到存储目录
    const result = await taskStore.selectImageFile(selectedVideo.value)
    if (result) {
      if (!platformForms.value[platform]) {
        platformForms.value[platform] = {}
      }
      platformForms.value[platform][fieldKey] = result
      ElMessage.success('封面上传成功')
    }
  } catch (error) {
    console.error('选择图片失败:', error)
    ElMessage.error('封面上传失败')
  }
}

// ============================================
// 自动选择封面
// ============================================
async function handleAutoSelectCover(platform: string, fieldKey: string) {
  if (!selectedVideo.value) {
    ElMessage.warning('请先选择视频')
    return
  }
  
  try {
    // 调用后端API自动选择推荐封面
    const result = await taskStore.autoSelectCover(selectedVideo.value)
    if (result && result.thumbnailPath) {
      if (!platformForms.value[platform]) {
        platformForms.value[platform] = {}
      }
      platformForms.value[platform][fieldKey] = result.thumbnailPath
      ElMessage.success('已选择推荐封面')
    }
  } catch (error) {
    console.error('自动选择封面失败:', error)
    ElMessage.error('自动选择封面失败')
  }
}

// ============================================
// 文件选择处理
// ============================================
async function handleSelectFile(platform: string, fieldKey: string, accept?: string) {
  try {
    // 调用后端API选择文件
    const result = await taskStore.selectFile(accept)
    if (result) {
      if (!platformForms.value[platform]) {
        platformForms.value[platform] = {}
      }
      platformForms.value[platform][fieldKey] = result
    }
  } catch (error) {
    console.error('选择文件失败:', error)
  }
}

// ============================================
// 验证商品链接
// ============================================
async function validateProductLink(platform: string, fieldKey: string) {
  const link = platformForms.value[platform]?.[fieldKey]
  if (!link) {
    ElMessage.warning('请输入商品链接')
    return
  }
  
  try {
    // 调用后端API验证链接
    const result = await taskStore.validateProductLink(link)
    if (result.valid) {
      ElMessage.success('链接验证成功')
      // 自动填充商品标题
      if (result.title && !platformForms.value[platform]?.productTitle) {
        if (!platformForms.value[platform]) {
          platformForms.value[platform] = {}
        }
        platformForms.value[platform].productTitle = result.title
      }
    } else {
      ElMessage.error('链接验证失败：' + (result.error || '未知错误'))
    }
  } catch (error) {
    console.error('验证商品链接失败:', error)
    ElMessage.error('验证失败')
  }
}

onMounted(() => {
  accountStore.fetchAccounts()
  videoStore.fetchVideos()
  scheduleStore.fetchConfig()
})
</script>

<template>
  <div class="publish-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">发布中心</h2>
        <p class="page-subtitle">创建新的视频发布任务</p>
      </div>
    </div>

    <div class="publish-container">
      <!-- 选择视频 -->
      <div class="section">
        <h3 class="section-title">
          <el-icon><VideoCamera /></el-icon>
          选择视频
        </h3>
        <div class="video-grid">
          <div
            v-for="video in videoStore.videos"
            :key="video.id"
            class="video-item"
            :class="{ selected: selectedVideo === video.id }"
            @click="selectedVideo = selectedVideo === video.id ? null : video.id"
          >
            <div class="video-thumb">
              <img v-if="video.thumbnail" :src="video.thumbnail" :alt="video.filename">
              <div v-else class="video-placeholder">
                <el-icon><VideoCamera /></el-icon>
              </div>
            </div>
            <div class="video-name">{{ video.title || video.filename }}</div>
          </div>
        </div>
        <el-empty v-if="videoStore.videos.length === 0" description="暂无视频，请先添加视频">
          <el-button type="primary" @click="$router.push('/videos')">去添加视频</el-button>
        </el-empty>
      </div>

      <!-- 选择账号 -->
      <div class="section">
        <h3 class="section-title">
          <el-icon><User /></el-icon>
          选择账号
          <span class="selection-hint">已选择 {{ selectedAccounts.length }} 个账号</span>
        </h3>

        <div class="account-groups">
          <div
            v-for="(accounts, platform) in groupedAccounts"
            :key="platform"
            class="account-group"
          >
            <div class="group-header" :style="{ color: getPlatformColor(platform) }">
              <el-icon :size="16">
                <component :is="getPlatformIcon(platform)" />
              </el-icon>
              <span>{{ getPlatformName(platform) }}</span>
            </div>
            <div class="account-list">
              <div
                v-for="account in accounts"
                :key="account.id"
                class="account-option"
                :class="{ selected: selectedAccounts.includes(account.id) }"
                @click="toggleAccount(account.id)"
              >
                <el-checkbox :model-value="selectedAccounts.includes(account.id)">
                  {{ account.name }}
                </el-checkbox>
              </div>
            </div>
          </div>
        </div>

        <el-empty v-if="availableAccounts.length === 0" description="没有有效的账号，请先添加并登录账号">
          <el-button type="primary" @click="$router.push('/accounts')">去添加账号</el-button>
        </el-empty>
      </div>

      <!-- 发布内容 -->
      <div v-if="selectedPlatforms.length > 0" class="section">
        <h3 class="section-title">
          <el-icon><Edit /></el-icon>
          发布内容
          <span class="platform-hint">已选平台: {{ selectedPlatformNames }}</span>
        </h3>

        <!-- 通用字段 -->
        <div class="common-fields">
          <el-form label-position="top">
            <el-form-item label="标题" required>
              <el-input
                v-model="commonForm.title"
                placeholder="请输入视频标题"
                maxlength="100"
                show-word-limit
              />
            </el-form-item>
            <el-form-item label="内容说明">
              <el-input
                v-model="commonForm.description"
                type="textarea"
                :rows="4"
                placeholder="请输入视频内容说明"
                maxlength="500"
                show-word-limit
              />
            </el-form-item>
          </el-form>
        </div>

        <!-- 平台特定字段 -->
        <div class="platform-fields-container">
          <div
            v-for="platform in selectedPlatforms"
            :key="platform"
            class="platform-fields"
          >
            <div class="platform-header" :style="{ borderLeftColor: getPlatformColor(platform) }">
              <el-icon :size="18" :style="{ color: getPlatformColor(platform) }">
                <component :is="getPlatformIcon(platform)" />
              </el-icon>
              <span class="platform-name">{{ getPlatformName(platform) }}</span>
            </div>
            <el-form label-position="top" class="platform-form">
              <el-form-item
                v-for="field in getPlatformFields(platform)"
                :key="field.key"
                :label="field.label"
                :required="field.required"
                v-show="!field.internal && shouldShowField(field, platformForms[platform])"
              >
                <!-- 输入框 -->
                <el-input
                  v-if="field.type === 'input'"
                  v-model="platformForms[platform][field.key]"
                  :placeholder="field.placeholder"
                  :maxlength="field.maxLength"
                  show-word-limit
                />

                <!-- 文本域 -->
                <el-input
                  v-else-if="field.type === 'textarea'"
                  v-model="platformForms[platform][field.key]"
                  type="textarea"
                  :rows="4"
                  :placeholder="field.placeholder"
                  :maxlength="field.maxLength"
                  show-word-limit
                />

                <!-- 下拉选择 -->
                <el-select
                  v-else-if="field.type === 'select'"
                  v-model="platformForms[platform][field.key]"
                  style="width: 100%"
                >
                  <el-option
                    v-for="option in field.options"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>

                <!-- 开关 -->
                <el-switch
                  v-else-if="field.type === 'switch'"
                  v-model="platformForms[platform][field.key]"
                />

                <!-- 标签输入 -->
                <el-select
                  v-else-if="field.type === 'tags'"
                  v-model="platformForms[platform][field.key]"
                  multiple
                  filterable
                  allow-create
                  default-first-option
                  :placeholder="field.placeholder || '输入标签，回车确认'"
                  style="width: 100%"
                />

                <!-- 数字输入 -->
                <el-input-number
                  v-else-if="field.type === 'number'"
                  v-model="platformForms[platform][field.key]"
                  :min="field.min"
                  :max="field.max"
                  style="width: 100%"
                />

                <!-- 图片上传 -->
                <div v-else-if="field.type === 'image'" class="image-upload">
                  <div class="upload-preview" v-if="platformForms[platform][field.key]">
                    <img :src="platformForms[platform][field.key]" class="preview-img" />
                    <el-button type="danger" size="small" @click="platformForms[platform][field.key] = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <div v-else class="upload-actions">
                    <el-button type="primary" @click="handleSelectImage(platform, field.key)">
                      <el-icon><Picture /></el-icon>
                      选择图片
                    </el-button>
                    <el-button
                      v-if="field.allowAutoSelect"
                      type="info"
                      link
                      :disabled="!selectedVideo"
                      @click="handleAutoSelectCover(platform, field.key)"
                    >
                      {{ field.autoSelectText || '使用推荐封面' }}
                    </el-button>
                  </div>
                </div>

                <!-- 文件上传 -->
                <div v-else-if="field.type === 'file'" class="file-upload">
                  <div class="file-info" v-if="platformForms[platform][field.key]">
                    <span>{{ platformForms[platform][field.key] }}</span>
                    <el-button type="danger" size="small" @click="platformForms[platform][field.key] = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <el-button v-else type="primary" @click="handleSelectFile(platform, field.key, field.accept)">
                    <el-icon><Upload /></el-icon>
                    选择文件
                  </el-button>
                </div>

                <!-- 合集选择 -->
                <div v-else-if="field.type === 'collection'" class="collection-select">
                  <el-select
                    v-model="platformForms[platform][field.key]"
                    :placeholder="field.placeholder"
                    style="width: 100%"
                    :loading="loadingCollections[platform]"
                    @focus="loadCollections(platform, field)"
                  >
                    <el-option
                      v-for="collection in collectionOptions[platform] || []"
                      :key="collection.value"
                      :label="collection.label"
                      :value="collection.value"
                    />
                  </el-select>
                </div>

                <!-- 商品链接 -->
                <div v-else-if="field.type === 'product'" class="product-input">
                  <el-input
                    v-model="platformForms[platform][field.key]"
                    :placeholder="field.placeholder"
                  >
                    <template #append>
                      <el-button @click="validateProductLink(platform, field.key)">
                        验证
                      </el-button>
                    </template>
                  </el-input>
                </div>

                <!-- 日期时间选择 -->
                <div v-else-if="field.type === 'datetime'" class="datetime-picker">
                  <el-date-picker
                    v-model="platformForms[platform][field.key]"
                    type="datetime"
                    :placeholder="field.placeholder"
                    format="YYYY-MM-DD HH:mm"
                    value-format="YYYY-MM-DDTHH:mm:ss"
                    :disabled-date="(time: Date) => time.getTime() < Date.now()"
                    style="width: 100%"
                  />
                  <div class="field-description" v-if="field.description">
                    <el-text type="info" size="small">{{ field.description }}</el-text>
                  </div>
                </div>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>

      <!-- 发布设置 -->
      <div class="section">
        <h3 class="section-title">
          <el-icon><Setting /></el-icon>
          发布设置
        </h3>

        <div class="publish-settings">
          <el-radio-group v-model="publishMode">
            <el-radio-button label="immediate">立即发布</el-radio-button>
            <el-radio-button label="scheduled">定时发布</el-radio-button>
          </el-radio-group>

          <div v-if="publishMode === 'scheduled'" class="schedule-picker">
            <el-date-picker
              v-model="scheduledTime"
              type="datetime"
              placeholder="选择发布时间"
              format="YYYY-MM-DD HH:mm"
              value-format="YYYY-MM-DDTHH:mm:ss"
              :disabled-date="(time: Date) => time.getTime() < Date.now()"
            />
          </div>
        </div>
      </div>

      <!-- 发布按钮 -->
      <div class="publish-actions">
        <el-button
          type="primary"
          size="large"
          :disabled="!canPublish"
          :loading="taskStore.loading"
          @click="handlePublish"
        >
          <el-icon><Upload /></el-icon>
          {{ publishMode === 'immediate' ? '立即发布' : '创建定时任务' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.publish-page {
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

.publish-container {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xl);
}

.section {
  background: var(--bg-secondary);
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

.selection-hint {
  margin-left: auto;
  font-size: 13px;
  font-weight: normal;
  color: var(--text-secondary);
}

.platform-hint {
  margin-left: auto;
  font-size: 13px;
  font-weight: normal;
  color: var(--text-tertiary);
  background: var(--bg-tertiary);
  padding: var(--spacing-1) var(--spacing-3);
  border-radius: var(--radius-md);
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: var(--spacing-md);
}

.video-item {
  cursor: pointer;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 2px solid transparent;
  transition: all var(--transition-fast);
}

.video-item:hover {
  border-color: var(--primary-light);
}

.video-item.selected {
  border-color: var(--primary-color);
  box-shadow: var(--shadow-glow);
}

.video-thumb {
  aspect-ratio: 16/9;
  background: var(--bg-secondary);
  overflow: hidden;
}

.video-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.video-name {
  padding: var(--spacing-sm);
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  background: var(--bg-secondary);
}

.account-groups {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.account-group {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}

.group-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-weight: 600;
  margin-bottom: var(--spacing-md);
}

.account-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--spacing-sm);
}

.account-option {
  padding: var(--spacing-sm);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.account-option:hover {
  background: var(--selected-bg);
}

.common-fields {
  margin-bottom: var(--spacing-xl);
  padding-bottom: var(--spacing-xl);
  border-bottom: 1px solid var(--border-color);
}

.platform-fields-container {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.platform-fields {
  background: var(--bg-primary);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.platform-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-md) var(--spacing-lg);
  background: var(--bg-secondary);
  border-left: 4px solid;
  border-bottom: 1px solid var(--border-color);
}

.platform-name {
  font-weight: 600;
  color: var(--text-primary);
}

.platform-form {
  padding: var(--spacing-lg);
}

.publish-settings {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.schedule-picker {
  margin-top: var(--spacing-md);
}

.publish-actions {
  display: flex;
  justify-content: center;
  padding: var(--spacing-lg);
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
}

.publish-actions .el-button {
  min-width: 200px;
}

/* ============================================
   新字段类型样式
   ============================================ */

/* 图片上传 */
.image-upload {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.upload-preview {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

.preview-img {
  width: 120px;
  height: 80px;
  object-fit: cover;
  border-radius: var(--radius-sm);
}

.upload-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

/* 文件上传 */
.file-upload {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.file-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

.file-info span {
  color: var(--text-secondary);
  font-size: 14px;
}

/* 合集选择 */
.collection-select {
  width: 100%;
}

/* 商品链接 */
.product-input {
  width: 100%;
}

/* 日期时间选择 */
.datetime-picker {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  width: 100%;
}

.field-description {
  margin-top: var(--spacing-xs);
}

/* Override Element Plus input focus styles within this component */
:deep(.el-input__wrapper.is-focus),
:deep(.el-input__wrapper:focus),
:deep(.el-textarea__inner:focus),
:deep(.el-select .el-input__wrapper.is-focus),
:deep(.el-select .el-input.is-focus .el-input__wrapper) {
  box-shadow: none !important;
  border-color: var(--border-color) !important;
  outline: none !important;
}
</style>
