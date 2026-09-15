/* In-memory catalogue for the offline mock API. Sizes marked with `note`
   are placeholders and must be verified before seeding (ui/UI_REVIEW.md UI-16). */
import type { Banner, Category, ClothingOption, Collection, Spec, Template, Work } from '@/types'

const M = (f: string) => `/static/mock/${f}.jpg`
const VERIFY = '示例数据，上线前核实'

export const SPEC_CATEGORIES = [
  { id: 'sc_common', name: '常用证件照' },
  { id: 'sc_exam', name: '考试报名' },
  { id: 'sc_job', name: '求职招聘' },
  { id: 'sc_cert', name: '资格认证' },
  { id: 'sc_visa', name: '签证' },
  { id: 'sc_passport', name: '护照' },
  { id: 'sc_school', name: '学籍' },
]

const BG = ['#FFFFFF', '#438EDB', '#FF0000', '#808080']
const spec = (id: string, name: string, wmm: number, hmm: number, wpx: number, hpx: number, cat: string, extra: Partial<Spec> = {}): Spec => ({
  id, name, width_mm: wmm, height_mm: hmm, width_px: wpx, height_px: hpx, dpi: 300,
  bg_default: '#438EDB', bg_allowed: BG, category: SPEC_CATEGORIES.find((c) => c.id === cat)!, ...extra,
})

export const SPECS: Spec[] = [
  spec('sp_1inch', '一寸', 25, 35, 295, 413, 'sc_common', { is_hot: true, note: '国内通用' }),
  spec('sp_2inch', '二寸', 35, 49, 413, 579, 'sc_common', { is_hot: true, note: '国内通用' }),
  spec('sp_s2inch', '小二寸', 35, 45, 413, 531, 'sc_common', { is_hot: true, note: '国内通用' }),
  spec('sp_b1inch', '大一寸', 33, 48, 390, 567, 'sc_common', { is_hot: true, note: '港澳通行证等' }),
  spec('sp_civil', '公务员报名', 25, 35, 295, 413, 'sc_exam', { note: VERIFY }),
  spec('sp_cet', '四六级报名', 25, 35, 295, 413, 'sc_exam', { note: VERIFY }),
  spec('sp_resume', '简历照', 25, 35, 295, 413, 'sc_job', { bg_default: '#FFFFFF' }),
  spec('sp_teacher', '教师资格', 35, 45, 413, 531, 'sc_cert', { note: VERIFY }),
  spec('sp_schengen', '申根签证', 35, 45, 413, 531, 'sc_visa', { bg_default: '#FFFFFF', bg_allowed: ['#FFFFFF'] }),
  spec('sp_us', '美国签证', 51, 51, 600, 600, 'sc_visa', { bg_default: '#FFFFFF', bg_allowed: ['#FFFFFF'] }),
  spec('sp_jp', '日本签证', 45, 45, 531, 531, 'sc_visa', { bg_default: '#FFFFFF', bg_allowed: ['#FFFFFF'] }),
  spec('sp_passport', '中国护照', 33, 48, 390, 567, 'sc_passport', { bg_default: '#FFFFFF', bg_allowed: ['#FFFFFF'] }),
  spec('sp_student', '学籍照', 35, 45, 413, 531, 'sc_school', { note: VERIFY }),
]

export const CLOTHING: ClothingOption[] = [
  { id: 'keep', name: '保持原服装', group: 'keep' },
  { id: 'm_white_shirt', name: '白衬衫', group: 'male' },
  { id: 'm_black_suit', name: '黑西装', group: 'male' },
  { id: 'm_navy_suit', name: '深蓝西装', group: 'male' },
  { id: 'f_white_shirt', name: '白衬衫', group: 'female' },
  { id: 'f_suit', name: '女士职业装', group: 'female' },
  { id: 'f_navy_suit', name: '深蓝西装', group: 'female' },
]

export const CATEGORIES: Category[] = [
  { id: 'c_interview', module: 'pro', name: '求职面试', icon: 'briefcase', cover_url: M('pro_interview'), desc: '正装·纯色背景' },
  { id: 'c_business', module: 'pro', name: '商务职场', icon: 'users', cover_url: M('pro_business'), desc: '西装·办公室' },
  { id: 'c_lecturer', module: 'pro', name: '讲师头像', icon: 'presentation', cover_url: M('pro_lecturer'), desc: '半身·浅色背景' },
  { id: 'c_website', module: 'pro', name: '企业官网', icon: 'building', cover_url: M('pro_website'), desc: '正装·灰色背景' },
  { id: 'c_consultant', module: 'pro', name: '顾问', icon: 'user-check', cover_url: M('id_male1'), desc: '正装·浅色背景' },
  { id: 'c_doctor', module: 'pro', name: '医生', icon: 'shield', cover_url: M('id_female1'), desc: '白大褂·白底' },

  { id: 'c_korean', module: 'portrait', name: '韩系清透', cover_url: M('portrait_korean'), desc: '自然光 · 户外' },
  { id: 'c_french', module: 'portrait', name: '法式氛围', cover_url: M('portrait_french'), desc: '街拍 · 胶片感' },
  { id: 'c_chinese', module: 'portrait', name: '新中式', cover_url: M('portrait_chinese'), desc: '国风 · 室内' },
  { id: 'c_birthday', module: 'portrait', name: '生日写真', cover_url: M('portrait_birthday'), desc: '气球 · 蛋糕' },
  { id: 'c_campus', module: 'portrait', name: '校园', desc: '校园 · 日常' },
  { id: 'c_travel', module: 'portrait', name: '旅行', desc: '海边 · 山野' },
  { id: 'c_autumn', module: 'portrait', name: '秋日', desc: '暖色 · 户外' },
  { id: 'c_hk', module: 'portrait', name: '港风', desc: '复古 · 胶片' },

  { id: 'c_wechat', module: 'avatar', name: '微信头像', icon: 'message-circle', cover_url: M('avatar_mood'), desc: '1:1·近景' },
  { id: 'c_premium', module: 'avatar', name: '高级感', icon: 'crown', cover_url: M('avatar_premium'), desc: '1:1·影棚光' },
  { id: 'c_fresh', module: 'avatar', name: '清新自然', icon: 'leaf', cover_url: M('work_portrait'), desc: '1:1·户外' },
  { id: 'c_illust', module: 'avatar', name: '插画风', icon: 'paint', cover_url: M('avatar_illust'), desc: '1:1·插画' },
]

const tpl = (id: string, module: Template['module'], name: string, cover: string, cat: string, subtitle: string, extra: Partial<Template> = {}): Template => ({
  id, module, name, cover_url: M(cover), tags: [], is_favorited: false, credit_cost: 1, subtitle,
  sample_urls: [M(cover)], output: module === 'avatar' ? { width: 1024, height: 1024, aspect: '1:1' } : { width: 1200, height: 1600, aspect: '3:4' },
  style: 'photo', category: (() => { const c = CATEGORIES.find((x) => x.id === cat); return c ? { id: c.id, name: c.name } : undefined })(), ...extra,
})

export const TEMPLATES: Template[] = [
  tpl('t_interview', 'pro', '面试职业照', 'pro_interview', 'c_interview', '深色西装 · 浅蓝背景 · 3:4', { tags: ['热门'] }),
  tpl('t_business', 'pro', '商务精英', 'pro_business', 'c_business', '白色西装 · 办公室背景 · 3:4', { tags: ['热门'] }),
  tpl('t_lecturer', 'pro', '讲师介绍图', 'pro_lecturer', 'c_lecturer', '半身 · 浅灰背景 · 3:4'),
  tpl('t_website', 'pro', '官网头像', 'pro_website', 'c_website', '正装 · 灰色背景 · 3:4'),
  tpl('t_consultant', 'pro', '顾问形象照', 'id_male1', 'c_consultant', '深蓝西装 · 白色背景 · 3:4', { tags: ['NEW'] }),
  tpl('t_doctor', 'pro', '医生形象照', 'id_female1', 'c_doctor', '白大褂 · 白色背景 · 3:4'),

  tpl('t_autumn', 'portrait', '秋日写真', 'portrait_autumn', 'c_autumn', '暖色调 · 户外 · 3:4', { tags: ['热门'] }),
  tpl('t_street', 'portrait', '法式街拍', 'portrait_street', 'c_french', '街拍 · 胶片感 · 3:4'),
  tpl('t_chinese', 'portrait', '新中式写真', 'portrait_chinese2', 'c_chinese', '国风 · 室内 · 3:4', { tags: ['热门'] }),
  tpl('t_birthday', 'portrait', '清新生日照', 'portrait_birthday2', 'c_birthday', '气球 · 蛋糕 · 3:4', { tags: ['NEW'] }),
  tpl('t_korean', 'portrait', '韩系清透写真', 'portrait_korean', 'c_korean', '自然光 · 户外 · 3:4'),
  tpl('t_french', 'portrait', '法式氛围写真', 'portrait_french', 'c_french', '贝雷帽 · 街景 · 3:4'),

  tpl('t_av_premium', 'avatar', '微信高级感', 'avatar_premium', 'c_premium', '影棚光 · 1:1', { tags: ['热门'] }),
  tpl('t_av_mood', 'avatar', '氛围感头像', 'avatar_mood', 'c_wechat', '暗调 · 1:1'),
  tpl('t_av_illust', 'avatar', '插画头像', 'avatar_illust', 'c_illust', '二次元 · 1:1', { style: 'illustration', tags: ['NEW'] }),
  tpl('t_av_fresh', 'avatar', '清新自然头像', 'work_portrait', 'c_fresh', '户外 · 1:1'),
]

export const COLLECTIONS: Collection[] = [
  { id: 'col_light', module: 'portrait', name: '轻写真', cover_url: M('portrait_korean'), desc: '自然光，少修饰', template_count: 12 },
  { id: 'col_half', module: 'portrait', name: '半身写真', cover_url: M('portrait_autumn'), desc: '半身构图', template_count: 9 },
  { id: 'col_festival', module: 'portrait', name: '节日主题', cover_url: M('portrait_birthday'), desc: '生日、圣诞、新年', template_count: 15 },
  { id: 'col_mood', module: 'portrait', name: '氛围写真', cover_url: M('portrait_street'), desc: '胶片感与暗调', template_count: 10 },
]

export const BANNERS: Record<string, Banner> = {
  idphoto: { title: '上传自拍，\n快速生成标准证件照', subtitle: '智能识别 · 自动裁切 · 多种规格', image_url: M('work_id'), link: { type: 'upload' } },
  pro: { title: '上传自拍，\n生成职业形象照', subtitle: '多种场景 · 正装换装 · 3:4 输出', image_url: M('work_pro'), link: { type: 'upload' } },
  portrait: { title: '上传自拍，\n生成氛围感写真', subtitle: '多种风格 · 一键生成', image_url: M('work_portrait'), link: { type: 'upload' } },
  avatar: { title: '上传自拍，\n生成专属头像', subtitle: '方图输出 · 多种风格', image_url: M('portrait_french'), link: { type: 'upload' } },
}

export const HOT_CATEGORY_IDS: Record<string, string[]> = {
  pro: ['c_interview', 'c_business', 'c_lecturer', 'c_website'],
  portrait: ['c_korean', 'c_french', 'c_chinese', 'c_birthday'],
  avatar: ['c_wechat', 'c_premium', 'c_fresh', 'c_illust'],
}

export const HOT_TEMPLATE_IDS: Record<string, string[]> = {
  pro: ['t_interview', 't_business', 't_lecturer', 't_website', 't_consultant', 't_doctor'],
  portrait: ['t_autumn', 't_street', 't_chinese', 't_birthday', 't_korean', 't_french'],
  avatar: ['t_av_premium', 't_av_mood', 't_av_illust', 't_av_fresh'],
}

/** Secondary rails on the tab homes: title + template ids (mock reuses templates across rails). */
export const RAILS: Record<string, { title: string; category_id?: string; collection_id?: string; ids: string[] }[]> = {
  pro: [
    { title: '求职面试', category_id: 'c_interview', ids: ['t_interview', 't_consultant', 't_website', 't_lecturer'] },
    { title: '商务职场', category_id: 'c_business', ids: ['t_business', 't_doctor', 't_interview', 't_website'] },
  ],
  portrait: [
    { title: '本周新增', ids: ['t_birthday', 't_street', 't_french', 't_autumn'] },
  ],
  avatar: [
    { title: '插画风', category_id: 'c_illust', ids: ['t_av_illust', 't_av_fresh', 't_av_mood', 't_av_premium'] },
  ],
}

export const MORE_SPEC_IDS = ['sp_civil', 'sp_resume', 'sp_teacher', 'sp_passport']

const daysAgo = (d: number) => new Date(Date.now() - d * 864e5).toISOString()

export const INITIAL_WORKS: Work[] = [
  { id: 'w_1', module: 'idphoto', url: M('work_id'), thumb_url: M('work_id'), width: 295, height: 413, created_at: daysAgo(1), task_id: 'tk_w1',
    spec: { id: 'sp_1inch', name: '一寸', width_mm: 25, height_mm: 35, width_px: 295, height_px: 413 }, meta: { bg: '#438EDB', clothing: 'keep' }, ai_label: false },
  { id: 'w_2', module: 'pro', url: M('work_pro'), thumb_url: M('work_pro'), width: 1200, height: 1600, created_at: daysAgo(2), task_id: 'tk_w2',
    template: { id: 't_interview', name: '面试职业照' }, meta: {}, ai_label: true },
  { id: 'w_3', module: 'portrait', url: M('work_portrait'), thumb_url: M('work_portrait'), width: 1200, height: 1600, created_at: daysAgo(3), task_id: 'tk_w3',
    template: { id: 't_korean', name: '韩系清透写真' }, meta: {}, ai_label: true },
  { id: 'w_4', module: 'avatar', url: M('avatar_premium'), thumb_url: M('avatar_premium'), width: 1024, height: 1024, created_at: daysAgo(5), task_id: 'tk_w4',
    template: { id: 't_av_premium', name: '微信高级感' }, meta: {}, ai_label: true },
  { id: 'w_5', module: 'idphoto', url: M('id_female2'), thumb_url: M('id_female2'), width: 413, height: 579, created_at: daysAgo(8), task_id: 'tk_w5',
    spec: { id: 'sp_2inch', name: '二寸', width_mm: 35, height_mm: 49, width_px: 413, height_px: 579 }, meta: { bg: '#438EDB', clothing: 'keep' }, ai_label: false },
]

/** Sample outputs used by the mock generator, keyed by module. */
export const SAMPLE_OUTPUT: Record<string, string[]> = {
  idphoto: [M('id_female1'), M('id_male1'), M('id_female2'), M('id_male2')],
  pro: [M('pro_interview'), M('pro_business'), M('pro_lecturer'), M('pro_website')],
  portrait: [M('portrait_autumn'), M('portrait_street'), M('portrait_chinese2'), M('portrait_korean')],
  avatar: [M('avatar_premium'), M('avatar_mood'), M('avatar_illust'), M('work_portrait')],
}

export const MOCK_USER = { id: 'u_1', nickname: '阳光小橙', avatar_url: M('user_avatar') }
