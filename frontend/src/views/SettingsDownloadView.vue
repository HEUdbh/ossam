<script setup>
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { SelectDownloadDirectory } from "../../wailsjs/go/main/App";
import {
  clearDownloadDirectory,
  getDownloadDirectory,
  setDownloadDirectory,
} from "../utils/settings";

const downloadDirectory = ref("");

const hasDirectory = computed(() => downloadDirectory.value.length > 0);
const displayDirectory = computed(() =>
  hasDirectory.value ? downloadDirectory.value : "未配置"
);

onMounted(() => {
  downloadDirectory.value = getDownloadDirectory();
});

async function chooseDirectory() {
  try {
    const selectedDirectory = await SelectDownloadDirectory(downloadDirectory.value);
    const normalizedDirectory = String(selectedDirectory || "").trim();

    // 用户取消选择时，保持当前配置不变。
    if (!normalizedDirectory) {
      return;
    }

    downloadDirectory.value = setDownloadDirectory(normalizedDirectory);
    ElMessage.success("下载目录已保存");
  } catch (error) {
    ElMessage.error(error?.message || "目录选择失败，请重试");
  }
}

function resetDirectory() {
  if (!hasDirectory.value) {
    return;
  }

  downloadDirectory.value = "";
  clearDownloadDirectory();
  ElMessage.success("已恢复为未配置");
}
</script>

<template>
  <section class="settings-block">
    <h3>下载目录设置</h3>
    <p>选择默认下载目录，保存后下次启动仍会保留。</p>

    <div class="option-item">
      <span class="label">当前下载目录</span>
      <span class="value" :title="displayDirectory">{{ displayDirectory }}</span>
    </div>

    <div class="actions">
      <el-button type="primary" @click="chooseDirectory">选择目录</el-button>
      <el-button :disabled="!hasDirectory" @click="resetDirectory">恢复未配置</el-button>
    </div>
  </section>
</template>

<style scoped>
.settings-block h3 {
  margin: 0;
  font-size: 18px;
}

.settings-block p {
  margin: 8px 0 0;
  color: var(--text-secondary);
}

.option-item {
  margin-top: 14px;
  border: 1px solid var(--line-color);
  border-radius: var(--radius-md);
  background: #ffffff;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  color: var(--text-secondary);
}

.value {
  color: var(--text-primary);
  font-weight: 600;
  word-break: break-all;
}

.actions {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
</style>
