<script setup>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const isMarketActive = computed(() => route.path.startsWith("/market"));
const isMineActive = computed(() => route.path.startsWith("/mine"));
const isSettingsActive = computed(() => route.path.startsWith("/settings"));

function openSettings() {
  const from = isSettingsActive.value ? "/market" : route.fullPath;
  router.push({ name: "settings-download", query: { from } });
}
</script>

<template>
  <div class="app-shell">
    <aside class="primary-sidebar">
      <div class="brand">ossam</div>

      <nav class="primary-nav">
        <router-link class="nav-link" :class="{ active: isMarketActive }" to="/market">
          应用市场
        </router-link>
        <router-link class="nav-link" :class="{ active: isMineActive }" to="/mine">
          我的
        </router-link>
      </nav>

      <button class="settings-button" :class="{ active: isSettingsActive }" @click="openSettings">
        设置
      </button>
    </aside>

    <main class="content-area">
      <router-view />
    </main>
  </div>
</template>
