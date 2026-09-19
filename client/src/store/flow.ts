import { defineStore } from 'pinia'
import type { FlowState, Module, Photo, Spec, Template } from '@/types'
import { uuid } from '@/utils/uuid'

const fresh = (): FlowState => ({
  module: null, kind: null, templateId: null, templateName: null, specId: null, spec: null, photo: null,
  params: { bg: '#438EDB', clothing: 'keep', beauty: 'natural' }, idempotencyKey: uuid(), notifyRequested: false, parentTaskId: null,
})

/** Single source of truth for the current generation attempt (docs/FRONTEND_ARCHITECTURE.md §6). */
export const useFlowStore = defineStore('flow', {
  state: () => fresh(),
  getters: {
    hasTarget: (s) => !!(s.templateId || s.specId),
    /** Every generation, ID photos included, is drawn by the gen model and costs one credit (D-26). */
    creditCost: () => 1,
    targetName: (s) => s.templateName || s.spec?.name || '',
  },
  actions: {
    start(module: Module) {
      Object.assign(this, fresh(), { module })
    },
    setTemplate(t: Template) {
      this.module = t.module
      this.kind = 'template'
      this.templateId = t.id
      this.templateName = t.name
      this.specId = null
      this.spec = null
    },
    setSpec(s: Spec) {
      this.module = 'idphoto'
      this.kind = 'idphoto'
      this.specId = s.id
      this.spec = s
      this.templateId = null
      this.templateName = null
      this.params.bg = s.bg_default
    },
    setPhoto(p: Photo | null) { this.photo = p },
    setParams(p: Partial<FlowState['params']>) { Object.assign(this.params, p) },
    setNotify(v: boolean) { this.notifyRequested = v },
    /** New idempotency key for a new attempt (regenerate / change template). */
    newAttempt(parentTaskId: string | null = null) {
      this.idempotencyKey = uuid()
      this.parentTaskId = parentTaskId
    },
  },
})
