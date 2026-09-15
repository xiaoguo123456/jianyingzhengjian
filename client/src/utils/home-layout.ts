import type { HomePayload, TemplateCard } from '@/types'

/** 首页按内容 ID 去重，同一模板不在不同推荐区重复出现。 */
export function homeTemplates(home: HomePayload | null): TemplateCard[] {
  const seen = new Set<string>()
  return [...(home?.hot_templates || []), ...(home?.rails || []).flatMap((rail) => rail.templates)]
    .filter((template) => {
      if (seen.has(template.id)) return false
      seen.add(template.id)
      return true
    })
}
