<script setup>
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import StatusState from "../components/StatusState.vue";
import { loadAppsConfig, useAppsStore } from "../stores/appsStore";

const route = useRoute();
const router = useRouter();
const { state, categories, appsByCategory } = useAppsStore();
const selectedCategory = ref("");

const currentApps = computed(() => appsByCategory(selectedCategory.value));

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

loadAppsConfig().finally(() => {
  syncSelectedCategory();
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
      <div class="sidebar-title">软件分类</div>
      <el-scrollbar class="category-scroll">
        <button
          v-for="category in categories"
          :key="category"
          class="category-button"
          :class="{ active: category === selectedCategory }"
          @click="chooseCategory(category)"
        >
          {{ category }}
        </button>
      </el-scrollbar>
    </aside>

    <section class="apps-panel">
      <header class="apps-header">
        <div>
          <h2>{{ state.config?.market_name || "应用市场" }}</h2>
          <p>精选开源工具，一键进入详情页面。</p>
        </div>
        <el-tag round effect="plain" type="primary">
          更新：{{ state.config?.last_updated || "-" }}
        </el-tag>
      </header>

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
        v-else-if="!currentApps.length"
        type="empty"
        title="当前分类暂无应用"
        description="可在 appsconfig.json 中补充该分类的应用项。"
      />

      <div v-else class="app-grid">
        <router-link
          v-for="app in currentApps"
          :key="`${selectedCategory}-${app.name}`"
          class="app-card"
          :to="{
            name: 'app-detail',
            params: { category: selectedCategory, name: app.name },
          }"
        >
          <img class="app-icon" :src="app.photo" :alt="app.name" />
          <div class="app-body">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-repo">{{ app.repo }}</div>
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
}

.category-sidebar {
  border-right: 1px solid var(--line-color);
  background: var(--surface-2);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sidebar-title {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.2px;
}

.category-scroll {
  min-height: 0;
}

.category-button {
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  padding: 9px 10px;
  text-align: left;
  cursor: pointer;
  transition:
    color var(--duration-fast) ease,
    background-color var(--duration-fast) ease,
    border-color var(--duration-fast) ease;
}

.category-button:hover {
  color: var(--text-primary);
  background: var(--surface-1);
  border-color: var(--line-color);
}

.category-button.active {
  color: var(--brand-color);
  border-color: rgba(59, 130, 246, 0.42);
  background: var(--brand-color-weak);
  font-weight: 600;
}

.apps-panel {
  min-height: 0;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.apps-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.apps-header h2 {
  margin: 0;
  font-size: 22px;
  line-height: 1.25;
}

.apps-header p {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.app-grid {
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 14px;
  align-content: start;
}

.app-card {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr);
  gap: 12px;
  text-decoration: none;
  color: var(--text-primary);
  border: 1px solid var(--line-color);
  border-radius: var(--radius-md);
  background: var(--surface-1);
  padding: 14px;
  transition:
    border-color var(--duration-fast) ease,
    box-shadow var(--duration-fast) ease,
    transform var(--duration-fast) ease;
}

.app-card:hover {
  border-color: rgba(59, 130, 246, 0.35);
  box-shadow: var(--shadow-card);
  transform: translateY(-2px);
}

.app-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  object-fit: cover;
  background: var(--surface-3);
}

.app-body {
  min-width: 0;
}

.app-name {
  font-weight: 600;
  line-height: 1.3;
}

.app-repo {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 1279px) {
  .market-layout {
    grid-template-columns: 190px minmax(0, 1fr);
  }

  .apps-panel {
    padding: 16px;
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
  }

  .category-button {
    width: auto;
    white-space: nowrap;
  }
}
</style>
