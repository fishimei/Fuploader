import {
  GetAccounts,
  AddAccount,
  DeleteAccount,
  ValidateAccount,
  LoginAccount,
  ReloginAccount,
  UpdateAccount
} from '../../wailsjs/go/app/App'
import type { Account } from '../types'

// 获取账号列表
export async function getAccounts(): Promise<Account[]> {
  try {
    const accounts = await GetAccounts()
    return (accounts || []) as Account[]
  } catch (error) {
    console.error('获取账号列表失败:', error)
    throw error
  }
}

// 添加账号
export async function addAccount(platform: string, name: string): Promise<Account> {
  try {
    const account = await AddAccount(platform, name)
    return account as Account
  } catch (error) {
    console.error('添加账号失败:', error)
    throw error
  }
}

// 删除账号
export async function deleteAccount(id: number): Promise<void> {
  try {
    await DeleteAccount(id)
  } catch (error) {
    console.error('删除账号失败:', error)
    throw error
  }
}

// 验证账号
export async function validateAccount(id: number): Promise<boolean> {
  try {
    const valid = await ValidateAccount(id)
    return valid
  } catch (error) {
    console.error('验证账号失败:', error)
    throw error
  }
}

// 登录账号
export async function loginAccount(id: number): Promise<void> {
  try {
    await LoginAccount(id)
  } catch (error) {
    console.error('登录账号失败:', error)
    throw error
  }
}

// 重新登录账号
export async function reloginAccount(id: number): Promise<void> {
  try {
    await ReloginAccount(id)
  } catch (error) {
    console.error('重新登录账号失败:', error)
    throw error
  }
}

// 更新账号
export async function updateAccount(account: Account): Promise<void> {
  try {
    // 转换为 Wails 模型类型（处理可选字段）
    const wailsAccount = {
      ...account,
      username: account.username || '',
      avatar: account.avatar || '',
      cookiePath: account.cookiePath || ''
    }
    await UpdateAccount(wailsAccount as any)
  } catch (error) {
    console.error('更新账号失败:', error)
    throw error
  }
}
