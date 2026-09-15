import { defineStore } from 'pinia'
import type { Task } from '@/types'

export const useTasksStore = defineStore('tasks', {
  state: () => ({ byId: {} as Record<string, Task> }),
  actions: {
    set(task: Task) { this.byId[task.id] = task },
    get(id: string): Task | undefined { return this.byId[id] },
  },
})
