<script setup>
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
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
      <button
        v-for="category in categories"
        :key="category"
        class="category-button"
        :class="{ active: category === selectedCategory }"
        @click="chooseCategory(category)"
      >
        {{ category }}
      </button>
    </aside>
    <section class="apps-panel">
      <header class="apps-header">
        <h1>{{ state.config?.market_name || "应用市场" }}</h1>
        <span class="updated">
          更新时间：{{ state.config?.last_updated || "-" }}
        </span>
      </header>

      <div v-if="state.loading" class="placeholder">正在加载配置...</div>
      <div v-else-if="state.error" class="placeholder error">
        配置加载失败：{{ state.error }}
      </div>
      <div v-else-if="!currentApps.length" class="placeholder">
        当前分类暂无应用。
      </div>
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
          <span class="app-name">{{ app.name }}</span>
        </router-link>
      </div>
    </section>
  </div>
</template>

<style scoped>
.market-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  height: 100%;
  min-height: 0;
}

.category-sidebar {
  border-right: 1px solid #e6e8eb;
  background: #f9fafb;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sidebar-title {
  color: #4b5563;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 4px;
}

.category-button {
  border: 1px solid #e5e7eb;
  background: #ffffff;
  color: #111827;
  border-radius: 8px;
  padding: 10px 12px;
  text-align: left;
  cursor: pointer;
}

.category-button.active {
  background: #1f2937;
  border-color: #1f2937;
  color: #ffffff;
}

.apps-panel {
  min-height: 0;
  padding: 20px;
  display: flex;
  flex-direction: column;
}

.apps-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 16px;
}

.apps-header h1 {
  margin: 0;
  font-size: 22px;
}

.updated {
  font-size: 13px;
  color: #6b7280;
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

.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 14px;
}

.app-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: #111827;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 14px 10px;
}

.app-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.08);
}

.app-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  object-fit: cover;
  background: #f3f4f6;
}

.app-name {
  font-size: 14px;
  text-align: center;
  overflow-wrap: anywhere;
}
</style>
