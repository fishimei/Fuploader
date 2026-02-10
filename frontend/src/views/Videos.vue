<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useVideoStore } from '../stores'
import { formatFileSize, formatDuration, formatDateTime, truncateText } from '../utils/format'
import { extractVideoFrame, uploadThumbnail, clearThumbnail, selectImageFile } from '../api/video'
import type { Video } from '../types'

const videoStore = useVideoStore()

const showEditDialog = ref(false)
const editingVideo = ref<Video | null>(null)
const editForm = ref({
  title: '',
  description: '',
  tags: [] as string[],
  thumbnail: ''
})
const tagInput = ref('')
const extractingFrame = ref(false)
const uploadingThumbnail = ref(false)

const showTimeSlider = ref(false)
const frameTime = ref(0)
const maxFrameTime = computed(() => {
  return editingVideo.value?.duration ? Math.floor(editingVideo.value.duration) : 0
})

async function handleSelectVideo() {
  try {
    const video = await videoStore.selectAndAddVideo()
    if (video) {
      ElMessage.success('视频添加成功')
    }
  } catch (error) {
    ElMessage.error('添加视频失败')
  }
}

async function handleDeleteVideo(video: Video) {
  try {
    await ElMessageBox.confirm(
      `确定要删除视频 "${video.filename}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await videoStore.deleteVideo(video.id)
    ElMessage.success('视频已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function openEditDialog(video: Video) {
  editingVideo.value = video
  editForm.value = {
    title: video.title || '',
    description: video.description || '',
    tags: video.tags || [],
    thumbnail: video.thumbnail || ''
  }
  frameTime.value = 0
  showTimeSlider.value = false
  showEditDialog.value = true
}

async function handleSaveEdit() {
  if (!editingVideo.value) return

  try {
    const updatedVideo = {
      ...editingVideo.value,
      title: editForm.value.title,
      description: editForm.value.description,
      tags: editForm.value.tags,
      thumbnail: editForm.value.thumbnail
    }
    await videoStore.updateVideo(updatedVideo)
    ElMessage.success('保存成功')
    showEditDialog.value = false
  } catch (error) {
    ElMessage.error('保存失败')
  }
}

function handleAddTag() {
  const tag = tagInput.value.trim()
  if (tag && !editForm.value.tags.includes(tag)) {
    editForm.value.tags.push(tag)
  }
  tagInput.value = ''
}

function handleRemoveTag(index: number) {
  editForm.value.tags.splice(index, 1)
}

function toggleTimeSlider() {
  showTimeSlider.value = !showTimeSlider.value
}

async function handleExtractFrame() {
  if (!editingVideo.value) return

  extractingFrame.value = true
  try {
    const result = await extractVideoFrame(editingVideo.value.id, frameTime.value)
    editForm.value.thumbnail = result.thumbnailPath
    ElMessage.success('封面提取成功')
    showTimeSlider.value = false
  } catch (error: any) {
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error(`封面提取失败: ${errorMsg}`)
    console.error('抽帧错误详情:', error)
  } finally {
    extractingFrame.value = false
  }
}

async function handleUploadThumbnail() {
  if (!editingVideo.value) return

  try {
    const filePath = await selectImageFile()
    if (!filePath) return

    uploadingThumbnail.value = true
    const result = await uploadThumbnail(editingVideo.value.id, filePath)
    editForm.value.thumbnail = result.thumbnailPath
    ElMessage.success('封面上传成功')
  } catch (error) {
    ElMessage.error('封面上传失败')
  } finally {
    uploadingThumbnail.value = false
  }
}

async function handleClearThumbnail() {
  if (!editingVideo.value) return

  try {
    await clearThumbnail(editingVideo.value.id)
    editForm.value.thumbnail = ''
    ElMessage.success('封面已清除')
  } catch (error) {
    ElMessage.error('清除封面失败')
  }
}

function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

onMounted(() => {
  videoStore.fetchVideos()
})
</script>

<template>
  <div class="videos-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">视频管理</h2>
        <p class="page-subtitle">管理您的视频素材</p>
      </div>
      <el-button type="primary" @click="handleSelectVideo">
        <el-icon><Plus /></el-icon>
        添加视频
      </el-button>
    </div>

    <div class="videos-grid" v-if="videoStore.videos.length > 0">
      <div
        v-for="video in videoStore.videos"
        :key="video.id"
        class="video-card"
      >
        <div class="video-thumbnail">
          <img v-if="video.thumbnail" :src="video.thumbnail" :alt="video.filename">
          <div v-else class="thumbnail-placeholder">
            <el-icon :size="48"><VideoCamera /></el-icon>
          </div>
          <div class="video-duration" v-if="video.duration">
            {{ formatDuration(video.duration) }}
          </div>
        </div>

        <div class="video-info">
          <h3 class="video-title">{{ truncateText(video.title || video.filename, 30) }}</h3>
          <p class="video-meta">
            <span>{{ formatFileSize(video.fileSize) }}</span>
            <span v-if="video.width && video.height">{{ video.width }}x{{ video.height }}</span>
          </p>
          <p class="video-date">{{ formatDateTime(video.createdAt) }}</p>
          <div class="video-tags" v-if="video.tags && video.tags.length > 0">
            <el-tag
              v-for="tag in video.tags.slice(0, 3)"
              :key="tag"
              size="small"
              effect="plain"
            >
              {{ tag }}
            </el-tag>
            <el-tag v-if="video.tags.length > 3" size="small" effect="plain">
              +{{ video.tags.length - 3 }}
            </el-tag>
          </div>
        </div>

        <div class="video-actions">
          <el-button type="primary" size="small" @click="openEditDialog(video)">
            <el-icon><Edit /></el-icon>
            编辑
          </el-button>
          <el-button type="danger" size="small" plain @click="handleDeleteVideo(video)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <el-empty
      v-else
      description="暂无视频，点击右上角添加"
      :image-size="120"
    >
      <el-button type="primary" @click="handleSelectVideo">
        添加视频
      </el-button>
    </el-empty>

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑视频信息"
      width="560px"
    >
      <el-form label-position="top">
        <!-- 封面编辑区域 -->
        <el-form-item label="视频封面">
          <div class="cover-edit-section">
            <div class="cover-preview">
              <img
                v-if="editForm.thumbnail"
                :src="editForm.thumbnail"
                class="cover-image"
                alt="视频封面"
              >
              <div v-else class="cover-placeholder">
                <el-icon :size="40"><Picture /></el-icon>
                <span>暂无封面</span>
              </div>
            </div>

            <div class="cover-actions">
              <el-button
                type="primary"
                size="small"
                :loading="extractingFrame"
                @click="toggleTimeSlider"
              >
                <el-icon><VideoPlay /></el-icon>
                从视频提取
              </el-button>
              <el-button
                type="primary"
                size="small"
                plain
                :loading="uploadingThumbnail"
                @click="handleUploadThumbnail"
              >
                <el-icon><Upload /></el-icon>
                上传图片
              </el-button>
              <el-button
                v-if="editForm.thumbnail"
                type="danger"
                size="small"
                plain
                @click="handleClearThumbnail"
              >
                <el-icon><Delete /></el-icon>
                清除封面
              </el-button>
            </div>

            <!-- 时间选择滑块 -->
            <div v-if="showTimeSlider" class="time-slider-section">
              <div class="time-display">
                <span>选择时间点: {{ formatTime(frameTime) }}</span>
              </div>
              <el-slider
                v-model="frameTime"
                :max="maxFrameTime"
                :step="1"
                show-stops
              />
              <div class="slider-actions">
                <el-button size="small" @click="showTimeSlider = false">取消</el-button>
                <el-button
                  type="primary"
                  size="small"
                  :loading="extractingFrame"
                  @click="handleExtractFrame"
                >
                  提取封面
                </el-button>
              </div>
            </div>
          </div>
        </el-form-item>

        <el-form-item label="视频标题">
          <el-input
            v-model="editForm.title"
            placeholder="输入视频标题"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="视频描述">
          <el-input
            v-model="editForm.description"
            type="textarea"
            :rows="3"
            placeholder="输入视频描述"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="标签">
          <div class="tags-input">
            <el-tag
              v-for="(tag, index) in editForm.tags"
              :key="tag"
              closable
              @close="handleRemoveTag(index)"
            >
              {{ tag }}
            </el-tag>
            <el-input
              v-model="tagInput"
              placeholder="输入标签按回车添加"
              style="width: 150px"
              @keyup.enter="handleAddTag"
            />
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.videos-page {
  padding-bottom: var(--spacing-xl);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.videos-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--spacing-lg);
}

.video-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: all var(--transition-fast);
}

.video-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  border-color: var(--primary-color);
}

.video-thumbnail {
  position: relative;
  aspect-ratio: 16/9;
  background: var(--bg-secondary);
  overflow: hidden;
}

.video-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumbnail-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.video-duration {
  position: absolute;
  bottom: var(--spacing-sm);
  right: var(--spacing-sm);
  background: var(--bg-overlay);
  color: var(--text-primary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.video-info {
  padding: var(--spacing-md);
}

.video-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-sm) 0;
}

.video-meta {
  display: flex;
  gap: var(--spacing-md);
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-xs) 0;
}

.video-date {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0 0 var(--spacing-sm) 0;
}

.video-tags {
  display: flex;
  gap: var(--spacing-xs);
  flex-wrap: wrap;
}

.video-actions {
  display: flex;
  gap: var(--spacing-sm);
  padding: 0 var(--spacing-md) var(--spacing-md);
}

.video-actions .el-button {
  flex: 1;
}

.tags-input {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
  align-items: center;
}

/* 封面编辑区域样式 */
.cover-edit-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.cover-preview {
  width: 100%;
  aspect-ratio: 16/9;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.cover-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  gap: var(--spacing-xs);
}

.cover-placeholder span {
  font-size: 12px;
}

.cover-actions {
  display: flex;
  gap: var(--spacing-sm);
  flex-wrap: wrap;
}

.time-slider-section {
  background: var(--bg-secondary);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

.time-display {
  font-size: 14px;
  color: var(--text-primary);
  margin-bottom: var(--spacing-sm);
  font-weight: 500;
}

.slider-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
  margin-top: var(--spacing-sm);
}
</style>
