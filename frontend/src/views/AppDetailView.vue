<script setup>
import { computed, onUnmounted, reactive, watch } from "vue";
import { ElMessage } from "element-plus";
import { ArrowLeft, Link, User } from "@element-plus/icons-vue";
import DOMPurify from "dompurify";
import { marked } from "marked";
import { useRoute, useRouter } from "vue-router";
import { GetDownloadTask, StartDownload } from "../../wailsjs/go/main/App";
import StatusState from "../components/StatusState.vue";
import {
  getAppDetailState,
  loadAppDetail,
  loadAppsConfig,
  useAppsStore,
} from "../stores/appsStore";
import { getDownloadDirectory } from "../utils/settings";

const route = useRoute();
const router = useRouter();
const { state, findApp } = useAppsStore();

const platformOptions = [
  { key: "windows", label: "Windows" },
  { key: "linux", label: "Linux" },
  { key: "macos", label: "macOS" },
];

const downloadStates = reactive({
  windows: createDownloadState(),
  linux: createDownloadState(),
  macos: createDownloadState(),
});

const pollerMap = new Map();
const category = computed(() => String(route.params.category || ""));
const appName = computed(() => String(route.params.name || ""));
const app = computed(() => findApp(category.value, appName.value));

const detailState = computed(() => getAppDetailState(app.value));
const detail = computed(() => detailState.value.data);
const downloads = computed(() => detail.value?.downloads || {});

const developer = computed(() => {
  const repo = String(app.value?.repo || "");
  const parts = repo.split("/");
  if (parts.length === 2 && parts[0]) {
    return parts[0];
  }
  return "Unknown";
});

const repoUrl = computed(() => {
  const repo = String(app.value?.repo || "").trim();
  if (!repo) {
    return "";
  }
  return `https://github.com/${repo}`;
});

const releaseTitle = computed(() => {
  if (!detail.value) {
    return "";
  }
  return detail.value.release_name || detail.value.release_tag || "Latest Release";
});

const readmeHtml = computed(() => renderMarkdown(detail.value?.readme));
const releaseNotesHtml = computed(() => renderMarkdown(detail.value?.release_body));

function goBack() {
  const hasRouterBack = Boolean(window.history.state?.back);
  if (hasRouterBack || window.history.length > 1) {
    router.back();
    return;
  }

  const nextQuery = category.value ? { category: category.value } : {};
  router.push({ name: "market", query: nextQuery });
}

function createDownloadState() {
  return {
    taskId: "",
    status: "",
    progress: 0,
    error: "",
  };
}

function resetDownloadStates() {
  platformOptions.forEach(({ key }) => {
    clearTaskPoller(key);
    downloadStates[key].taskId = "";
    downloadStates[key].status = "";
    downloadStates[key].progress = 0;
    downloadStates[key].error = "";
  });
}

function getPlatformDownload(platform) {
  return downloads.value?.[platform] || null;
}

function isPlatformAvailable(platform) {
  return Boolean(getPlatformDownload(platform)?.available);
}

function isTaskBusy(platform) {
  const status = downloadStates[platform].status;
  return status === "started" || status === "in_progress";
}

function downloadButtonLabel(platform) {
  const option = platformOptions.find((item) => item.key === platform);
  const label = option?.label || platform;

  if (!isPlatformAvailable(platform)) {
    return `${label} 不可用`;
  }

  const downloadState = downloadStates[platform];
  if (downloadState.status === "started" || downloadState.status === "in_progress") {
    return `${label} ${downloadState.progress}%`;
  }
  if (downloadState.status === "completed") {
    return `${label} 已下载`;
  }
  if (downloadState.status === "failed") {
    return `${label} 重试下载`;
  }
  return `${label} 下载`;
}

function downloadButtonType(platform) {
  const status = downloadStates[platform].status;
  if (status === "completed") {
    return "success";
  }
  if (status === "failed") {
    return "danger";
  }
  return "primary";
}

function downloadHint(platform) {
  const target = getPlatformDownload(platform);
  if (!target?.available) {
    return "未匹配到该平台可下载资产";
  }
  return `${target.asset_name} (${target.arch || "unknown arch"})`;
}

function applyTaskSnapshot(platform, snapshot) {
  if (!snapshot) {
    return;
  }

  downloadStates[platform].taskId = snapshot.task_id || "";
  downloadStates[platform].status = snapshot.status || "";
  downloadStates[platform].progress = Number(snapshot.progress || 0);
  downloadStates[platform].error = snapshot.error || "";
}

function clearTaskPoller(platform) {
  const timer = pollerMap.get(platform);
  if (timer) {
    clearInterval(timer);
    pollerMap.delete(platform);
  }
}

function startTaskPoller(platform, taskId) {
  clearTaskPoller(platform);

  const timer = setInterval(async () => {
    try {
      const snapshot = await GetDownloadTask(taskId);
      applyTaskSnapshot(platform, snapshot);
      if (snapshot.status === "completed" || snapshot.status === "failed") {
        clearTaskPoller(platform);
        if (snapshot.status === "completed") {
          ElMessage.success(`${platform} 下载完成`);
        } else if (snapshot.error) {
          ElMessage.error(snapshot.error);
        }
      }
    } catch (error) {
      clearTaskPoller(platform);
      const message = error?.message || String(error);
      downloadStates[platform].status = "failed";
      downloadStates[platform].error = message;
      ElMessage.error(message || "下载任务查询失败");
    }
  }, 900);

  pollerMap.set(platform, timer);
}

async function handleDownload(platform) {
  const target = getPlatformDownload(platform);
  if (!target?.available || isTaskBusy(platform)) {
    return;
  }

  const downloadDir = getDownloadDirectory();
  if (!downloadDir) {
    ElMessage.warning("请先在设置中配置下载目录");
    return;
  }

  try {
    const snapshot = await StartDownload({
      download_url: target.download_url,
      file_name: target.asset_name,
      platform,
      download_dir: downloadDir,
    });
    applyTaskSnapshot(platform, snapshot);
    startTaskPoller(platform, snapshot.task_id);
  } catch (error) {
    const message = error?.message || String(error);
    downloadStates[platform].status = "failed";
    downloadStates[platform].error = message;
    ElMessage.error(message || "启动下载任务失败");
  }
}

function renderMarkdown(source) {
  const text = String(source || "").trim();
  if (!text) {
    return "";
  }

  const html = marked.parse(text, { gfm: true, breaks: true });
  return DOMPurify.sanitize(typeof html === "string" ? html : "");
}

loadAppsConfig();

watch(
  () => [app.value?.repo, app.value?.match],
  async ([repo, match]) => {
    resetDownloadStates();
    if (!repo || !match || !app.value) {
      return;
    }
    await loadAppDetail(app.value);
  },
  { immediate: true }
);

onUnmounted(() => {
  resetDownloadStates();
});
</script>

<template>
  <section class="detail-page">
    <div class="detail-toolbar">
      <el-button class="back-button" plain @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
        <span>返回</span>
      </el-button>
    </div>

    <StatusState
      v-if="state.loading"
      type="loading"
      title="正在加载应用详情"
      description="正在读取本地应用配置。"
    />
    <StatusState
      v-else-if="state.error"
      type="error"
      title="应用配置加载失败"
      :description="state.error"
    />
    <StatusState
      v-else-if="!app"
      type="empty"
      title="未找到应用"
      description="请检查路由参数或配置文件。"
    />
    <StatusState
      v-else-if="detailState.loading && !detailState.loaded"
      type="loading"
      title="正在拉取 Release 与 README"
      description="正在从 GitHub 获取最新版本信息。"
    />
    <StatusState
      v-else-if="detailState.error && !detailState.loaded"
      type="error"
      title="详情加载失败"
      :description="detailState.error"
    />

    <div v-else class="detail-layout">
      <el-card class="overview-card" shadow="never">
        <div class="overview-head">
          <img class="app-icon" :src="app.photo" :alt="app.name" />

          <div class="overview-meta">
            <h2>{{ app.name }}</h2>
            <p class="app-summary">{{ app.summary }}</p>

            <div class="meta-inline">
              <span class="meta-item">
                <el-icon><User /></el-icon>
                {{ developer }}
              </span>
              <a class="repo-link" :href="repoUrl" target="_blank" rel="noreferrer">
                <el-icon><Link /></el-icon>
                {{ app.repo }}
              </a>
            </div>

            <p class="release-title">最新版本：{{ releaseTitle }}</p>
          </div>
        </div>

        <div class="download-actions">
          <div v-for="item in platformOptions" :key="item.key" class="download-item">
            <el-button
              :type="downloadButtonType(item.key)"
              :loading="isTaskBusy(item.key)"
              :disabled="!isPlatformAvailable(item.key) || isTaskBusy(item.key)"
              @click="handleDownload(item.key)"
            >
              {{ downloadButtonLabel(item.key) }}
            </el-button>
            <div class="download-hint">{{ downloadHint(item.key) }}</div>
          </div>
        </div>
      </el-card>

      <el-alert
        v-if="detailState.error && detailState.loaded"
        type="warning"
        :closable="false"
        :title="`部分内容加载失败：${detailState.error}`"
      />

      <el-card class="markdown-card" shadow="never">
        <template #header>README</template>
        <div v-if="readmeHtml" class="markdown-body" v-html="readmeHtml" />
        <el-empty v-else description="暂无 README 内容" :image-size="80" />
      </el-card>

      <el-card class="markdown-card" shadow="never">
        <template #header>Release Notes</template>
        <div v-if="releaseNotesHtml" class="markdown-body" v-html="releaseNotesHtml" />
        <el-empty v-else description="暂无版本说明" :image-size="80" />
      </el-card>
    </div>
  </section>
</template>

<style scoped>
.detail-page {
  height: 100%;
  padding: 18px;
  overflow-y: auto;
  overflow-x: hidden;
}

.detail-toolbar {
  margin-bottom: 12px;
  position: sticky;
  top: 0;
  z-index: 3;
  padding-bottom: 6px;
  background: linear-gradient(180deg, #ffffff 76%, rgba(255, 255, 255, 0));
}

.back-button {
  gap: 6px;
  font-weight: 600;
  border-color: #cfe4d6;
  color: var(--text-secondary);
}

.back-button:hover {
  color: var(--brand-color);
  border-color: rgba(5, 150, 105, 0.35);
}

.detail-layout {
  min-height: 100%;
  display: grid;
  gap: 14px;
  align-content: start;
}

.overview-card,
.markdown-card {
  border-radius: var(--radius-lg);
  border-color: var(--line-color);
}

.overview-card :deep(.el-card__body) {
  background: linear-gradient(180deg, #ffffff, #f8fdf9);
}

.overview-head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.app-icon {
  width: 72px;
  height: 72px;
  border-radius: var(--radius-md);
  object-fit: cover;
  background: var(--surface-3);
  border: 1px solid #dbeee2;
}

.overview-meta {
  min-width: 0;
}

.overview-meta h2 {
  margin: 0;
  font-size: 26px;
  line-height: 1.2;
}

.app-summary {
  margin: 6px 0 0;
  color: var(--text-secondary);
}

.meta-inline {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--text-secondary);
}

.repo-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--brand-color);
  text-decoration: none;
}

.repo-link:hover {
  text-decoration: underline;
}

.release-title {
  margin: 10px 0 0;
  font-size: 13px;
  color: var(--text-tertiary);
}

.download-actions {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 10px;
}

.download-item {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-md);
  background: rgba(236, 253, 245, 0.55);
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.download-item :deep(.el-button) {
  justify-content: flex-start;
  font-weight: 600;
}

.download-hint {
  min-height: 34px;
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
}

.markdown-card :deep(.el-card__header) {
  font-weight: 700;
  color: var(--text-primary);
}

.markdown-body {
  color: var(--text-primary);
  line-height: 1.7;
  word-break: break-word;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 16px 0 10px;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  margin: 8px 0;
}

.markdown-body :deep(pre) {
  overflow-x: auto;
  padding: 10px;
  border-radius: var(--radius-md);
  background: #0f172a;
  color: #e2e8f0;
}

.markdown-body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.markdown-body :deep(a) {
  color: var(--brand-color);
}

@media (max-width: 960px) {
  .detail-page {
    padding: 14px;
  }

  .overview-head {
    flex-direction: column;
  }

  .overview-meta h2 {
    font-size: 22px;
  }
}
</style>
