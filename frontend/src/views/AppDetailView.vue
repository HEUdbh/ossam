<script setup>
import { computed, onUnmounted, reactive, watch } from "vue";
import { ElMessage } from "element-plus";
import { Link, User } from "@element-plus/icons-vue";
import DOMPurify from "dompurify";
import { marked } from "marked";
import { useRoute } from "vue-router";
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
  return detail.value.release_name || detail.value.release_tag || "Latest release";
});

const readmeHtml = computed(() => renderMarkdown(detail.value?.readme));
const releaseNotesHtml = computed(() => renderMarkdown(detail.value?.release_body));

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
    return `${label} unavailable`;
  }

  const downloadState = downloadStates[platform];
  if (downloadState.status === "started" || downloadState.status === "in_progress") {
    return `${label} ${downloadState.progress}%`;
  }
  if (downloadState.status === "completed") {
    return `${label} downloaded`;
  }
  if (downloadState.status === "failed") {
    return `${label} retry`;
  }
  return `${label} download`;
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
    return "No matched release asset for this platform.";
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
          ElMessage.success(`${platform} download completed`);
        } else if (snapshot.error) {
          ElMessage.error(snapshot.error);
        }
      }
    } catch (error) {
      clearTaskPoller(platform);
      const message = error?.message || String(error);
      downloadStates[platform].status = "failed";
      downloadStates[platform].error = message;
      ElMessage.error(message || "Failed to query download task");
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
    ElMessage.warning("Please configure a download directory in Settings first.");
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
    ElMessage.error(message || "Failed to start download task");
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
    <StatusState
      v-if="state.loading"
      type="loading"
      title="Loading app detail"
      description="Reading local app configuration."
    />
    <StatusState
      v-else-if="state.error"
      type="error"
      title="Failed to load app config"
      :description="state.error"
    />
    <StatusState
      v-else-if="!app"
      type="empty"
      title="App not found"
      description="Please check route params and appsconfig.json."
    />
    <StatusState
      v-else-if="detailState.loading && !detailState.loaded"
      type="loading"
      title="Loading release and README"
      description="Fetching latest release metadata from GitHub."
    />
    <StatusState
      v-else-if="detailState.error && !detailState.loaded"
      type="error"
      title="Failed to load detail"
      :description="detailState.error"
    />

    <div v-else class="detail-layout">
      <el-card class="overview-card" shadow="never">
        <div class="overview-head">
          <img class="app-icon" :src="app.photo" :alt="app.name" />
          <div class="overview-meta">
            <h2>{{ app.name }}</h2>
            <p>{{ category }}</p>
            <div class="meta-inline">
              <el-space :size="14">
                <span class="meta-item">
                  <el-icon><User /></el-icon>
                  {{ developer }}
                </span>
                <a class="repo-link" :href="repoUrl" target="_blank" rel="noreferrer">
                  <el-icon><Link /></el-icon>
                  {{ app.repo }}
                </a>
              </el-space>
            </div>
            <p v-if="releaseTitle" class="release-title">Release: {{ releaseTitle }}</p>
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
        :title="`Partial load warning: ${detailState.error}`"
      />

      <el-card class="markdown-card" shadow="never">
        <template #header>
          <span>README</span>
        </template>
        <div v-if="readmeHtml" class="markdown-body" v-html="readmeHtml" />
        <el-empty v-else description="README not available" :image-size="80" />
      </el-card>

      <el-card class="markdown-card" shadow="never">
        <template #header>
          <span>Release Notes</span>
        </template>
        <div v-if="releaseNotesHtml" class="markdown-body" v-html="releaseNotesHtml" />
        <el-empty v-else description="Release notes not available" :image-size="80" />
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
  min-height: 100%;
  display: grid;
  gap: 14px;
  align-content: start;
}

.overview-card,
.markdown-card {
  border-radius: var(--radius-md);
  border-color: var(--line-color);
}

.overview-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.app-icon {
  width: 68px;
  height: 68px;
  border-radius: 14px;
  object-fit: cover;
  background: var(--surface-3);
}

.overview-meta h2 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
}

.overview-meta p {
  margin: 6px 0 0;
  color: var(--text-secondary);
}

.meta-inline {
  margin-top: 6px;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.release-title {
  margin-top: 6px;
  font-size: 13px;
  color: var(--text-tertiary);
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

.download-actions {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.download-item {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-sm);
  padding: 10px;
  background: var(--surface-2);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.download-item :deep(.el-button) {
  justify-content: flex-start;
}

.download-hint {
  min-height: 34px;
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
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
  border-radius: 10px;
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
    align-items: flex-start;
  }
}
</style>
