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

const activeTab = computed(() => {
  if (route.name === "settings-about") {
    return "about";
  }
  if (route.name === "settings-cdn") {
    return "cdn";
  }
  return "download";
});

function closeModal() {
  router.push(fromRoute.value);
}

function switchTab(tab) {
  const routeName =
    tab === "about" ? "settings-about" : tab === "cdn" ? "settings-cdn" : "settings-download";
  router.push({
    name: routeName,
    query: { from: fromRoute.value },
  });
}
</script>

<template>
  <el-dialog
    :model-value="true"
    width="780px"
    top="7vh"
    class="settings-dialog"
    modal-class="settings-overlay"
    :show-close="false"
    :close-on-click-modal="false"
    @close="closeModal"
  >
    <template #header>
      <div class="dialog-header">
        <div>
          <h2>设置</h2>
          <p>管理下载目录、CDN 加速与作者信息。</p>
        </div>
        <el-button text @click="closeModal">关闭</el-button>
      </div>
    </template>

    <el-tabs :model-value="activeTab" class="settings-tabs" @tab-change="switchTab">
      <el-tab-pane name="download" label="下载设置" />
      <el-tab-pane name="cdn" label="CDN 设置" />
      <el-tab-pane name="about" label="关于" />
    </el-tabs>

    <div class="dialog-content">
      <router-view />
    </div>
  </el-dialog>
</template>

<style scoped>
:deep(.settings-dialog .el-dialog) {
  border-radius: 0;
  border: 1px solid var(--line-color);
  background: var(--surface-dialog);
  box-shadow: none;
  backdrop-filter: blur(8px);
}

:deep(.settings-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 14px 16px 10px;
  border-bottom: 1px solid var(--line-color);
}

:deep(.settings-dialog .el-dialog__body) {
  padding: 10px 16px 16px;
}

.dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.dialog-header h2 {
  margin: 0;
  font-size: 20px;
  line-height: 1.2;
}

.dialog-header p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.settings-tabs {
  margin-bottom: 10px;
}

.dialog-content {
  border-top: 1px solid var(--line-color);
  background: var(--surface-dialog);
  padding: 14px 0 0;
}

:deep(.settings-overlay) {
  background: rgba(255, 255, 255, 0.38);
}

@media (max-width: 960px) {
  :deep(.settings-dialog) {
    width: calc(100vw - 20px) !important;
  }

  :deep(.settings-dialog .el-dialog__header),
  :deep(.settings-dialog .el-dialog__body) {
    padding-left: 12px;
    padding-right: 12px;
  }
}
</style>
