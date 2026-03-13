<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import StatusState from "../components/StatusState.vue";
import { loadAppsConfig, loadRepoStars, useAppsStore } from "../stores/appsStore";
import { getCategoryDisplayName } from "../utils/marketMeta";

const HERO_LIMIT = 4;

const route = useRoute();
const router = useRouter();
const { state, categories, appsByCategory, getRepoStarCount } = useAppsStore();

const selectedCategory = ref("");

const normalizedSearch = computed(() => String(route.query.q || "").trim().toLowerCase());
const isSearchMode = computed(() => normalizedSearch.value.length > 0);

const categoryItems = computed(() => {
  if (!selectedCategory.value) {
    return [];
  }

  return appsByCategory(selectedCategory.value).map((app, index) => ({
    app,
    category: selectedCategory.value,
    index,
    star: getStarValue(app.repo),
  }));
});

const sortedCategoryItems = computed(() =>
  [...categoryItems.value].sort((left, right) => {
    const leftStar = typeof left.star === "number" ? left.star : -1;
    const rightStar = typeof right.star === "number" ? right.star : -1;
    if (leftStar !== rightStar) {
      return rightStar - leftStar;
    }
    return left.index - right.index;
  })
);

const heroItems = computed(() => {
  if (isSearchMode.value) {
    return [];
  }
  return sortedCategoryItems.value.slice(0, HERO_LIMIT);
});

const compactItems = computed(() => {
  if (isSearchMode.value) {
    return searchItems.value;
  }

  const remaining = sortedCategoryItems.value.slice(HERO_LIMIT);
  if (remaining.length) {
    return remaining;
  }

  return sortedCategoryItems.value;
});

const allItems = computed(() => {
  const result = [];
  categories.value.forEach((category) => {
    appsByCategory(category).forEach((app, index) => {
      result.push({
        app,
        category,
        index,
        star: getStarValue(app.repo),
      });
    });
  });
  return result;
});

const searchItems = computed(() => {
  const keyword = normalizedSearch.value;
  if (!keyword) {
    return [];
  }

  return allItems.value
    .filter(({ app, category }) => {
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
    })
    .sort((left, right) => {
      const leftStar = typeof left.star === "number" ? left.star : -1;
      const rightStar = typeof right.star === "number" ? right.star : -1;
      if (leftStar !== rightStar) {
        return rightStar - leftStar;
      }
      return String(left.app.name || "").localeCompare(String(right.app.name || ""));
    });
});

const emptyTitle = computed(() =>
  isSearchMode.value ? "没有找到匹配的应用" : "当前分类暂无应用"
);

const emptyDescription = computed(() =>
  isSearchMode.value
    ? "请尝试更换关键词（支持名称、简介、仓库搜索）。"
    : "可以在 config/rules.json 与 config/categories/*.json 中补充应用。"
);

const marketTitle = computed(() =>
  isSearchMode.value ? `搜索结果 · ${searchItems.value.length}` : "应用市场"
);

const marketSubtitle = computed(() =>
  isSearchMode.value
    ? `关键词：${String(route.query.q || "").trim()}`
    : `${getCategoryDisplayName(selectedCategory.value)} · 发现优质开源工具`
);

function getStarValue(repo) {
  const value = getRepoStarCount(repo);
  return typeof value === "number" && !Number.isNaN(value) ? value : null;
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
    query: {
      ...route.query,
      category: selectedCategory.value,
    },
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

function starLabel(repo) {
  return formatStarCount(getRepoStarCount(repo));
}

function heroStyle(item) {
  const photo = String(item?.app?.photo || "").trim();
  if (!photo) {
    return {};
  }

  return {
    backgroundImage: `linear-gradient(102deg, rgba(15,23,42,.85), rgba(15,23,42,.35)), url("${photo}")`,
  };
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
  <div class="market-view">
    <header class="market-header">
      <div>
        <h1>{{ marketTitle }}</h1>
        <p>{{ marketSubtitle }}</p>
      </div>
      <el-tag effect="plain" type="success">更新：{{ state.config?.last_updated || "-" }}</el-tag>
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
      title="正在加载应用列表"
      description="正在读取本地配置文件，请稍候。"
    />
    <StatusState
      v-else-if="state.error"
      type="error"
      title="配置加载失败"
      :description="state.error"
    />
    <StatusState
      v-else-if="!compactItems.length"
      type="empty"
      :title="emptyTitle"
      :description="emptyDescription"
    />

    <template v-else>
      <section v-if="!isSearchMode && heroItems.length" class="hero-section">
        <el-carousel height="220px" trigger="click" indicator-position="outside" :interval="4200" arrow="always">
          <el-carousel-item v-for="item in heroItems" :key="`hero-${item.category}-${item.app.name}`">
            <router-link
              class="hero-card"
              :to="{
                name: 'app-detail',
                params: { category: item.category, name: item.app.name },
              }"
              :style="heroStyle(item)"
            >
              <div class="hero-content">
                <el-tag type="success" effect="light" class="hero-tag">
                  {{ getCategoryDisplayName(item.category) }}
                </el-tag>
                <h2>{{ item.app.name }}</h2>
                <p>{{ item.app.summary || '查看项目详情、发布版本与多平台下载。' }}</p>
                <span class="hero-meta">★ {{ starLabel(item.app.repo) }}</span>
              </div>
            </router-link>
          </el-carousel-item>
        </el-carousel>
      </section>

      <section class="apps-section">
        <div class="apps-section-title">
          {{ isSearchMode ? "匹配应用" : "更多应用" }}
        </div>

        <div class="compact-grid" :class="{ searching: isSearchMode }">
          <router-link
            v-for="item in compactItems"
            :key="`${item.category}-${item.app.name}`"
            class="compact-card"
            :to="{
              name: 'app-detail',
              params: { category: item.category, name: item.app.name },
            }"
          >
            <img class="compact-icon" :src="item.app.photo" :alt="item.app.name" />
            <div class="compact-body">
              <div class="compact-header">
                <h3>{{ item.app.name }}</h3>
                <el-tag v-if="isSearchMode" size="small" effect="plain" type="success">
                  {{ getCategoryDisplayName(item.category) }}
                </el-tag>
              </div>
              <p>{{ item.app.summary }}</p>
              <span class="compact-star">★ {{ starLabel(item.app.repo) }}</span>
            </div>
          </router-link>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.market-view {
  height: 100%;
  min-height: 0;
  padding: 18px 18px 14px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
  background: linear-gradient(180deg, #ffffff 0%, #fbfefc 100%);
}

.market-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.market-header h1 {
  margin: 0;
  font-size: 34px;
  line-height: 1.1;
  letter-spacing: 0.2px;
}

.market-header p {
  margin: 8px 0 0;
  color: var(--text-secondary);
}

.stars-alert {
  margin-bottom: 2px;
}

.hero-section :deep(.el-carousel__container) {
  border-radius: var(--radius-lg);
}

.hero-section :deep(.el-carousel__button) {
  background: rgba(15, 23, 42, 0.4);
}

.hero-card {
  height: 100%;
  border-radius: var(--radius-lg);
  text-decoration: none;
  color: #e2e8f0;
  background:
    linear-gradient(102deg, rgba(15, 23, 42, 0.85), rgba(15, 23, 42, 0.35)),
    linear-gradient(120deg, #1f2937, #334155);
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: flex-end;
  padding: 22px;
}

.hero-content {
  max-width: min(640px, 100%);
}

.hero-tag {
  margin-bottom: 10px;
}

.hero-content h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.1;
  color: #f8fafc;
}

.hero-content p {
  margin: 10px 0 0;
  font-size: 14px;
  line-height: 1.55;
  color: rgba(226, 232, 240, 0.94);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.hero-meta {
  margin-top: 14px;
  display: inline-block;
  color: #facc15;
  font-weight: 700;
  font-size: 14px;
}

.apps-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.apps-section-title {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.compact-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.compact-grid.searching {
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

.compact-card {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-lg);
  background: #fff;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.05);
  text-decoration: none;
  color: var(--text-primary);
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  transition: all 0.2s ease;
}

.compact-card:hover {
  transform: translateY(-2px);
  border-color: rgba(18, 199, 123, 0.36);
  box-shadow: 0 12px 24px rgba(18, 199, 123, 0.16);
}

.compact-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  border: 1px solid #e8f1ed;
  object-fit: cover;
  background: var(--surface-3);
}

.compact-body {
  min-width: 0;
}

.compact-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.compact-header h3 {
  margin: 0;
  min-width: 0;
  font-size: 16px;
  line-height: 1.25;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.compact-body p {
  margin: 7px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.compact-star {
  margin-top: 10px;
  display: inline-block;
  color: var(--warning-color);
  font-size: 13px;
  font-weight: 700;
}

@media (max-width: 960px) {
  .market-view {
    padding: 14px;
    gap: 12px;
  }

  .market-header h1 {
    font-size: 28px;
  }

  .hero-section :deep(.el-carousel__container) {
    height: 200px !important;
  }

  .hero-content h2 {
    font-size: 24px;
  }
}
</style>
