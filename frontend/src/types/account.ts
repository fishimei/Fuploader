// 平台类型
export type PlatformType = 'douyin' | 'tencent' | 'kuaishou' | 'tiktok' | 'bilibili' | 'xiaohongshu' | 'baijiahao'

// 账号状态: 0-无效, 1-有效, 2-已过期
export type AccountStatus = 0 | 1 | 2

// 平台字段类型 - 扩展新类型
export type PlatformFieldType = 
  | 'input' 
  | 'textarea' 
  | 'select' 
  | 'switch' 
  | 'tags' 
  | 'number'
  | 'file'        // 新增：文件上传
  | 'image'       // 新增：图片选择
  | 'datetime'    // 新增：日期时间选择
  | 'collection'  // 新增：合集选择
  | 'product'     // 新增：商品链接

// 日历配置（TikTok专用）
export interface CalendarConfig {
  monthSelector: string
  daySelector: string
  hourSelector: string
  minuteSelector: string
  minuteStep: number
}

// 平台特殊字段定义 - 扩展接口
export interface PlatformField {
  key: string
  label: string
  type: PlatformFieldType
  required?: boolean
  placeholder?: string
  options?: { label: string; value: string }[]
  maxLength?: number
  min?: number
  max?: number
  defaultValue?: any
  
  // 新增：条件显示
  showWhen?: (form: Record<string, any>) => boolean
  
  // 新增：内部字段（不显示给用户）
  internal?: boolean
  
  // 新增：动态加载选项
  loadOptions?: () => Promise<{ label: string; value: string }[]>
  
  // 新增：自动生成
  autoGenerate?: boolean
  generateFrom?: string
  
  // 新增：文件类型限制
  accept?: string
  
  // 新增：日历配置（TikTok专用）
  calendarConfig?: CalendarConfig
  
  // 新增：自动选择相关
  allowAutoSelect?: boolean
  autoSelectText?: string
  
  // 新增：描述信息
  description?: string
}

// 平台配置
export const PLATFORM_CONFIG: Record<PlatformType, { name: string; icon: string; color: string }> = {
  douyin: { name: '抖音', icon: 'VideoCamera', color: '#000000' },
  tencent: { name: '视频号', icon: 'ChatDotRound', color: '#07C160' },
  kuaishou: { name: '快手', icon: 'Orange', color: '#FF6600' },
  tiktok: { name: 'TikTok', icon: 'VideoPlay', color: '#000000' },
  bilibili: { name: 'Bilibili', icon: 'VideoPlay', color: '#FB7299' },
  xiaohongshu: { name: '小红书', icon: 'Notebook', color: '#FF2442' },
  baijiahao: { name: '百家号', icon: 'Document', color: '#2932E1' }
}

// 平台标签数量限制配置（null表示无限制或无标签字段）
export const PLATFORM_TAGS_LIMIT: Record<PlatformType, number | null> = {
  douyin: 5,      // 抖音：最多5个标签
  tencent: 5,     // 视频号：最多5个标签
  kuaishou: 3,    // 快手：最多3个标签
  tiktok: 5,      // TikTok：最多5个标签
  bilibili: 5,    // Bilibili：最多5个标签
  xiaohongshu: 5, // 小红书：最多5个标签
  baijiahao: 5    // 百家号：最多5个标签
}

// 平台发布字段配置 - 扩展各平台字段
export const PLATFORM_PUBLISH_FIELDS: Record<PlatformType, PlatformField[]> = {
  // 抖音 - 补充封面、同步、商品链接
  douyin: [
    { key: 'tags', label: '话题标签', type: 'tags', placeholder: '输入标签，回车确认，如：#美食' },
    { key: 'location', label: '位置', type: 'input', placeholder: '添加位置信息' },
    { 
      key: 'thumbnail', 
      label: '视频封面', 
      type: 'image', 
      placeholder: '选择封面图片',
      allowAutoSelect: true,
      autoSelectText: '使用推荐封面'
    },
    { key: 'allowDownload', label: '允许下载', type: 'switch', defaultValue: true },
    { key: 'allowComment', label: '允许评论', type: 'switch', defaultValue: true },
    { key: 'syncToutiao', label: '同步到今日头条', type: 'switch', defaultValue: false },
    { key: 'syncXigua', label: '同步到西瓜视频', type: 'switch', defaultValue: false },
    { 
      key: 'productLink', 
      label: '商品链接', 
      type: 'product', 
      placeholder: '粘贴商品链接'
    },
    { 
      key: 'productTitle', 
      label: '商品短标题', 
      type: 'input', 
      placeholder: '输入商品短标题',
      showWhen: (form) => !!form.productLink
    }
  ],
  
  // 视频号 - 补充短标题、合集、原创声明
  tencent: [
    { key: 'tags', label: '话题标签', type: 'tags', placeholder: '输入标签，回车确认' },
    { key: 'description', label: '视频描述', type: 'textarea', placeholder: '输入视频描述', maxLength: 200 },
    { key: 'location', label: '位置', type: 'input', placeholder: '添加位置信息' },
    { 
      key: 'shortTitle', 
      label: '短标题', 
      type: 'input', 
      placeholder: '6-16个字符，自动从标题生成',
      maxLength: 16,
      autoGenerate: true,
      generateFrom: 'title'
    },
    { 
      key: 'collection', 
      label: '添加到合集', 
      type: 'collection', 
      placeholder: '选择合集'
    },
    { key: 'isOriginal', label: '声明原创', type: 'switch', defaultValue: false },
    { 
      key: 'originalType', 
      label: '原创类型', 
      type: 'select', 
      options: [
        { label: '知识科普', value: 'knowledge' },
        { label: '生活记录', value: 'lifestyle' },
        { label: '其他', value: 'other' }
      ],
      showWhen: (form) => form.isOriginal === true
    },
    { key: 'isDraft', label: '保存为草稿', type: 'switch', defaultValue: false }
  ],
  
  // 快手
  kuaishou: [
    { key: 'tags', label: '话题标签', type: 'tags', placeholder: '输入标签，回车确认' },
    { key: 'location', label: '位置', type: 'input', placeholder: '添加位置信息' },
    { key: 'allowDownload', label: '允许下载', type: 'switch', defaultValue: true },
    { key: 'useFileChooser', label: '使用文件选择器', type: 'switch', defaultValue: true, internal: true },
    { key: 'skipNewFeatureGuide', label: '跳过新功能引导', type: 'switch', defaultValue: true, internal: true }
  ],
  
  // TikTok - 日历选择器
  tiktok: [
    { key: 'tags', label: 'Hashtags', type: 'tags', placeholder: 'Enter hashtags' },
    { key: 'allowComment', label: 'Allow Comments', type: 'switch', defaultValue: true },
    { key: 'allowDuet', label: 'Allow Duet', type: 'switch', defaultValue: false },
    { 
      key: 'scheduleTime', 
      label: 'Schedule Post', 
      type: 'datetime', 
      placeholder: 'Select publish time',
      calendarConfig: {
        monthSelector: 'div.calendar-wrapper span.month-title',
        daySelector: 'div.calendar-wrapper span.day.valid',
        hourSelector: 'span.tiktok-timepicker-left',
        minuteSelector: 'span.tiktok-timepicker-right',
        minuteStep: 5
      }
    },
    { key: 'useIframe', label: 'Use iframe mode', type: 'switch', defaultValue: true, internal: true }
  ],
  
  // Bilibili
  bilibili: [
    { key: 'tags', label: '标签', type: 'tags', placeholder: '输入标签，回车确认' },
    { key: 'copyright', label: '转载类型', type: 'select', options: [
      { label: '自制', value: '1' },
      { label: '转载', value: '2' }
    ], defaultValue: '1' },
    {
      key: 'thumbnail',
      label: '视频封面',
      type: 'image',
      placeholder: '选择封面图片',
      allowAutoSelect: true,
      autoSelectText: '使用推荐封面'
    }
  ],
  
  // 小红书 - 补充封面
  xiaohongshu: [
    { key: 'tags', label: '话题标签', type: 'tags', placeholder: '输入标签，回车确认' },
    { key: 'location', label: '位置', type: 'input', placeholder: '添加位置信息' },
    { 
      key: 'thumbnail', 
      label: '自定义封面', 
      type: 'image', 
      placeholder: '选择封面图片',
      accept: 'image/*'
    },
    { key: 'syncToutiao', label: '同步到今日头条', type: 'switch', defaultValue: false },
    { key: 'syncXigua', label: '同步到西瓜视频', type: 'switch', defaultValue: false }
  ],
  
  // 百家号 - 补充封面检测、安全验证
  baijiahao: [
    { key: 'tags', label: '标签', type: 'tags', placeholder: '输入标签，回车确认' },
    { key: 'category', label: '分类', type: 'select', required: true, options: [
      { label: '美食', value: 'food' },
      { label: '科技', value: 'tech' },
      { label: '娱乐', value: 'entertainment' },
      { label: '生活', value: 'life' },
      { label: '教育', value: 'education' },
      { label: '体育', value: 'sports' },
      { label: '财经', value: 'finance' },
      { label: '时尚', value: 'fashion' }
    ]},
    { key: 'coverType', label: '封面模式', type: 'select', options: [
      { label: '自动', value: 'auto' },
      { label: '单图', value: 'single' },
      { label: '三图', value: 'triple' }
    ], defaultValue: 'auto' },
    { key: 'original', label: '原创文章', type: 'switch', defaultValue: true },
    { key: 'waitCoverGenerated', label: '等待封面生成', type: 'switch', defaultValue: true, internal: true },
    { key: 'checkSecurityVerify', label: '检测安全验证', type: 'switch', defaultValue: true, internal: true },
    { 
      key: 'autoOptimizeTitle', 
      label: '自动优化标题', 
      type: 'switch', 
      defaultValue: false,
      description: '标题少于8字时自动添加后缀'
    }
  ]
}

// 账号模型
export interface Account {
  id: number
  platform: PlatformType
  name: string
  username?: string
  avatar?: string
  cookiePath?: string
  status: AccountStatus
  createdAt: string
  updatedAt: string
}

// 账号状态变更事件
export interface AccountStatusChangedEvent {
  accountId: number
  oldStatus: AccountStatus
  newStatus: AccountStatus
}

// 登录成功事件
export interface LoginSuccessEvent {
  accountId: number
  platform: PlatformType
  username: string
}

// 登录失败事件
export interface LoginErrorEvent {
  accountId: number
  platform: PlatformType
  error: string
}
