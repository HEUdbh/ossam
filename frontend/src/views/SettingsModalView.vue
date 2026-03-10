<script setup>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const fromRoute = computed(() => {
  if (
    typeof route.query.from === "string" &&
    route.query.from.startsWith("/") &&
    !route.query.from.startsWith("/settings")
  ) {
    return route.query.from;
  }
  return "/market";
});

const downloadRoute = computed(() => ({
  name: "settings-download",
  query: { from: fromRoute.value },
}));

const aboutRoute = computed(() => ({
  name: "settings-about",
  query: { from: fromRoute.value },
}));

function closeModal() {
  router.push(fromRoute.value);
}
</script>

<template>
  <div class="settings-overlay">
    <div class="settings-window">
      <header class="window-header">
        <h2>设置</h2>
        <button class="close" @click="closeModal">关闭</button>
      </header>

      <nav class="sub-nav">
        <router-link class="sub-link" :to="downloadRoute">
          下载地址设置
        </router-link>
        <router-link class="sub-link" :to="aboutRoute">
          关于作者
        </router-link>
      </nav>

      <div class="window-content">
        <router-view />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-overlay {
  height: 100%;
  background: rgba(15, 23, 42, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.settings-window {
  width: min(720px, 100%);
  background: #ffffff;
  border-radius: 14px;
  border: 1px solid #e5e7eb;
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.2);
}

.window-header {
  padding: 14px 16px;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.window-header h2 {
  margin: 0;
  font-size: 18px;
}

.close {
  border: 1px solid #d1d5db;
  background: #f9fafb;
  border-radius: 8px;
  padding: 6px 10px;
  cursor: pointer;
}

.sub-nav {
  display: flex;
  gap: 10px;
  padding: 12px 16px 0;
}

.sub-link {
  text-decoration: none;
  color: #111827;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 6px 10px;
}

.sub-link.router-link-active {
  background: #1f2937;
  color: #ffffff;
  border-color: #1f2937;
}

.window-content {
  padding: 16px;
  min-height: 180px;
}
</style>
