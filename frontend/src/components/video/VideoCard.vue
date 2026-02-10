<template>
  <div class="video-card">
    <div class="video-thumbnail" @click="$emit('preview', video.id)">
      <el-image
        :src="video.thumbnail || defaultThumbnail"
        :alt="video.title || video.filename"
        fit="cover"
        class="thumbnail-image"
        :preview-src-list="video.thumbnail ? [video.thumbnail] : []"
        :preview-teleported="true"
        hide-on-click-modal
      >
        <template #error>
          <div class="thumbnail-fallback">
            <el-icon :size="40"><VideoCamera /></el-icon>
          </div>
        </template>
        <template #placeholder>
          <div class="thumbnail-loading">
            <el-icon :size="24" class="is-loading"><Loading /></el-icon>
          </div>
        </template>
      </el-image>
      <div class="video-duration" v-if="video.duration">
        {{ formatDuration(video.duration) }}
      </div>
      <div class="video-overlay">
        <el-icon><VideoPlay /></el-icon>
      </div>
    </div>

    <div class="video-info">
      <h4 class="video-title">{{ video.title || video.filename }}</h4>
      <div class="video-meta">
        <span class="video-size">{{ formatFileSize(video.fileSize) }}</span>
        <span class="video-resolution" v-if="video.width && video.height">
          {{ video.width }}x{{ video.height }}
        </span>
      </div>
      <div class="video-tags" v-if="video.tags && video.tags.length > 0">
        <el-tag
          v-for="tag in video.tags.slice(0, 3)"
          :key="tag"
          size="small"
          effect="plain"
        >
          {{ tag }}
        </el-tag>
        <span v-if="video.tags.length > 3" class="more-tags">
          +{{ video.tags.length - 3 }}
        </span>
      </div>
    </div>

    <div class="video-actions">
      <el-button type="primary" size="small" @click="$emit('edit', video.id)">
        <el-icon><Edit /></el-icon>
      </el-button>
      <el-button type="success" size="small" @click="$emit('publish', video.id)">
        <el-icon><Upload /></el-icon>
      </el-button>
      <el-button type="danger" size="small" @click="$emit('delete', video.id)">
        <el-icon><Delete /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Video } from '../../types'
import { formatDuration, formatFileSize } from '../../utils/format'

const props = defineProps<{
  video: Video
}>()

defineEmits<{
  preview: [id: number]
  edit: [id: number]
  publish: [id: number]
  delete: [id: number]
}>()

const defaultThumbnail = '/images/default-video-thumb.png'
</script>

<style scoped>
.video-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-card);
  transition: all var(--transition-slow);
  border: 1px solid var(--border-color);
}

.video-card:hover {
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-2px);
}

.video-thumbnail {
  position: relative;
  width: 100%;
  aspect-ratio: 16/9;
  overflow: hidden;
  cursor: pointer;
  background: var(--bg-secondary);
}

.thumbnail-image {
  width: 100%;
  height: 100%;
}

.thumbnail-fallback,
.thumbnail-loading {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  color: var(--text-tertiary);
}

.video-duration {
  position: absolute;
  bottom: 8px;
  right: 8px;
  background: var(--bg-overlay);
  color: var(--text-primary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.video-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-overlay);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity var(--transition-base);
}

.video-thumbnail:hover .video-overlay {
  opacity: 1;
}

.video-overlay .el-icon {
  font-size: 48px;
  color: var(--text-primary);
}

.video-info {
  padding: 12px;
}

.video-title {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.video-meta {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--text-tertiary);
}

.video-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.more-tags {
  font-size: 12px;
  color: var(--text-tertiary);
}

.video-actions {
  padding: 0 12px 12px;
  display: flex;
  gap: 8px;
}
</style>
