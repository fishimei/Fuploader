// 视频模型
export interface Video {
  id: number
  filename: string
  filePath: string
  fileSize: number
  duration?: number
  width?: number
  height?: number
  title?: string
  description?: string
  tags?: string[]
  thumbnail?: string
  createdAt: string
}

// 视频表单数据 (用于创建/编辑)
export interface VideoFormData {
  title: string
  description: string
  tags: string[]
  thumbnail: string
}

// 封面信息
export interface CoverInfo {
  thumbnailPath: string
}
