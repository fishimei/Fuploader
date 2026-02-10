// 日志类型定义

// 日志级别
export type LogLevel = 'info' | 'warn' | 'error' | 'debug' | 'success'

// 平台类型
export type Platform = 'bilibili' | 'douyin' | 'xiaohongshu' | 'kuaishou' | 'tiktok' | 'baijiahao' | ''

// 简洁日志条目
export interface SimpleLog {
  date: string     // 日期，格式：2006/1/2
  time: string     // 时间，格式：15:04:05
  message: string  // 日志内容
  platform: string // 平台标识
  level: LogLevel  // 日志级别
}

// 日志查询参数
export interface LogQuery {
  keyword?: string   // 关键词搜索
  limit?: number     // 返回条数，默认100
  platform?: string  // 平台筛选
  level?: LogLevel   // 级别筛选
}

// 平台配置
export interface PlatformConfig {
  value: Platform
  label: string
  color: string
}

// 平台列表
export const PLATFORMS: PlatformConfig[] = [
  { value: 'bilibili', label: 'B站', color: '#00A1D6' },
  { value: 'douyin', label: '抖音', color: '#000000' },
  { value: 'xiaohongshu', label: '小红书', color: '#FF2442' },
  { value: 'kuaishou', label: '快手', color: '#FF5000' },
  { value: 'tiktok', label: 'TikTok', color: '#000000' },
  { value: 'baijiahao', label: '百家号', color: '#2932E1' },
]

// 获取平台显示名称
export function getPlatformLabel(platform: string): string {
  const p = PLATFORMS.find(p => p.value === platform)
  return p?.label || platform || '系统'
}

// 获取平台颜色
export function getPlatformColor(platform: string): string {
  const p = PLATFORMS.find(p => p.value === platform)
  return p?.color || '#999'
}
