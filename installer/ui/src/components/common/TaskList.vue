<script setup lang="ts">
import type { Task } from '@/types/install'

defineProps<{
  tasks: Task[]
}>()
</script>

<template>
  <ul class="task-list">
    <li
      v-for="task in tasks"
      :key="task.id"
      class="task-item"
      :class="`task-item--${task.status}`"
    >
      <span class="task-icon" aria-hidden="true">
        <span v-if="task.status === 'pending'" class="task-dot task-dot--pending" />
        <v-progress-circular
          v-else-if="task.status === 'running'"
          indeterminate
          size="16"
          width="2"
          color="primary"
        />
        <svg v-else-if="task.status === 'done'" viewBox="0 0 16 16" fill="none" width="16" height="16">
          <circle cx="8" cy="8" r="7" fill="#16a34a" />
          <path d="M4.5 8.5l2.5 2.5 4.5-5" stroke="#fff" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <svg v-else-if="task.status === 'error'" viewBox="0 0 16 16" fill="none" width="16" height="16">
          <circle cx="8" cy="8" r="7" fill="#dc2626" />
          <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="#fff" stroke-width="1.6" stroke-linecap="round"/>
        </svg>
        <svg v-else-if="task.status === 'cancelled'" viewBox="0 0 16 16" fill="none" width="16" height="16">
          <circle cx="8" cy="8" r="7" fill="#f59e0b" />
          <path d="M5.5 8.5l5 0" stroke="#fff" stroke-width="1.6" stroke-linecap="round"/>
        </svg>
      </span>

      <span class="task-num">{{ task.id }}.</span>
      <span class="task-name">{{ task.name }}</span>
    </li>
  </ul>
</template>

<style scoped>
.task-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.task-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 10px;
  border-radius: 8px;
  transition: background-color .12s ease;
}

.task-item--running {
  background: #eff6ff;
}
.task-item--error {
  background: #fef2f2;
}
.task-item--cancelled {
  background: #fffbeb;
}

.task-icon {
  flex: 0 0 auto;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.task-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
}
.task-dot--pending {
  background: #e2e8f0;
  border: 1.5px solid #cbd5e1;
}

.task-num {
  font-size: 12px;
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
  min-width: 20px;
}

.task-name {
  font-size: 13.5px;
  color: #334155;
  line-height: 1.3;
}

.task-item--running .task-name { color: #1e40af; font-weight: 500; }
.task-item--done   .task-name  { color: #15803d; }
.task-item--error  .task-name  { color: #b91c1c; font-weight: 500; }
.task-item--pending .task-name { color: #94a3b8; }
</style>
