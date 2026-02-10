import { GetLogs, SetLogDedupEnabled, IsLogDedupEnabled, GetLogPlatforms } from '../../wailsjs/go/app/App'
import type { SimpleLog, LogQuery } from '../types'

// 获取日志列表
export async function getLogs(query: LogQuery = {}): Promise<SimpleLog[]> {
  try {
    const result = await GetLogs({
      keyword: query.keyword || '',
      limit: query.limit || 100,
      platform: query.platform || '',
      level: query.level || ''
    })
    return result || []
  } catch (error) {
    console.error('获取日志失败:', error)
    return []
  }
}

// 设置日志归并开关
export async function setLogDedupEnabled(enabled: boolean): Promise<void> {
  try {
    await SetLogDedupEnabled(enabled)
  } catch (error) {
    console.error('设置日志归并开关失败:', error)
  }
}

// 获取日志归并状态
export async function isLogDedupEnabled(): Promise<boolean> {
  try {
    return await IsLogDedupEnabled()
  } catch (error) {
    console.error('获取日志归并状态失败:', error)
    return true // 默认启用
  }
}

// 获取有日志的平台列表
export async function getLogPlatforms(): Promise<string[]> {
  try {
    return await GetLogPlatforms()
  } catch (error) {
    console.error('获取日志平台列表失败:', error)
    return []
  }
}
