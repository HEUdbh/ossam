<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Search } from "@element-plus/icons-vue";
import StatusState from "../components/StatusState.vue";
import { loadAppsConfig, loadRepoStars, useAppsStore } from "../stores/appsStore";

const route = useRoute();
const router = useRouter();
const { state, categories, appsByCategory, getRepoStarCount } = useAppsStore();

const selectedCategory = ref("");
const searchQuery = ref("");
const categoryLabels = {
  Security: "安全工具",
  DevTools: "开发利器",
  System: "系统增强",
  Network: "网络插件",
  Database: "数据管理",
  Terminal: "终端效率",
  Utility: "效率办公",
};

const normalizedSearch = computed(() => String(searchQuery.value || "").trim().toLowerCase());
const isSearchMode = computed(() => normalizedSearch.value.length > 0);

const currentCategoryApps = computed(() => {
  const apps = appsByCategory(selectedCategory.value);
  return apps.map((app) => ({ app, category: selectedCategory.value }));
});

const allApps = computed(() => {
  const result = [];
  categories.value.forEach((category) => {
    appsByCategory(category).forEach((app) => {
      result.push({ app, category });
    });
  });
  return result;
});

const filteredApps = computed(() => {
  if (!isSearchMode.value) {
    return currentCategoryApps.value;
  }

  const keyword = normalizedSearch.value;
  return allApps.value.filter(({ app, category }) => {
    const haystack = [
      app.name,
      app.summary,
      app.repo,
      category,
      getCategoryDisplayName(category),
    ]
      .map((item) => String(item || "").toLowerCase())
      .join(" ");
    return haystack.includes(keyword);
  });
});

const emptyTitle = computed(() =>
  isSearchMode.value ? "没有找到匹配的应用" : "当前分类暂无应用"
);
const emptyDescription = computed(() =>
  isSearchMode.value
    ? "请尝试更换关键词，支持按名称、简介、仓库地址进行搜索。"
    : "可在 appsconfig.json 中补充该分类的应用条目。"
);

function getCategoryDisplayName(category) {
  const normalized = String(category || "").trim();
  return categoryLabels[normalized] || normalized;
}

function syncSelectedCategory() {
  if (!categories.value.length) {
    selectedCategory.value = "";
    return;
  }

  const categoryFromQuery =
    typeof route.query.category === "string" ? route.query.category : "";
  if (categoryFromQuery && categories.value.includes(categoryFromQuery)) {
    selectedCategory.value = categoryFromQuery;
    return;
  }

  selectedCategory.value = categories.value[0];
  router.replace({
    name: "market",
    query: { category: selectedCategory.value },
  });
}

function chooseCategory(category) {
  if (!category || category === selectedCategory.value) {
    return;
  }
  router.push({ name: "market", query: { category } });
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

function starLabel(repo) {
  return formatStarCount(getRepoStarCount(repo));
}

async function initializeMarket() {
  await loadAppsConfig();
  syncSelectedCategory();
  void loadRepoStars();
}

onMounted(() => {
  initializeMarket();
});

watch(
  () => route.query.category,
  () => {
    syncSelectedCategory();
  }
);

watch(categories, () => {
  syncSelectedCategory();
});
</script>

<template>
  <div class="market-layout">
    <aside class="category-sidebar">
      <div class="sidebar-title">分类浏览</div>
      <el-scrollbar class="category-scroll">
        <button
          v-for="category in categories"
          :key="category"
          class="category-button"
          :class="{ active: category === selectedCategory }"
          @click="chooseCategory(category)"
        >
          <span class="category-dot" />
          <span class="category-text">{{ getCategoryDisplayName(category) }}</span>
        </button>
      </el-scrollbar>
    </aside>

    <section class="apps-panel">
      <header class="apps-header">
        <div class="apps-heading">
          <h2>{{ state.config?.market_name || "应用市场" }}</h2>
          <p>发现并探索高质量开源项目</p>
        </div>

        <div class="apps-actions">
          <el-input
            v-model="searchQuery"
            clearable
            class="search-input"
            placeholder="搜索名称、简介或仓库"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-tag round effect="plain" type="success">
            更新：{{ state.config?.last_updated || "-" }}
          </el-tag>
        </div>
      </header>

      <el-alert
        v-if="state.repoStarsError"
        class="stars-alert"
        type="warning"
        :closable="false"
        title="Star 数据更新失败，已自动回退缓存值。"
      />

      <StatusState
        v-if="state.loading"
        type="loading"
        title="正在加载应用清单"
        description="正在读取本地配置文件，请稍候。"
      />
      <StatusState
        v-else-if="state.error"
        type="error"
        title="配置加载失败"
        :description="state.error"
      />
      <StatusState
        v-else-if="!filteredApps.length"
        type="empty"
        :title="emptyTitle"
        :description="emptyDescription"
      />

      <div v-else class="app-grid">
        <router-link
          v-for="item in filteredApps"
          :key="`${item.category}-${item.app.name}`"
          class="app-card"
          :to="{
            name: 'app-detail',
            params: { category: item.category, name: item.app.name },
          }"
        >
          <img class="app-icon" :src="item.app.photo" :alt="item.app.name" />
          <div class="app-body">
            <div class="app-header">
              <h3 class="app-name">{{ item.app.name }}</h3>
              <el-tag
                v-if="isSearchMode"
                size="small"
                effect="plain"
                type="success"
                class="category-badge"
              >
                {{ getCategoryDisplayName(item.category) }}
              </el-tag>
            </div>
            <p class="app-summary">{{ item.app.summary }}</p>
            <div class="app-meta">
              <span class="app-star">★ {{ starLabel(item.app.repo) }}</span>
            </div>
          </div>
        </router-link>
      </div>
    </section>
  </div>
</template>

<style scoped>
.market-layout {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  background: var(--surface-1);
}

.category-sidebar {
  border-right: 1px solid var(--line-color);
  background: var(--surface-2);
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sidebar-title {
  padding: 0 8px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.category-scroll {
  min-height: 0;
}

.category-button {
  width: 100%;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px;
  text-align: left;
  cursor: pointer;
  transition:
    color var(--duration-fast) ease,
    background-color var(--duration-fast) ease,
    border-color var(--duration-fast) ease;
}

.category-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: transparent;
  transition: background-color var(--duration-fast) ease;
}

.category-button:hover {
  color: var(--brand-color);
  background: var(--brand-color-soft);
  border-color: rgba(5, 150, 105, 0.18);
}

.category-button:hover .category-dot {
  background: rgba(5, 150, 105, 0.4);
}

.category-button.active {
  color: var(--brand-color);
  background: var(--brand-color-weak);
  border-color: rgba(5, 150, 105, 0.3);
  font-weight: 600;
}

.category-button.active .category-dot {
  background: var(--brand-color);
}

.category-text {
  min-width: 0;
}

.apps-panel {
  min-height: 0;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.apps-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.apps-heading h2 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
}

.apps-heading p {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.apps-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-input {
  width: 320px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  box-shadow: 0 0 0 1px #d5e8da inset;
}

.stars-alert {
  margin-bottom: -2px;
}

.app-grid {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 14px;
  align-content: start;
  padding-right: 4px;
}

.app-card {
  border: 1px solid var(--line-color);
  border-radius: 18px;
  background: linear-gradient(180deg, #ffffff 0%, #fbfefb 100%);
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.05);
  text-decoration: none;
  color: var(--text-primary);
  padding: 16px;
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  gap: 12px;
  transition:
    transform var(--duration-fast) ease,
    box-shadow var(--duration-fast) ease,
    border-color var(--duration-fast) ease;
}

.app-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-card);
  border-color: rgba(5, 150, 105, 0.3);
}

.app-icon {
  width: 58px;
  height: 58px;
  border-radius: 14px;
  object-fit: cover;
  background: var(--surface-3);
  border: 1px solid #e2efe6;
}

.app-body {
  min-width: 0;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.app-name {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.2;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.category-badge {
  flex-shrink: 0;
}

.app-summary {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.app-meta {
  margin-top: 10px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.app-star {
  color: var(--warning-color);
  font-size: 13px;
  font-weight: 700;
}

@media (max-width: 1279px) {
  .market-layout {
    grid-template-columns: 188px minmax(0, 1fr);
  }

  .search-input {
    width: 260px;
  }
}

@media (max-width: 960px) {
  .market-layout {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr);
  }

  .category-sidebar {
    border-right: none;
    border-bottom: 1px solid var(--line-color);
    padding: 12px;
  }

  .category-scroll :deep(.el-scrollbar__view) {
    display: flex;
    gap: 8px;
    padding-bottom: 2px;
  }

  .category-button {
    width: auto;
    min-width: 112px;
    white-space: nowrap;
  }

  .apps-panel {
    padding: 14px;
  }

  .apps-header {
    flex-direction: column;
  }

  .apps-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .search-input {
    width: 100%;
  }
}
</style>
