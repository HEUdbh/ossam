<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  ArrowRight,
  Connection,
  DataAnalysis,
  Grid,
  Lock,
  Monitor,
  Search,
  Setting,
  Tools,
  User,
  Promotion,
} from "@element-plus/icons-vue";
import { SetCDNSettings } from "../wailsjs/go/main/App";
import { loadAppsConfig, useAppsStore } from "./stores/appsStore";
import {
  getCDNSettings,
  getDownloadDirectory,
  setCDNSettings,
} from "./utils/settings";
import { getCategoryDisplayName } from "./utils/marketMeta";

let startupDownloadDirectoryChecked = false;
let searchSyncTimer = null;

const LAST_CATEGORY_KEY = "ossam.market.lastCategory";
const SIDEBAR_COLLAPSE_KEY = "ossam.ui.sidebarCollapsed";

const route = useRoute();
const router = useRouter();
const { categories } = useAppsStore();

const globalSearch = ref("");
const sidebarCollapsed = ref(readSidebarCollapsed());

const isMineActive = computed(() => route.path.startsWith("/mine"));
const isSettingsActive = computed(() => route.path.startsWith("/settings"));

const CATEGORY_ICON_MAP = {
  Security: Lock,
  DevTools: Tools,
  System: Monitor,
  Network: Connection,
  Database: DataAnalysis,
  Terminal: Promotion,
  Utility: Grid,
};

const routeCategory = computed(() => {
  const fromParams = typeof route.params.category === "string" ? route.params.category : "";
  if (fromParams) {
    return fromParams;
  }

  const fromQuery = typeof route.query.category === "string" ? route.query.category : "";
  return fromQuery;
});

const activeCategory = computed(() => {
  if (routeCategory.value && categories.value.includes(routeCategory.value)) {
    return routeCategory.value;
  }

  const cached = readLastCategory();
  if (cached && categories.value.includes(cached)) {
    return cached;
  }

  return categories.value[0] || "";
});

function getCategoryIcon(category) {
  return CATEGORY_ICON_MAP[category] || Grid;
}

function readSidebarCollapsed() {
  if (typeof window === "undefined") {
    return false;
  }
  return window.localStorage.getItem(SIDEBAR_COLLAPSE_KEY) === "1";
}

function persistSidebarCollapsed(value) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(SIDEBAR_COLLAPSE_KEY, value ? "1" : "0");
}

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}

function readLastCategory() {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(LAST_CATEGORY_KEY) || "";
}

function persistLastCategory(category) {
  if (typeof window === "undefined" || !category) {
    return;
  }
  window.localStorage.setItem(LAST_CATEGORY_KEY, category);
}

function syncSearchQuery(value) {
  const nextSearch = String(value || "").trim();
  const currentSearch = typeof route.query.q === "string" ? route.query.q : "";
  if (nextSearch === currentSearch) {
    return;
  }

  const query = { ...route.query };
  if (nextSearch) {
    query.q = nextSearch;
  } else {
    delete query.q;
  }

  router.replace({ path: route.path, query });
}

function resolveFallbackCategory() {
  const fromParams = typeof route.params.category === "string" ? route.params.category.trim() : "";
  if (fromParams) {
    return fromParams;
  }

  const fromQuery = typeof route.query.category === "string" ? route.query.category.trim() : "";
  if (fromQuery) {
    return fromQuery;
  }

  const cached = readLastCategory();
  if (cached) {
    return cached;
  }

  return categories.value[0] || "";
}

function handleBack() {
  const hasRouterBack = Boolean(window.history.state?.back);
  if (hasRouterBack || window.history.length > 1) {
    router.back();
    return;
  }

  const fallbackCategory = resolveFallbackCategory();
  const query = fallbackCategory ? { category: fallbackCategory } : {};
  router.push({ name: "market", query });
}

function selectCategory(category) {
  if (!category) {
    return;
  }

  persistLastCategory(category);

  const query = { category };
  if (typeof route.query.q === "string" && route.query.q.trim()) {
    query.q = route.query.q.trim();
  }

  router.push({
    name: "market",
    query,
  });
}

function openMine() {
  router.push({ name: "mine" });
}

function getSettingsFromRoute() {
  return isSettingsActive.value ? "/market" : route.fullPath;
}

function goToDownloadSettings(from) {
  router.push({ name: "settings-download", query: { from } });
}

function openSettings() {
  goToDownloadSettings(getSettingsFromRoute());
}

function notifyAuthPending(action) {
  ElMessage.info(`${action} 功能建设中`);
}

async function checkDownloadDirectoryOnStartup() {
  if (startupDownloadDirectoryChecked) {
    return;
  }
  startupDownloadDirectoryChecked = true;

  if (getDownloadDirectory()) {
    return;
  }

  try {
    await ElMessageBox.confirm(
      "尚未设置下载目录，建议先完成设置后再开始下载。",
      "下载目录未配置",
      {
        type: "warning",
        confirmButtonText: "前往设置",
        cancelButtonText: "稍后再说",
      }
    );
    goToDownloadSettings(getSettingsFromRoute());
  } catch {
    // User chose to postpone.
  }
}

async function syncCDNSettingsOnStartup() {
  try {
    const localSettings = getCDNSettings();
    const synced = await SetCDNSettings(localSettings);
    setCDNSettings({
      enabled: synced.enabled,
      selected_source: synced.selected_source,
      custom_sources: synced.custom_sources,
    });
  } catch {
    // Keep startup resilient even if CDN sync fails.
  }
}

onMounted(() => {
  void loadAppsConfig();
  syncCDNSettingsOnStartup();
  checkDownloadDirectoryOnStartup();
});

onBeforeUnmount(() => {
  if (searchSyncTimer) {
    clearTimeout(searchSyncTimer);
  }
});

watch(
  () => route.query.q,
  (value) => {
    const nextValue = typeof value === "string" ? value : "";
    if (nextValue !== globalSearch.value) {
      globalSearch.value = nextValue;
    }
  },
  { immediate: true }
);

watch(globalSearch, (value) => {
  if (searchSyncTimer) {
    clearTimeout(searchSyncTimer);
  }

  searchSyncTimer = setTimeout(() => {
    syncSearchQuery(value);
  }, 240);
});

watch(
  activeCategory,
  (category) => {
    if (category) {
      persistLastCategory(category);
    }
  },
  { immediate: true }
);

watch(
  sidebarCollapsed,
  (value) => {
    persistSidebarCollapsed(value);
  },
  { immediate: true }
);
</script>

<template>
  <div class="app-root">
    <header class="global-topbar">
      <div class="topbar-left">
        <button class="back-button" title="返回" @click="handleBack">
          <el-icon><ArrowLeft /></el-icon>
        </button>

        <div class="brand-wrap">
          <img class="brand-logo" src="./assets/images/logo-universal.png" alt="ossam" />
          <span class="brand-name">ossam</span>
        </div>
      </div>

      <div class="topbar-center">
        <el-input
          v-model="globalSearch"
          clearable
          class="global-search"
          placeholder="搜索应用名称、简介或仓库"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="topbar-right">
        <el-button text @click="notifyAuthPending('登录')">登录</el-button>
        <el-button text class="register-btn" @click="notifyAuthPending('注册')">注册</el-button>
      </div>
    </header>

    <div class="app-shell">
      <aside class="app-sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-top">
          <div class="sidebar-head">
            <span v-if="!sidebarCollapsed" class="sidebar-label">分类</span>
            <button
              class="sidebar-toggle"
              :title="sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'"
              @click="toggleSidebar"
            >
              <el-icon>
                <ArrowRight v-if="sidebarCollapsed" />
                <ArrowLeft v-else />
              </el-icon>
            </button>
          </div>

          <nav class="category-nav">
            <button
              v-for="category in categories"
              :key="category"
              class="category-nav-item"
              :class="{ active: category === activeCategory }"
              :title="getCategoryDisplayName(category)"
              @click="selectCategory(category)"
            >
              <el-icon>
                <component :is="getCategoryIcon(category)" />
              </el-icon>
              <span class="nav-label">{{ getCategoryDisplayName(category) }}</span>
            </button>
          </nav>

          <div class="sidebar-divider" />

          <button
            class="sidebar-link"
            :class="{ active: isMineActive }"
            title="我的"
            @click="openMine"
          >
            <el-icon><User /></el-icon>
            <span class="nav-label">我的</span>
          </button>
        </div>

        <el-button
          class="settings-button"
          :class="{ active: isSettingsActive }"
          title="设置"
          @click="openSettings"
        >
          <el-icon><Setting /></el-icon>
          <span class="nav-label">设置</span>
        </el-button>
      </aside>

      <main class="content-area">
        <router-view />
      </main>
    </div>
  </div>
</template>
