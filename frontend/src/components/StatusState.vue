<script setup>
import { computed } from "vue";
import { Box, CircleCloseFilled, Loading } from "@element-plus/icons-vue";

const props = defineProps({
  type: {
    type: String,
    default: "empty",
  },
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
});

const iconComponent = computed(() => {
  if (props.type === "loading") {
    return Loading;
  }
  if (props.type === "error") {
    return CircleCloseFilled;
  }
  return Box;
});
</script>

<template>
  <div class="state-card" :class="`is-${type}`">
    <el-icon class="state-icon" :class="{ spinning: type === 'loading' }">
      <component :is="iconComponent" />
    </el-icon>
    <h3 class="state-title">{{ title }}</h3>
    <p v-if="description" class="state-desc">{{ description }}</p>
  </div>
</template>

<style scoped>
.state-card {
  min-height: 220px;
  border-radius: var(--radius-lg);
  border: 1px dashed var(--line-color);
  background: var(--surface-2);
  color: var(--text-primary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
}

.state-card.is-error {
  border-color: rgba(239, 68, 68, 0.35);
  background: rgba(239, 68, 68, 0.06);
}

.state-icon {
  font-size: 24px;
  color: var(--text-tertiary);
}

.state-card.is-error .state-icon {
  color: var(--danger-color);
}

.state-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.state-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  text-align: center;
  max-width: 520px;
}

.spinning {
  animation: spin 1.1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
