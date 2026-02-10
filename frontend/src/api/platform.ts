// 平台特定 API
import {
  SelectImageFile,
  SelectFile,
  AutoSelectCover,
  UploadThumbnail
} from '../../wailsjs/go/app/App'

// 获取用户合集列表（视频号）
export async function getCollections(platform: string): Promise<{ label: string; value: string }[]> {
  try {
    // TODO: 实现获取合集列表功能
    console.log('获取合集列表:', platform)
    return []
  } catch (error) {
    console.error('获取合集列表失败:', error)
    return []
  }
}

// 自动选择推荐封面（从视频第一帧提取）
export async function autoSelectCover(videoId: number): Promise<{ thumbnailPath: string }> {
  try {
    const result = await AutoSelectCover(videoId)
    return { thumbnailPath: result.thumbnailPath }
  } catch (error) {
    console.error('自动选择封面失败:', error)
    throw error
  }
}

// 选择图片文件并保存为封面
// 先选择文件，然后上传到存储目录，返回可访问的 URL
export async function selectImageFile(videoId?: number): Promise<string> {
  try {
    // 1. 选择图片文件
    const sourcePath = await SelectImageFile()
    if (!sourcePath) {
      return ''
    }

    // 2. 如果有 videoId，保存到存储目录并返回 URL
    if (videoId) {
      const result = await UploadThumbnail(videoId, sourcePath)
      return result.thumbnailPath
    }

    // 3. 没有 videoId，直接返回原始路径（可能无法访问）
    return sourcePath
  } catch (error) {
    console.error('选择图片失败:', error)
    throw error
  }
}

// 验证商品链接（抖音）
export async function validateProductLink(link: string): Promise<{ valid: boolean; title?: string; error?: string }> {
  try {
    // TODO: 实现验证商品链接功能
    console.log('验证商品链接:', link)
    return { valid: false, error: '未实现' }
  } catch (error) {
    console.error('验证商品链接失败:', error)
    throw error
  }
}

// 选择文件
export async function selectFile(accept?: string): Promise<string> {
  try {
    const result = await SelectFile(accept || '')
    return result || ''
  } catch (error) {
    console.error('选择文件失败:', error)
    throw error
  }
}
