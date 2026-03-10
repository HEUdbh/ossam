<script setup>
import { computed } from "vue";
import { Link, User } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import StatusState from "../components/StatusState.vue";
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
    <StatusState
      v-if="state.loading"
      type="loading"
      title="正在加载应用详情"
      description="正在读取本地配置并匹配应用信息。"
    />
    <StatusState
      v-else-if="state.error"
      type="error"
      title="配置加载失败"
      :description="state.error"
    />
    <StatusState
      v-else-if="!app"
      type="empty"
      title="未找到该应用"
      description="请确认路由参数与 appsconfig.json 配置是否一致。"
    />

    <div v-else class="detail-layout">
      <el-card class="overview-card" shadow="never">
        <div class="overview-head">
          <img class="app-icon" :src="app.photo" :alt="app.name" />
          <div class="overview-meta">
            <h2>{{ app.name }}</h2>
            <p>{{ category }}</p>
          </div>
        </div>
      </el-card>

      <el-card class="meta-card" shadow="never">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="介绍">
            <span class="muted">阶段1占位：后续接入仓库介绍信息</span>
          </el-descriptions-item>
          <el-descriptions-item label="版本">
            <span class="muted">阶段1占位：待接入 Release</span>
          </el-descriptions-item>
          <el-descriptions-item label="开发者">
            <el-space :size="6">
              <el-icon><User /></el-icon>
              <span>{{ developer }}</span>
            </el-space>
          </el-descriptions-item>
          <el-descriptions-item label="原仓库">
            <a class="repo-link" :href="repoUrl" target="_blank" rel="noreferrer">
              <el-space :size="6">
                <el-icon><Link /></el-icon>
                <span>{{ app.repo }}</span>
              </el-space>
            </a>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card class="download-card" shadow="never">
        <template #header>
          <span>下载</span>
        </template>
        <div class="download-actions">
          <el-button disabled>Windows 下载（占位）</el-button>
          <el-button disabled>macOS 下载（占位）</el-button>
          <el-button disabled>Linux 下载（占位）</el-button>
        </div>
      </el-card>
    </div>
  </section>
</template>

<style scoped>
.detail-page {
  height: 100%;
  padding: 18px;
}

.detail-layout {
  height: 100%;
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: 14px;
  align-content: start;
}

.overview-card,
.meta-card,
.download-card {
  border-radius: var(--radius-md);
  border-color: var(--line-color);
}

.overview-card {
  grid-column: span 12;
}

.meta-card {
  grid-column: span 12;
}

.download-card {
  grid-column: span 12;
}

.overview-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.app-icon {
  width: 64px;
  height: 64px;
  border-radius: 14px;
  object-fit: cover;
  background: var(--surface-3);
}

.overview-meta h2 {
  margin: 0;
  font-size: 22px;
  line-height: 1.25;
}

.overview-meta p {
  margin: 6px 0 0;
  color: var(--text-secondary);
}

.repo-link {
  color: var(--brand-color);
  text-decoration: none;
}

.repo-link:hover {
  text-decoration: underline;
}

.download-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.download-actions :deep(.el-button) {
  min-width: 170px;
}

.muted {
  color: var(--text-secondary);
}

@media (max-width: 960px) {
  .detail-page {
    padding: 14px;
  }
}
</style>
