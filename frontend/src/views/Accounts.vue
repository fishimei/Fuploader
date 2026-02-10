<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAccountStore, useScreenshotStore } from '../stores'
import { PLATFORM_CONFIG } from '../types'
import { ACCOUNT_STATUS_LABELS } from '../utils/constants'
import type { Account, PlatformType, Screenshot } from '../types'
import { SCREENSHOT_TYPE_LABELS } from '../types'

const accountStore = useAccountStore()
const screenshotStore = useScreenshotStore()

const showAddDialog = ref(false)
const adding = ref(false)
const validating = ref<number | null>(null)
const loggingIn = ref<number | null>(null)

// 截图管理相关
const showScreenshotDrawer = ref(false)
const currentAccount = ref<Account | null>(null)
const screenshotFilter = ref({
  platform: '',
  type: '',
  dateRange: [] as Date[]
})

const newAccount = ref({
  platform: '' as PlatformType | '',
  name: ''
})

const platformOptions = Object.entries(PLATFORM_CONFIG).map(([value, config]) => ({
  value: value as PlatformType,
  label: config.name
}))

// 截图类型选项
const screenshotTypeOptions = [
  { label: '全部类型', value: '' },
  { label: '上传成功', value: 'upload_success' },
  { label: '发布成功', value: 'publish_success' },
  { label: '错误截图', value: 'error' },
  { label: '超时截图', value: 'timeout' }
]

// 当前账号的截图列表
const accountScreenshots = computed(() => {
  if (!currentAccount.value) return []
  return screenshotStore.screenshots.filter(s => 
    s.platform === currentAccount.value?.platform
  )
})

// 当前账号的截图数量
function getAccountScreenshotCount(platform: string): number {
  const stat = screenshotStore.platformStats.find(s => s.platform === platform)
  return stat?.screenshotCount || 0
}

async function handleAddAccount() {
  if (!newAccount.value.platform || !newAccount.value.name.trim()) {
    ElMessage.warning('请填写完整信息')
    return
  }

  adding.value = true
  try {
    await accountStore.addAccount(newAccount.value.platform, newAccount.value.name.trim())
    ElMessage.success('账号添加成功')
    showAddDialog.value = false
    newAccount.value = { platform: '', name: '' }
  } catch (error: any) {
    const errorMsg = error?.message || error?.toString() || '添加账号失败'
    console.error('添加账号失败:', error)
    ElMessage.error(`添加账号失败: ${errorMsg}`)
  } finally {
    adding.value = false
  }
}

async function handleDeleteAccount(account: Account) {
  try {
    await ElMessageBox.confirm(
      `确定要删除账号 "${account.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await accountStore.deleteAccount(account.id)
    ElMessage.success('账号已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function handleValidateAccount(account: Account) {
  validating.value = account.id
  try {
    const valid = await accountStore.validateAccount(account.id)
    if (valid) {
      ElMessage.success('账号验证通过')
    } else {
      ElMessage.warning('账号已失效，请重新登录')
    }
  } catch (error) {
    ElMessage.error('验证失败')
  } finally {
    validating.value = null
  }
}

async function handleLoginAccount(account: Account) {
  loggingIn.value = account.id
  try {
    await accountStore.loginAccount(account.id)
    ElMessage.info('请在新打开的浏览器窗口中完成登录')
  } catch (error) {
    ElMessage.error('登录失败')
    loggingIn.value = null
  }
}

async function handleReloginAccount(account: Account) {
  // 确认重新登录
  try {
    await ElMessageBox.confirm(
      '确定要重新登录吗？这将清除当前登录状态，需要重新扫码。',
      '确认重新登录',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return // 用户取消
  }

  loggingIn.value = account.id
  try {
    await accountStore.reloginAccount(account.id)
    ElMessage.info('请在新打开的浏览器窗口中完成登录')
  } catch (error) {
    ElMessage.error('重新登录失败')
    loggingIn.value = null
  }
}

// 打开截图管理抽屉
async function openScreenshotDrawer(account: Account) {
  currentAccount.value = account
  showScreenshotDrawer.value = true
  screenshotFilter.value.platform = account.platform
  await loadScreenshots()
}

// 加载截图列表
async function loadScreenshots() {
  const query: any = {
    platform: screenshotFilter.value.platform,
    type: screenshotFilter.value.type,
    page: screenshotStore.currentPage,
    pageSize: screenshotStore.pageSize
  }
  
  if (screenshotFilter.value.dateRange && screenshotFilter.value.dateRange.length === 2) {
    query.startDate = screenshotFilter.value.dateRange[0].toISOString().split('T')[0]
    query.endDate = screenshotFilter.value.dateRange[1].toISOString().split('T')[0]
  }
  
  await screenshotStore.fetchScreenshots(query)
}

// 处理截图选择
function handleScreenshotSelect(selection: Screenshot[]) {
  screenshotStore.selectedIds = selection.map(s => s.id)
}

// 删除单个截图
async function handleDeleteScreenshot(screenshot: Screenshot) {
  try {
    await ElMessageBox.confirm(
      '确定要删除这张截图吗？',
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await screenshotStore.removeScreenshot(screenshot.id)
    ElMessage.success('截图已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 批量删除截图
async function handleBatchDeleteScreenshots() {
  if (screenshotStore.selectedIds.length === 0) {
    ElMessage.warning('请先选择要删除的截图')
    return
  }
  
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${screenshotStore.selectedIds.length} 张截图吗？`,
      '确认批量删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const deleted = await screenshotStore.removeSelected()
    ElMessage.success(`已删除 ${deleted} 张截图`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

// 删除所有截图
async function handleDeleteAllScreenshots() {
  try {
    await ElMessageBox.confirm(
      '确定要删除所有截图吗？此操作不可恢复！',
      '确认一键删除',
      {
        confirmButtonText: '全部删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )
    
    const deleted = await screenshotStore.removeAll()
    ElMessage.success(`已删除 ${deleted} 张截图`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 打开截图目录
async function handleOpenScreenshotDir() {
  try {
    await screenshotStore.openDir(currentAccount.value?.platform || '')
  } catch (error) {
    ElMessage.error('打开目录失败')
  }
}

// 预览截图
const previewVisible = ref(false)
const previewImage = ref('')

function handlePreviewScreenshot(screenshot: Screenshot) {
  previewImage.value = `file://${screenshot.path}`
  previewVisible.value = true
}

// 处理分页
function handlePageChange(page: number) {
  screenshotStore.setPage(page)
  loadScreenshots()
}

function handlePageSizeChange(size: number) {
  screenshotStore.setPageSize(size)
  loadScreenshots()
}

// 处理筛选变化
function handleFilterChange() {
  screenshotStore.setPage(1)
  loadScreenshots()
}

function getPlatformColor(platform: string): string {
  return PLATFORM_CONFIG[platform as PlatformType]?.color || 'var(--primary-color)'
}

function getPlatformName(platform: string): string {
  return PLATFORM_CONFIG[platform as PlatformType]?.name || platform
}

function getScreenshotTypeLabel(type: string): string {
  return SCREENSHOT_TYPE_LABELS[type] || type
}

onMounted(() => {
  accountStore.fetchAccounts()
  screenshotStore.fetchPlatformStats()
})
</script>

<template>
  <div class="accounts-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">账号管理</h2>
        <p class="page-subtitle">管理您的社交平台账号</p>
      </div>
      <el-button type="primary" @click="showAddDialog = true">
        <el-icon><Plus /></el-icon>
        添加账号
      </el-button>
    </div>

    <div class="accounts-grid" v-if="accountStore.accounts.length > 0">
      <div
        v-for="account in accountStore.accounts"
        :key="account.id"
        class="account-card"
      >
        <div class="account-header">
          <div class="platform-badge" :style="{ backgroundColor: getPlatformColor(account.platform) + '20' }">
            <el-icon :size="20" :color="getPlatformColor(account.platform)">
              <component :is="PLATFORM_CONFIG[account.platform]?.icon || 'Platform'" />
            </el-icon>
          </div>
          <div class="account-status">
            <el-tag
              :type="ACCOUNT_STATUS_LABELS[account.status]?.type"
              size="small"
              effect="dark"
            >
              {{ ACCOUNT_STATUS_LABELS[account.status]?.label }}
            </el-tag>
          </div>
        </div>

        <div class="account-info">
          <h3 class="account-name">{{ account.name }}</h3>
          <p class="account-platform">{{ getPlatformName(account.platform) }}</p>
          <p class="account-username" v-if="account.username">
            <el-icon><User /></el-icon>
            {{ account.username }}
          </p>
        </div>

        <!-- 截图数量显示 -->
        <div class="screenshot-count" @click="openScreenshotDrawer(account)">
          <el-icon><Picture /></el-icon>
          <span>{{ getAccountScreenshotCount(account.platform) }} 张截图</span>
          <el-icon class="arrow-icon"><ArrowRight /></el-icon>
        </div>

        <div class="account-actions">
          <el-button
            v-if="account.status !== 1"
            type="primary"
            size="small"
            :loading="loggingIn === account.id"
            @click="handleLoginAccount(account)"
          >
            <el-icon><Key /></el-icon>
            登录
          </el-button>
          <el-button
            v-else
            type="warning"
            size="small"
            :loading="loggingIn === account.id"
            @click="handleReloginAccount(account)"
          >
            <el-icon><Refresh /></el-icon>
            重新登录
          </el-button>

          <el-button
            type="default"
            size="small"
            :loading="validating === account.id"
            @click="handleValidateAccount(account)"
          >
            <el-icon><CircleCheck /></el-icon>
            验证
          </el-button>

          <el-button
            type="danger"
            size="small"
            plain
            @click="handleDeleteAccount(account)"
          >
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <el-empty
      v-else
      description="暂无账号，点击右上角添加"
      :image-size="120"
    >
      <el-button type="primary" @click="showAddDialog = true">
        添加账号
      </el-button>
    </el-empty>

    <!-- 添加账号对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加新账号"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form label-position="top">
        <el-form-item label="选择平台">
          <el-select
            v-model="newAccount.platform"
            placeholder="请选择平台"
            style="width: 100%"
          >
            <el-option
              v-for="option in platformOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </el-form-item>
        
        <el-form-item label="账号名称">
          <el-input
            v-model="newAccount.name"
            placeholder="请输入账号名称/备注"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button
          type="primary"
          :loading="adding"
          :disabled="!newAccount.platform || !newAccount.name.trim()"
          @click="handleAddAccount"
        >
          添加
        </el-button>
      </template>
    </el-dialog>

    <!-- 截图管理抽屉 -->
    <el-drawer
      v-model="showScreenshotDrawer"
      :title="`${currentAccount?.name || ''} - 截图管理`"
      size="800px"
      destroy-on-close
    >
      <div class="screenshot-drawer-content">
        <!-- 筛选栏 -->
        <div class="screenshot-filters">
          <el-select
            v-model="screenshotFilter.type"
            placeholder="截图类型"
            clearable
            style="width: 150px"
            @change="handleFilterChange"
          >
            <el-option
              v-for="option in screenshotTypeOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          
          <el-date-picker
            v-model="screenshotFilter.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            @change="handleFilterChange"
          />
          
          <el-button @click="handleOpenScreenshotDir">
            <el-icon><FolderOpened /></el-icon>
            打开目录
          </el-button>
        </div>

        <!-- 操作栏 -->
        <div class="screenshot-actions">
          <el-button
            type="danger"
            :disabled="!screenshotStore.hasSelection"
            @click="handleBatchDeleteScreenshots"
          >
            <el-icon><Delete /></el-icon>
            批量删除 ({{ screenshotStore.selectedCount }})
          </el-button>
          
          <el-button type="danger" plain @click="handleDeleteAllScreenshots">
            <el-icon><DeleteFilled /></el-icon>
            一键删除全部
          </el-button>
        </div>

        <!-- 截图列表 -->
        <el-table
          :data="screenshotStore.screenshots"
          v-loading="screenshotStore.loading"
          @selection-change="handleScreenshotSelect"
          style="width: 100%"
        >
          <el-table-column type="selection" width="55" />
          
          <el-table-column label="预览" width="100">
            <template #default="{ row }">
              <div class="screenshot-preview-thumb" @click="handlePreviewScreenshot(row)">
                <el-icon><Picture /></el-icon>
              </div>
            </template>
          </el-table-column>
          
          <el-table-column label="文件名" prop="filename" show-overflow-tooltip />
          
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-tag size="small">{{ getScreenshotTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          
          <el-table-column label="大小" width="100">
            <template #default="{ row }">
              {{ screenshotStore.formatFileSize(row.size) }}
            </template>
          </el-table-column>
          
          <el-table-column label="创建时间" width="180">
            <template #default="{ row }">
              {{ new Date(row.createdAt).toLocaleString() }}
            </template>
          </el-table-column>
          
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                type="danger"
                size="small"
                circle
                @click="handleDeleteScreenshot(row)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="screenshot-pagination">
          <el-pagination
            v-model:current-page="screenshotStore.currentPage"
            v-model:page-size="screenshotStore.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="screenshotStore.total"
            layout="total, sizes, prev, pager, next"
            @size-change="handlePageSizeChange"
            @current-change="handlePageChange"
          />
        </div>

        <!-- 统计信息 -->
        <div class="screenshot-stats">
          <el-descriptions :column="3" border>
            <el-descriptions-item label="总截图数">{{ screenshotStore.total }} 张</el-descriptions-item>
            <el-descriptions-item label="总大小">{{ screenshotStore.formatFileSize(screenshotStore.totalSize) }}</el-descriptions-item>
            <el-descriptions-item label="选中">{{ screenshotStore.selectedCount }} 张</el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
    </el-drawer>

    <!-- 图片预览 -->
    <el-image-viewer
      v-if="previewVisible"
      :url-list="[previewImage]"
      @close="previewVisible = false"
    />
  </div>
</template>

<style scoped>
.accounts-page {
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

.accounts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--spacing-lg);
}

.account-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  transition: all var(--transition-fast);
}

.account-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  border-color: var(--primary-color);
}

.account-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-md);
}

.platform-badge {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.account-info {
  margin-bottom: var(--spacing-md);
}

.account-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-xs) 0;
}

.account-platform {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-sm) 0;
}

.account-username {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0;
}

.screenshot-count {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-sm);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  margin-bottom: var(--spacing-md);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-size: 13px;
  color: var(--text-secondary);
}

.screenshot-count:hover {
  background: var(--primary-color);
  color: white;
}

.screenshot-count:hover .arrow-icon {
  transform: translateX(4px);
}

.arrow-icon {
  margin-left: auto;
  transition: transform var(--transition-fast);
}

.account-actions {
  display: flex;
  gap: var(--spacing-sm);
}

.account-actions .el-button {
  flex: 1;
}

/* 截图抽屉样式 */
.screenshot-drawer-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
  height: 100%;
}

.screenshot-filters {
  display: flex;
  gap: var(--spacing-md);
  flex-wrap: wrap;
  align-items: center;
}

.screenshot-actions {
  display: flex;
  gap: var(--spacing-md);
}

.screenshot-preview-thumb {
  width: 60px;
  height: 60px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.screenshot-preview-thumb:hover {
  background: var(--primary-color);
  color: white;
}

.screenshot-pagination {
  display: flex;
  justify-content: center;
  padding-top: var(--spacing-md);
}

.screenshot-stats {
  margin-top: auto;
  padding-top: var(--spacing-lg);
}

:deep(.el-descriptions__label) {
  background: var(--bg-secondary);
  color: var(--text-primary) !important;
  font-weight: 500;
}

:deep(.el-descriptions__content) {
  background: var(--bg-card);
  color: var(--text-primary) !important;
}
</style>
