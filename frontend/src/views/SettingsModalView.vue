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

const activeTab = computed(() => (route.name === "settings-about" ? "about" : "download"));

function closeModal() {
  router.push(fromRoute.value);
}

function switchTab(tab) {
  const routeName = tab === "about" ? "settings-about" : "settings-download";
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
    :show-close="false"
    :close-on-click-modal="false"
    @close="closeModal"
  >
    <template #header>
      <div class="dialog-header">
        <div>
          <h2>设置</h2>
          <p>管理下载行为与作者信息</p>
        </div>
        <el-button text @click="closeModal">关闭</el-button>
      </div>
    </template>

    <el-tabs :model-value="activeTab" class="settings-tabs" @tab-change="switchTab">
      <el-tab-pane name="download" label="下载地址设置" />
      <el-tab-pane name="about" label="关于作者" />
    </el-tabs>

    <div class="dialog-content">
      <router-view />
    </div>
  </el-dialog>
</template>

<style scoped>
:deep(.settings-dialog) {
  border-radius: var(--radius-xl);
}

:deep(.settings-dialog .el-dialog) {
  border: 1px solid var(--line-color);
  background: linear-gradient(180deg, #ffffff, #f8fdf9);
  box-shadow: var(--shadow-soft);
}

:deep(.settings-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 18px 20px 12px;
  border-bottom: 1px solid var(--line-color);
}

:deep(.settings-dialog .el-dialog__body) {
  padding: 12px 20px 20px;
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
  line-height: 1.3;
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
  border: 1px solid var(--line-color);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
  padding: 16px;
}

@media (max-width: 960px) {
  :deep(.settings-dialog) {
    width: calc(100vw - 20px) !important;
  }

  :deep(.settings-dialog .el-dialog__header),
  :deep(.settings-dialog .el-dialog__body) {
    padding-left: 14px;
    padding-right: 14px;
  }
}
</style>
