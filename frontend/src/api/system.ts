import {
  GetAppVersion,
  OpenDirectory,
  GetHeadlessConfig,
  SetHeadlessConfig
} from '../../wailsjs/go/app/App'
import type { AppVersion } from '../types'

// 获取应用版本
export async function getAppVersion(): Promise<AppVersion> {
  try {
    const version = await GetAppVersion()
    return version as AppVersion
  } catch (error) {
    console.error('获取应用版本失败:', error)
    throw error
  }
}

// 打开目录
export async function openDirectory(path: string): Promise<void> {
  try {
    await OpenDirectory(path)
  } catch (error) {
    console.error('打开目录失败:', error)
    throw error
  }
}

// 获取浏览器无头模式配置
export async function getHeadlessConfig(): Promise<boolean> {
  try {
    const result = await GetHeadlessConfig()
    return result
  } catch (error) {
    console.error('获取无头模式配置失败:', error)
    return false
  }
}

// 设置浏览器无头模式配置
export async function setHeadlessConfig(headless: boolean): Promise<void> {
  try {
    await SetHeadlessConfig(headless)
  } catch (error) {
    console.error('设置无头模式配置失败:', error)
    throw error
  }
}
