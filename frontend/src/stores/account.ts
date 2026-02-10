import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Account, PlatformType } from '../types'
import * as accountApi from '../api/account'

export const useAccountStore = defineStore('account', () => {
  // State
  const accounts = ref<Account[]>([])
  const loading = ref(false)
  const currentAccount = ref<Account | null>(null)

  // Getters
  const accountList = computed(() => accounts.value)
  const isLoading = computed(() => loading.value)
  
  const accountsByPlatform = computed(() => {
    const grouped: Record<string, Account[]> = {}
    accounts.value.forEach(account => {
      if (!grouped[account.platform]) {
        grouped[account.platform] = []
      }
      grouped[account.platform].push(account)
    })
    return grouped
  })

  const validAccounts = computed(() => 
    accounts.value.filter(a => a.status === 1)
  )

  // Actions
  async function fetchAccounts() {
    loading.value = true
    try {
      accounts.value = await accountApi.getAccounts()
    } finally {
      loading.value = false
    }
  }

  async function addAccount(platform: PlatformType, name: string) {
    loading.value = true
    try {
      const account = await accountApi.addAccount(platform, name)
      accounts.value.push(account)
      return account
    } finally {
      loading.value = false
    }
  }

  async function deleteAccount(id: number) {
    loading.value = true
    try {
      await accountApi.deleteAccount(id)
      accounts.value = accounts.value.filter(a => a.id !== id)
    } finally {
      loading.value = false
    }
  }

  async function validateAccount(id: number) {
    loading.value = true
    try {
      const valid = await accountApi.validateAccount(id)
      const account = accounts.value.find(a => a.id === id)
      if (account) {
        account.status = valid ? 1 : 0
      }
      return valid
    } finally {
      loading.value = false
    }
  }

  async function loginAccount(id: number) {
    loading.value = true
    try {
      await accountApi.loginAccount(id)
    } finally {
      loading.value = false
    }
  }

  async function reloginAccount(id: number) {
    loading.value = true
    try {
      await accountApi.reloginAccount(id)
    } finally {
      loading.value = false
    }
  }

  async function updateAccount(account: Account) {
    loading.value = true
    try {
      await accountApi.updateAccount(account)
      const index = accounts.value.findIndex(a => a.id === account.id)
      if (index !== -1) {
        accounts.value[index] = account
      }
    } finally {
      loading.value = false
    }
  }

  function updateAccountStatus(id: number, status: number) {
    const account = accounts.value.find(a => a.id === id)
    if (account) {
      account.status = status as 0 | 1 | 2
    }
  }

  return {
    accounts,
    loading,
    currentAccount,
    accountList,
    isLoading,
    accountsByPlatform,
    validAccounts,
    fetchAccounts,
    addAccount,
    deleteAccount,
    validateAccount,
    loginAccount,
    reloginAccount,
    updateAccount,
    updateAccountStatus
  }
})
