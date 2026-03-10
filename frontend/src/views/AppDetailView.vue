<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import { loadAppsConfig, useAppsStore } from "../stores/appsStore";

const route = useRoute();
const { state, findApp } = useAppsStore();

const category = computed(() => String(route.params.category || ""));
const appName = computed(() => String(route.params.name || ""));

const app = computed(() => findApp(category.value, appName.value));

const developer = computed(() => {
  if (!app.value?.repo) {
    return "未知";
  }
  const parts = app.value.repo.split("/");
  if (parts.length === 2 && parts[0]) {
    return parts[0];
  }
  return "未知";
});

const repoUrl = computed(() => {
  if (!app.value?.repo) {
    return "";
  }
  return `https://github.com/${app.value.repo}`;
});

loadAppsConfig();
</script>

<template>
  <section class="detail-page">
    <div v-if="state.loading" class="placeholder">正在加载应用详情...</div>
    <div v-else-if="state.error" class="placeholder error">
      配置加载失败：{{ state.error }}
    </div>
    <div v-else-if="!app" class="placeholder">未找到该应用。</div>
    <div v-else class="detail-card">
      <header class="detail-header">
        <img class="icon" :src="app.photo" :alt="app.name" />
        <div class="meta">
          <h1>{{ app.name }}</h1>
          <p>{{ category }}</p>
        </div>
      </header>

      <div class="row">
        <span class="label">介绍</span>
        <span class="value muted">阶段1占位：后续接入仓库介绍信息</span>
      </div>
      <div class="row">
        <span class="label">版本</span>
        <span class="value muted">阶段1占位：待接入 Release</span>
      </div>
      <div class="row">
        <span class="label">开发者</span>
        <span class="value">{{ developer }}</span>
      </div>
      <div class="row">
        <span class="label">原仓库</span>
        <a class="value link" :href="repoUrl" target="_blank">{{ app.repo }}</a>
      </div>

      <div class="download-panel">
        <button disabled>Windows 下载（占位）</button>
        <button disabled>macOS 下载（占位）</button>
        <button disabled>Linux 下载（占位）</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.detail-page {
  padding: 24px;
}

.placeholder {
  border: 1px dashed #d1d5db;
  border-radius: 12px;
  background: #f8fafc;
  padding: 20px;
  color: #374151;
}

.placeholder.error {
  border-color: #fca5a5;
  background: #fef2f2;
  color: #991b1b;
}

.detail-card {
  border: 1px solid #e5e7eb;
  background: #ffffff;
  border-radius: 14px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  background: #f3f4f6;
}

.meta h1 {
  margin: 0;
  font-size: 22px;
}

.meta p {
  margin: 6px 0 0;
  color: #6b7280;
}

.row {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 10px;
  align-items: start;
}

.label {
  color: #6b7280;
}

.value {
  color: #111827;
}

.muted {
  color: #6b7280;
}

.link {
  color: #2563eb;
  text-decoration: none;
}

.link:hover {
  text-decoration: underline;
}

.download-panel {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.download-panel button {
  border: 1px solid #d1d5db;
  border-radius: 8px;
  padding: 8px 12px;
  background: #f9fafb;
  color: #6b7280;
}
</style>
