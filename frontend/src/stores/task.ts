import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UploadTask, TaskStatus, UploadProgressEvent } from '../types'
import * as taskApi from '../api/task'
import * as platformApi from '../api/platform'

export const useTaskStore = defineStore('task', () => {
  // State
  const tasks = ref<UploadTask[]>([])
  const loading = ref(false)
  const progressMap = ref<Map<number, number>>(new Map())
  const messageMap = ref<Map<number, string>>(new Map())

  // Getters
  const taskList = computed(() => tasks.value)
  const isLoading = computed(() => loading.value)
  
  const pendingTasks = computed(() => 
    tasks.value.filter(t => t.status === 'pending')
  )
  
  const runningTasks = computed(() => 
    tasks.value.filter(t => t.status === 'uploading')
  )
  
  const completedTasks = computed(() => 
    tasks.value.filter(t => t.status === 'success')
  )
  
  const failedTasks = computed(() => 
    tasks.value.filter(t => t.status === 'failed')
  )

  // Actions
  async function fetchTasks(status?: TaskStatus) {
    loading.value = true
    try {
      tasks.value = await taskApi.getUploadTasks(status)
    } finally {
      loading.value = false
    }
  }

  async function createTask(params: {
    videoId: number
    platformData: Array<{
      platform: string
      accounts: number[]
      fields: Record<string, any>
    }>
    commonData: {
      title: string
      description: string
    }
    scheduleTime?: string | null
  }) {
    loading.value = true
    try {
      // 提取所有账号ID
      const accountIds = params.platformData.flatMap(p => p.accounts)

      // 构建发布元数据
      const publishMetadata = {
        common: params.commonData,
        platforms: params.platformData.reduce((acc, p) => {
          acc[p.platform] = p.fields
          return acc
        }, {} as Record<string, any>)
      }

      const newTasks = await taskApi.createUploadTask(
        params.videoId,
        accountIds,
        params.scheduleTime,
        publishMetadata
      )
      tasks.value.push(...newTasks)
      return newTasks
    } finally {
      loading.value = false
    }
  }

  async function cancelTask(id: number) {
    loading.value = true
    try {
      await taskApi.cancelUploadTask(id)
      const task = tasks.value.find(t => t.id === id)
      if (task) {
        task.status = 'cancelled'
      }
    } finally {
      loading.value = false
    }
  }

  async function retryTask(id: number) {
    loading.value = true
    try {
      await taskApi.retryUploadTask(id)
      const task = tasks.value.find(t => t.id === id)
      if (task) {
        task.status = 'pending'
        task.errorMsg = undefined
        task.retryCount++
      }
    } finally {
      loading.value = false
    }
  }

  async function deleteTask(id: number) {
    loading.value = true
    try {
      await taskApi.deleteUploadTask(id)
      tasks.value = tasks.value.filter(t => t.id !== id)
    } finally {
      loading.value = false
    }
  }

  // 更新任务进度 (由事件监听调用)
  function updateProgress(event: UploadProgressEvent) {
    console.log('[TaskStore] Update Progress:', event)
    progressMap.value.set(event.taskId, event.progress)
    messageMap.value.set(event.taskId, event.message)
    
    const task = tasks.value.find(t => t.id === event.taskId)
    if (task) {
      console.log('[TaskStore] Found task:', task.id, 'current status:', task.status, 'current progress:', task.progress)
      task.progress = event.progress
      // 确保任务状态为 uploading，否则进度条不会显示
      if (task.status === 'pending') {
        console.log('[TaskStore] Changing status from pending to uploading')
        task.status = 'uploading'
      }
    } else {
      console.warn('[TaskStore] Task not found:', event.taskId)
    }
  }

  // 更新任务状态
  function updateTaskStatus(taskId: number, status: TaskStatus) {
    const task = tasks.value.find(t => t.id === taskId)
    if (task) {
      task.status = status
    }
  }

  // 更新任务发布链接
  function updateTaskPublishUrl(taskId: number, publishUrl: string) {
    const task = tasks.value.find(t => t.id === taskId)
    if (task) {
      task.publishUrl = publishUrl
    }
  }

  // 更新任务错误信息
  function updateTaskError(taskId: number, errorMsg: string) {
    const task = tasks.value.find(t => t.id === taskId)
    if (task) {
      task.errorMsg = errorMsg
      task.status = 'failed'
    }
  }

  function getProgress(taskId: number): number {
    return progressMap.value.get(taskId) || 0
  }

  function getMessage(taskId: number): string {
    return messageMap.value.get(taskId) || ''
  }

  // ============================================
  // 平台特定功能
  // ============================================

  // 获取合集列表
  async function getCollections(platform: string) {
    return await platformApi.getCollections(platform)
  }

  // 自动选择封面
  async function autoSelectCover(videoId: number) {
    return await platformApi.autoSelectCover(videoId)
  }

  // 验证商品链接
  async function validateProductLink(link: string) {
    return await platformApi.validateProductLink(link)
  }

  // 选择图片文件
  async function selectImageFile(videoId?: number) {
    return await platformApi.selectImageFile(videoId)
  }

  // 选择文件
  async function selectFile(accept?: string) {
    return await platformApi.selectFile(accept)
  }

  return {
    tasks,
    loading,
    progressMap,
    messageMap,
    taskList,
    isLoading,
    pendingTasks,
    runningTasks,
    completedTasks,
    failedTasks,
    fetchTasks,
    createTask,
    cancelTask,
    retryTask,
    deleteTask,
    updateProgress,
    updateTaskStatus,
    updateTaskPublishUrl,
    updateTaskError,
    getProgress,
    getMessage,
    getCollections,
    autoSelectCover,
    validateProductLink,
    selectImageFile,
    selectFile
  }
})
