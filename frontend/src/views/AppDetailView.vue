<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Download, Link, Star, User } from "@element-plus/icons-vue";
import DOMPurify from "dompurify";
import { marked } from "marked";
import { useRoute } from "vue-router";
import { GetDownloadTask, StartDownload } from "../../wailsjs/go/main/App";
import StatusState from "../components/StatusState.vue";
import {
  getAppDetailState,
  loadAppDetail,
  loadAppsConfig,
  loadRepoStars,
  useAppsStore,
} from "../stores/appsStore";
import { getDownloadDirectory } from "../utils/settings";
import { getCategoryDisplayName } from "../utils/marketMeta";

const route = useRoute();
const { state, findApp, getRepoStarCount } = useAppsStore();

const releaseAssetsPanelRef = ref(null);
const selectedAssetKey = ref("");
let taskPoller = null;

const downloadTask = reactive({
  task_id: "",
  status: "",
  progress: 0,
  error: "",
  file_name: "",
  platform: "",
});

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
  return "unknown";
});

const iconSrc = computed(() => {
  const photo = String(app.value?.photo || "").trim();
  if (photo) {
    return photo;
  }
  return `https://avatars.githubusercontent.com/${developer.value}`;
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
    return "-";
  }
  return detail.value.release_name || detail.value.release_tag || "Latest Release";
});

const releasePublishedLabel = computed(() => formatDate(detail.value?.release_published_at));
const starsLabel = computed(() => formatStarCount(getRepoStarCount(app.value?.repo)));

const availablePlatformCount = computed(() =>
  Object.values(downloads.value || {}).filter((item) => item?.available).length
);

const readmeHtml = computed(() => renderMarkdown(detail.value?.readme));
const releaseNotesHtml = computed(() => renderMarkdown(detail.value?.release_body));

const contributors = computed(() => {
  if (!Array.isArray(detail.value?.contributors)) {
    return [];
  }
  return detail.value.contributors.slice(0, 4);
});

const releaseAssets = computed(() => {
  if (!Array.isArray(detail.value?.release_assets)) {
    return [];
  }
  return detail.value.release_assets.map((asset, index) => ({
    ...asset,
    key: `${asset.name || "asset"}-${index}`,
  }));
});

const selectedAsset = computed(() =>
  releaseAssets.value.find((item) => item.key === selectedAssetKey.value) || null
);

const cdnMeta = computed(() => {
  const meta = detail.value?.cdn_meta;
  if (!meta || typeof meta !== "object") {
    return {
      enabled: false,
      selected_source: "",
      label: "直连",
    };
  }

  return {
    enabled: Boolean(meta.enabled),
    selected_source: String(meta.selected_source || ""),
    label: String(meta.label || (meta.enabled ? "CDN" : "直连")),
  };
});

const cdnBadgeText = computed(() =>
  cdnMeta.value.enabled ? `CDN已启用 (${cdnMeta.value.label})` : "直连"
);

function resetDownloadTask() {
  clearTaskPoller();
  downloadTask.task_id = "";
  downloadTask.status = "";
  downloadTask.progress = 0;
  downloadTask.error = "";
  downloadTask.file_name = "";
  downloadTask.platform = "";
}

function isDownloadBusy() {
  return downloadTask.status === "started" || downloadTask.status === "in_progress";
}

function applyTaskSnapshot(snapshot) {
  if (!snapshot) {
    return;
  }
  downloadTask.task_id = snapshot.task_id || "";
  downloadTask.status = snapshot.status || "";
  downloadTask.progress = Number(snapshot.progress || 0);
  downloadTask.error = snapshot.error || "";
  downloadTask.file_name = snapshot.file_name || "";
  downloadTask.platform = snapshot.platform || "";
}

function clearTaskPoller() {
  if (taskPoller) {
    clearInterval(taskPoller);
    taskPoller = null;
  }
}

function startTaskPoller(taskId) {
  clearTaskPoller();

  taskPoller = setInterval(async () => {
    try {
      const snapshot = await GetDownloadTask(taskId);
      applyTaskSnapshot(snapshot);

      if (snapshot.status === "completed" || snapshot.status === "failed") {
        clearTaskPoller();
        if (snapshot.status === "completed") {
          ElMessage.success("下载完成");
        } else if (snapshot.error) {
          ElMessage.error(snapshot.error);
        }
      }
    } catch (error) {
      clearTaskPoller();
      const message = error?.message || String(error);
      downloadTask.status = "failed";
      downloadTask.error = message;
      ElMessage.error(message || "下载任务查询失败");
    }
  }, 900);
}

async function startDownloadForAsset(asset, downloadDir) {
  if (!asset || isDownloadBusy()) {
    return;
  }

  const platform = ["windows", "linux", "macos"].includes(asset.platform) ? asset.platform : "";

  try {
    const snapshot = await StartDownload({
      download_url: asset.download_url,
      file_name: asset.name || "release-asset",
      platform,
      download_dir: downloadDir,
    });
    applyTaskSnapshot(snapshot);
    startTaskPoller(snapshot.task_id);
  } catch (error) {
    const message = error?.message || String(error);
    downloadTask.status = "failed";
    downloadTask.error = message;
    ElMessage.error(message || "启动下载任务失败");
  }
}

async function scrollToReleaseAssets() {
  await nextTick();
  if (releaseAssetsPanelRef.value && typeof releaseAssetsPanelRef.value.scrollIntoView === "function") {
    releaseAssetsPanelRef.value.scrollIntoView({
      behavior: "smooth",
      block: "center",
      inline: "nearest",
    });
  }
}

function buildLocalTargetPath(downloadDir, fileName) {
  const dir = String(downloadDir || "").trim();
  const name = String(fileName || "release-asset").trim() || "release-asset";
  if (!dir) {
    return name;
  }

  const normalizedDir = dir.endsWith("\\") || dir.endsWith("/") ? dir.slice(0, -1) : dir;
  const separator = normalizedDir.includes("\\") ? "\\" : "/";
  return `${normalizedDir}${separator}${name}`;
}

function escapeHtml(value) {
  return String(value || "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

async function selectAsset(asset) {
  if (!asset) {
    return;
  }

  selectedAssetKey.value = asset.key;

  if (isDownloadBusy()) {
    ElMessage.warning("当前有下载任务进行中，请稍后重试。");
    return;
  }

  const downloadDir = getDownloadDirectory();
  if (!downloadDir) {
    ElMessage.warning("请先在设置中配置下载目录");
    return;
  }

  const localTargetPath = buildLocalTargetPath(downloadDir, asset.name || "release-asset");
  const message = `
    <div style="line-height:1.7">
      <div><strong>目标文件：</strong>${escapeHtml(asset.name || "release-asset")}</div>
      <div><strong>下载地址：</strong></div>
      <div style="word-break:break-all;color:#475569;">${escapeHtml(localTargetPath)}</div>
      <div style="margin-top:6px;"><strong>链路：</strong>${escapeHtml(cdnBadgeText.value)}</div>
    </div>
  `;

  try {
    await ElMessageBox.confirm(message, "确认下载", {
      confirmButtonText: "下载",
      cancelButtonText: "取消",
      dangerouslyUseHTMLString: true,
      type: "info",
    });
    await startDownloadForAsset(asset, downloadDir);
  } catch {
    // User cancelled.
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

function formatDate(value) {
  const raw = String(value || "").trim();
  if (!raw) {
    return "-";
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw;
  }

  return date.toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatStarCount(value) {
  if (typeof value !== "number" || Number.isNaN(value) || value < 0) {
    return "-";
  }
  if (value < 1000) {
    return String(value);
  }
  if (value < 10000) {
    return `${(value / 1000).toFixed(1)}k`;
  }
  if (value < 1000000) {
    return `${Math.round(value / 1000)}k`;
  }
  return `${(value / 1000000).toFixed(1)}m`;
}

function formatAssetSize(size) {
  const bytes = Number(size || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "-";
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatDownloadCount(value) {
  const count = Number(value || 0);
  if (!Number.isFinite(count) || count < 0) {
    return "-";
  }
  return count.toLocaleString("en-US");
}

onMounted(async () => {
  await loadAppsConfig();
  if (!state.repoStarsLoaded && !state.repoStarsLoading) {
    void loadRepoStars();
  }
});

watch(
  releaseAssets,
  (assets) => {
    if (!assets.length) {
      selectedAssetKey.value = "";
      return;
    }

    if (!assets.some((asset) => asset.key === selectedAssetKey.value)) {
      selectedAssetKey.value = assets[0].key;
    }
  },
  { immediate: true }
);

watch(
  () => [app.value?.repo, app.value?.match],
  async ([repo, match]) => {
    resetDownloadTask();
    if (!repo || !match || !app.value) {
      return;
    }
    await loadAppDetail(app.value);
  },
  { immediate: true }
);

onUnmounted(() => {
  resetDownloadTask();
});
</script>

<template>
  <section class="detail-page">
    <StatusState
      v-if="state.loading"
      type="loading"
      title="正在加载应用详情"
      description="正在读取应用配置，请稍候。"
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
      <section class="hero-section clean-card">
        <div class="hero-main">
          <img class="hero-icon" :src="iconSrc" :alt="app.name" />

          <div class="hero-meta">
            <h1>{{ app.name }}</h1>
            <p class="hero-owner">by @{{ developer }}</p>
            <div class="hero-tags">
              <span class="tag">{{ getCategoryDisplayName(category) }}</span>
              <span class="tag muted">{{ releaseTitle }}</span>
            </div>
            <p class="hero-summary">{{ app.summary }}</p>
          </div>
        </div>

        <div class="hero-actions">
          <el-button class="focus-release-btn" type="primary" @click="scrollToReleaseAssets">
            <el-icon><Download /></el-icon>
            <span>选择 Release 包</span>
          </el-button>

          <a class="source-btn" :href="repoUrl" target="_blank" rel="noreferrer">
            <el-icon><Link /></el-icon>
            <span>View Source</span>
          </a>

          <div class="cdn-badge" :class="{ enabled: cdnMeta.enabled }">
            {{ cdnBadgeText }}
          </div>
        </div>
      </section>

      <section class="metrics-row">
        <article class="metric-item clean-card">
          <el-icon><Star /></el-icon>
          <div>
            <p class="metric-label">Stars</p>
            <p class="metric-value">{{ starsLabel }}</p>
          </div>
        </article>
        <article class="metric-item clean-card">
          <el-icon><User /></el-icon>
          <div>
            <p class="metric-label">Author</p>
            <p class="metric-value">@{{ developer }}</p>
          </div>
        </article>
        <article class="metric-item clean-card">
          <el-icon><Link /></el-icon>
          <div>
            <p class="metric-label">Latest</p>
            <p class="metric-value ellipsis">{{ releaseTitle }}</p>
          </div>
        </article>
        <article class="metric-item clean-card">
          <el-icon><Star /></el-icon>
          <div>
            <p class="metric-label">Platforms</p>
            <p class="metric-value">{{ availablePlatformCount }}</p>
          </div>
        </article>
      </section>

      <section class="content-grid">
        <div class="left-column">
          <article class="panel clean-card">
            <header class="panel-head">
              <h2>README.md</h2>
              <a class="panel-link" :href="repoUrl" target="_blank" rel="noreferrer">GitHub</a>
            </header>

            <div v-if="readmeHtml" class="markdown-body" v-html="readmeHtml" />
            <el-empty v-else description="暂无 README 内容" :image-size="80" />
          </article>

          <article class="panel clean-card">
            <header class="panel-head">
              <h2>Version History</h2>
            </header>

            <div class="version-item">
              <div class="version-head">
                <strong>{{ releaseTitle }}</strong>
                <span class="latest-pill">Latest</span>
              </div>
              <p class="version-date">{{ releasePublishedLabel }}</p>
              <div v-if="releaseNotesHtml" class="markdown-body" v-html="releaseNotesHtml" />
              <el-empty v-else description="暂无版本说明" :image-size="72" />
            </div>
          </article>
        </div>

        <aside class="right-column">
          <article class="side-panel clean-card">
            <h3>Contributors</h3>
            <div v-if="contributors.length" class="contributors-list">
              <a
                v-for="contributor in contributors"
                :key="`${contributor.login}-${contributor.profile_url}`"
                class="contributor-item"
                :href="contributor.profile_url || '#'"
                target="_blank"
                rel="noreferrer"
              >
                <img class="contributor-avatar" :src="contributor.avatar_url" :alt="contributor.display_name" />
                <div class="contributor-meta">
                  <strong>{{ contributor.display_name }}</strong>
                  <span>{{ contributor.contributions }} commits</span>
                </div>
              </a>
            </div>
            <el-empty v-else description="暂无贡献者信息" :image-size="72" />
          </article>

          <article ref="releaseAssetsPanelRef" class="side-panel clean-card">
            <h3>Latest Release Assets</h3>
            <div v-if="releaseAssets.length" class="asset-list">
              <button
                v-for="asset in releaseAssets"
                :key="asset.key"
                class="asset-item"
                :class="{ active: selectedAsset?.key === asset.key }"
                @click="selectAsset(asset)"
              >
                <div class="asset-head">
                  <strong class="asset-name">{{ asset.name }}</strong>
                  <span class="asset-cdn" :class="{ enabled: cdnMeta.enabled }">{{ cdnBadgeText }}</span>
                </div>
                <div class="asset-meta">
                  <span>大小 {{ formatAssetSize(asset.size) }}</span>
                  <span>下载 {{ formatDownloadCount(asset.download_count) }}</span>
                </div>
              </button>
            </div>
            <el-empty v-else description="当前版本暂无可下载资产" :image-size="72" />
          </article>

          <div v-if="downloadTask.status || downloadTask.error" class="asset-download-status">
            <p>
              下载状态：{{ downloadTask.status || "idle" }}
              <span v-if="isDownloadBusy()">({{ downloadTask.progress }}%)</span>
            </p>
            <p v-if="downloadTask.error" class="download-error">{{ downloadTask.error }}</p>
          </div>

          <el-alert
            v-if="detailState.error && detailState.loaded"
            type="warning"
            :closable="false"
            :title="`部分内容加载失败：${detailState.error}`"
          />
        </aside>
      </section>
    </div>
  </section>
</template>

<style scoped>
.detail-page {
  height: 100%;
  padding: 18px;
  overflow: auto;
  background: var(--surface-page);
}

.detail-layout {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.detail-layout > section + section {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--line-color);
}

.clean-card {
  border: none;
  border-radius: 2px;
  background: var(--surface-container);
}

.hero-section {
  padding: 22px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 24px;
}

.hero-main {
  display: flex;
  gap: 18px;
  min-width: 0;
}

.hero-icon {
  width: 96px;
  height: 96px;
  border-radius: 18px;
  object-fit: cover;
  border: 1px solid var(--line-color);
  background: var(--surface-3);
}

.hero-meta {
  min-width: 0;
}

.hero-meta h1 {
  margin: 0;
  font-size: 44px;
  line-height: 1.05;
  letter-spacing: 0.2px;
}

.hero-owner {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 16px;
}

.hero-tags {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  background: rgba(217, 72, 15, 0.12);
  color: var(--brand-color);
}

.tag.muted {
  background: var(--surface-3);
  color: #64748b;
}

.hero-summary {
  margin: 14px 0 0;
  color: var(--text-secondary);
  line-height: 1.7;
}

.hero-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.focus-release-btn,
.source-btn {
  height: 44px;
  border-radius: 2px;
  font-weight: 700;
}

.focus-release-btn .el-icon {
  margin-right: 6px;
}

.source-btn {
  border: none;
  text-decoration: none;
  color: #334155;
  background: var(--surface-control);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.source-btn:hover {
  color: var(--brand-color);
  background: var(--brand-color-soft);
}

.cdn-badge {
  align-self: flex-start;
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
  color: #64748b;
  background: var(--surface-3);
  font-weight: 700;
}

.cdn-badge.enabled {
  color: var(--brand-color);
  background: rgba(217, 72, 15, 0.12);
}

.metrics-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.metric-item {
  padding: 14px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.metric-item .el-icon {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: rgba(217, 72, 15, 0.12);
  color: var(--brand-color);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.metric-label {
  margin: 0;
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: #94a3b8;
  font-weight: 700;
}

.metric-value {
  margin: 4px 0 0;
  font-size: 20px;
  font-weight: 800;
  color: #0f172a;
}

.metric-value.ellipsis {
  max-width: 210px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 0;
}

.left-column,
.right-column {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.left-column {
  padding-right: 16px;
}

.right-column {
  padding-left: 16px;
  border-left: 1px solid var(--line-color);
}

.left-column > .panel + .panel {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--line-color);
}

.right-column > * + * {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--line-color);
}

.panel,
.side-panel {
  padding: 18px;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--line-color);
  padding-bottom: 12px;
  margin-bottom: 12px;
}

.panel-head h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.15;
}

.panel-link {
  color: var(--brand-color);
  font-weight: 700;
  text-decoration: none;
}

.panel-link:hover {
  text-decoration: underline;
}

.version-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.version-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.version-head strong {
  font-size: 20px;
}

.latest-pill {
  border-radius: 999px;
  background: var(--brand-color);
  color: #fff;
  padding: 3px 8px;
  font-size: 10px;
  text-transform: uppercase;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.version-date {
  margin: 0;
  color: #64748b;
  font-size: 13px;
  font-weight: 600;
}

.side-panel h3 {
  margin: 0 0 12px;
  font-size: 20px;
}

.contributors-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.contributor-item {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: inherit;
}

.contributor-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--line-color);
}

.contributor-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.contributor-meta strong {
  color: #0f172a;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contributor-meta span {
  color: #64748b;
  font-size: 12px;
}

.asset-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.asset-item {
  width: 100%;
  border: none;
  border-radius: 2px;
  background: transparent;
  padding: 10px 8px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s ease;
}

.asset-item + .asset-item {
  border-top: 1px solid var(--line-color);
}

.asset-item:hover {
  background: rgba(217, 72, 15, 0.04);
}

.asset-item.active {
  background: rgba(217, 72, 15, 0.06);
}

.asset-item.active .asset-name {
  color: var(--brand-color);
}

.asset-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.asset-name {
  color: #0f172a;
  font-size: 13px;
  line-height: 1.35;
  word-break: break-word;
}

.asset-cdn {
  flex-shrink: 0;
  border-radius: 999px;
  background: var(--surface-3);
  color: #64748b;
  padding: 2px 8px;
  font-size: 10px;
  font-weight: 700;
}

.asset-cdn.enabled {
  background: rgba(217, 72, 15, 0.15);
  color: var(--brand-color);
}

.asset-meta {
  margin-top: 7px;
  display: flex;
  gap: 12px;
  color: #64748b;
  font-size: 12px;
}

.asset-download-status {
  border: none;
  border-radius: 2px;
  background: var(--surface-container);
  padding: 10px 12px;
  color: #64748b;
  font-size: 12px;
}

.asset-download-status p {
  margin: 0;
}

.asset-download-status p + p {
  margin-top: 6px;
}

.download-error {
  margin: 0;
  color: #dc2626;
  font-size: 12px;
  line-height: 1.4;
}

.markdown-body {
  color: var(--text-primary);
  line-height: 1.75;
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 16px 0 10px;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(li),
.markdown-body :deep(blockquote),
.markdown-body :deep(td),
.markdown-body :deep(th) {
  margin: 8px 0;
  overflow-wrap: anywhere;
}

.markdown-body :deep(pre) {
  max-width: 100%;
  overflow-x: auto;
  white-space: pre;
  padding: 12px;
  border-radius: 2px;
  background: #0f172a;
  color: #e2e8f0;
}

.markdown-body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.markdown-body :deep(a) {
  color: var(--brand-color);
  overflow-wrap: anywhere;
}

.markdown-body :deep(img),
.markdown-body :deep(picture),
.markdown-body :deep(video),
.markdown-body :deep(canvas),
.markdown-body :deep(svg),
.markdown-body :deep(iframe) {
  max-width: 100%;
  height: auto;
}

.markdown-body :deep(img),
.markdown-body :deep(picture > img),
.markdown-body :deep(video),
.markdown-body :deep(canvas),
.markdown-body :deep(svg) {
  display: block;
}

.markdown-body :deep(table) {
  display: block;
  max-width: 100%;
  width: max-content;
  overflow-x: auto;
  border-collapse: collapse;
}

@media (max-width: 1280px) {
  .hero-section {
    grid-template-columns: 1fr;
  }

  .metrics-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .left-column,
  .right-column {
    padding-left: 0;
    padding-right: 0;
  }

  .right-column {
    border-left: none;
  }
}

@media (max-width: 900px) {
  .detail-page {
    padding: 12px;
  }

  .hero-main {
    flex-direction: column;
  }

  .hero-meta h1 {
    font-size: 32px;
  }

  .panel-head h2 {
    font-size: 24px;
  }

  .metrics-row {
    grid-template-columns: 1fr;
  }
}
</style>
