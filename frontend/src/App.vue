<script setup>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Setting, Shop, User } from "@element-plus/icons-vue";

const route = useRoute();
const router = useRouter();

const isMarketActive = computed(() => route.path.startsWith("/market"));
const isMineActive = computed(() => route.path.startsWith("/mine"));
const isSettingsActive = computed(() => route.path.startsWith("/settings"));

const pageTitle = computed(() => {
  if (route.path.startsWith("/market/app/")) {
    return "应用详情";
  }
  if (route.path.startsWith("/mine")) {
    return "我的";
  }
  if (route.path.startsWith("/settings")) {
    return "设置";
  }
  return "应用市场";
});

const pageSubtitle = computed(() => {
  if (route.path.startsWith("/market/app/")) {
    return "查看开源应用基础信息与下载入口";
  }
  if (route.path.startsWith("/mine")) {
    return "管理个人使用偏好（阶段占位）";
  }
  if (route.path.startsWith("/settings")) {
    return "下载设置与作者信息";
  }
  return "探索精选开源工具";
});

function openSettings() {
  const from = isSettingsActive.value ? "/market" : route.fullPath;
  router.push({ name: "settings-download", query: { from } });
}
</script>

<template>
  <div class="app-shell">
    <aside class="primary-sidebar">
      <div class="brand-card">
        <div class="brand-mark">OS</div>
        <div>
          <div class="brand-name">ossam</div>
          <div class="brand-desc">Open Source App Market</div>
        </div>
      </div>

      <nav class="primary-nav">
        <router-link class="nav-link" :class="{ active: isMarketActive }" to="/market">
          <el-icon><Shop /></el-icon>
          <span>应用市场</span>
        </router-link>
        <router-link class="nav-link" :class="{ active: isMineActive }" to="/mine">
          <el-icon><User /></el-icon>
          <span>我的</span>
        </router-link>
      </nav>

      <el-button class="settings-button" :class="{ active: isSettingsActive }" @click="openSettings">
        <el-icon><Setting /></el-icon>
        <span>设置</span>
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
