<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessageBox } from "element-plus";
import { Expand, Fold, Setting, Shop, User } from "@element-plus/icons-vue";
import { getDownloadDirectory } from "./utils/settings";

let startupDownloadDirectoryChecked = false;
const SIDEBAR_COLLAPSE_KEY = "ossam.ui.primarySidebarCollapsed";

const route = useRoute();
const router = useRouter();
const sidebarCollapsed = ref(readSidebarCollapsed());

const isMarketActive = computed(() => route.path.startsWith("/market"));
const isMineActive = computed(() => route.path.startsWith("/mine"));
const isSettingsActive = computed(() => route.path.startsWith("/settings"));

const pageMeta = computed(() => {
  if (route.path.startsWith("/market/app/")) {
    return {
      title: "应用详情",
      subtitle: "查看项目主页、最新发布信息与多平台下载入口",
    };
  }
  if (route.path.startsWith("/mine")) {
    return {
      title: "我的",
      subtitle: "管理个人偏好、下载记录与常用项目",
    };
  }
  if (route.path.startsWith("/settings")) {
    return {
      title: "设置",
      subtitle: "管理下载目录与作者相关信息",
    };
  }
  return {
    title: "应用市场",
    subtitle: "发现并管理高质量开源工具",
  };
});

const pageTitle = computed(() => pageMeta.value.title);
const pageSubtitle = computed(() => pageMeta.value.subtitle);

function getSettingsFromRoute() {
  return isSettingsActive.value ? "/market" : route.fullPath;
}

function goToDownloadSettings(from) {
  router.push({ name: "settings-download", query: { from } });
}

function openSettings() {
  goToDownloadSettings(getSettingsFromRoute());
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

onMounted(() => {
  checkDownloadDirectoryOnStartup();
});

watch(
  sidebarCollapsed,
  (value) => {
    persistSidebarCollapsed(value);
  },
  { immediate: true }
);
</script>

<template>
  <div class="app-shell">
    <aside class="primary-sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-top">
        <div class="brand-card">
          <div class="brand-mark">OS</div>
          <div class="brand-copy">
            <div class="brand-name">ossam</div>
            <div class="brand-desc">Open Source App Market</div>
          </div>
        </div>

        <button
          class="sidebar-toggle"
          :title="sidebarCollapsed ? '展开侧栏' : '折叠侧栏'"
          @click="toggleSidebar"
        >
          <el-icon>
            <Expand v-if="sidebarCollapsed" />
            <Fold v-else />
          </el-icon>
          <span class="nav-label">菜单</span>
        </button>

        <nav class="primary-nav">
          <router-link
            class="nav-link"
            :class="{ active: isMarketActive }"
            to="/market"
            title="应用市场"
          >
            <el-icon><Shop /></el-icon>
            <span class="nav-label">应用市场</span>
          </router-link>
          <router-link class="nav-link" :class="{ active: isMineActive }" to="/mine" title="我的">
            <el-icon><User /></el-icon>
            <span class="nav-label">我的</span>
          </router-link>
        </nav>
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
      <header class="page-header">
        <div>
          <h1>{{ pageTitle }}</h1>
          <p>{{ pageSubtitle }}</p>
        </div>
      </header>

      <section class="page-content">
        <router-view />
      </section>
    </main>
  </div>
</template>
