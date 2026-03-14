<script setup>
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { GetCDNSettings, SetCDNSettings } from "../../wailsjs/go/main/App";
import { BUILTIN_CDN_SOURCES, getCDNSettings, setCDNSettings } from "../utils/settings";
import { invalidateAppDetailCache } from "../stores/appsStore";

const loading = ref(false);
const saving = ref(false);
const enabled = ref(true);
const selectedSource = ref(BUILTIN_CDN_SOURCES[0]);
const builtinSources = ref([...BUILTIN_CDN_SOURCES]);
const customSources = ref([]);
const newCustomSource = ref("");

const allSources = computed(() => [
  ...builtinSources.value.map((url) => ({ url, builtin: true })),
  ...customSources.value.map((url) => ({ url, builtin: false })),
]);

const hasCustomSources = computed(() => customSources.value.length > 0);

function applySettings(settings) {
  const nextBuiltin = Array.isArray(settings?.builtin_sources)
    ? settings.builtin_sources
    : BUILTIN_CDN_SOURCES;
  const nextCustom = Array.isArray(settings?.custom_sources) ? settings.custom_sources : [];
  const fallbackSource = nextBuiltin[0] || BUILTIN_CDN_SOURCES[0];

  builtinSources.value = [...nextBuiltin];
  customSources.value = [...nextCustom];
  enabled.value = Boolean(settings?.enabled);
  selectedSource.value = String(settings?.selected_source || "").trim() || fallbackSource;
}

function buildPayload() {
  return {
    enabled: enabled.value,
    selected_source: selectedSource.value,
    custom_sources: customSources.value,
  };
}

async function persistSettings() {
  saving.value = true;
  try {
    const response = await SetCDNSettings(buildPayload());
    applySettings(response);
    setCDNSettings({
      enabled: response.enabled,
      selected_source: response.selected_source,
      custom_sources: response.custom_sources,
    });
    invalidateAppDetailCache();
    return true;
  } catch (error) {
    ElMessage.error(error?.message || "保存 CDN 设置失败");
    return false;
  } finally {
    saving.value = false;
  }
}

async function loadSettings() {
  loading.value = true;
  try {
    const localSettings = getCDNSettings();
    const synced = await SetCDNSettings(localSettings);
    applySettings(synced);
    setCDNSettings({
      enabled: synced.enabled,
      selected_source: synced.selected_source,
      custom_sources: synced.custom_sources,
    });
  } catch {
    try {
      const remote = await GetCDNSettings();
      applySettings(remote);
      setCDNSettings({
        enabled: remote.enabled,
        selected_source: remote.selected_source,
        custom_sources: remote.custom_sources,
      });
    } catch (error) {
      ElMessage.error(error?.message || "加载 CDN 设置失败");
    }
  } finally {
    loading.value = false;
  }
}

async function toggleEnabled() {
  await persistSettings();
}

async function changeSelectedSource() {
  await persistSettings();
}

async function addCustomSource() {
  const candidate = String(newCustomSource.value || "").trim();
  if (!candidate) {
    return;
  }

  customSources.value = [...customSources.value, candidate];
  const ok = await persistSettings();
  if (ok) {
    newCustomSource.value = "";
    ElMessage.success("已新增自定义加速源");
  }
}

async function saveCustomSource(index) {
  const ok = await persistSettings();
  if (ok) {
    ElMessage.success(`已保存第 ${index + 1} 条自定义加速源`);
  }
}

async function removeCustomSource(index) {
  const next = customSources.value.filter((_, idx) => idx !== index);
  customSources.value = next;
  const ok = await persistSettings();
  if (ok) {
    ElMessage.success("已删除自定义加速源");
  }
}

onMounted(() => {
  loadSettings();
});
</script>

<template>
  <section class="settings-block">
    <h3>CDN 设置</h3>
    <p>控制 GitHub 请求加速（release 列表接口保持当前逻辑不变）。</p>

    <el-skeleton v-if="loading" :rows="5" animated />

    <template v-else>
      <div class="option-item switch-item">
        <span class="label">启用加速</span>
        <el-switch
          v-model="enabled"
          :disabled="saving"
          active-text="启用"
          inactive-text="禁用"
          @change="toggleEnabled"
        />
      </div>

      <div class="option-item">
        <span class="label">当前加速源</span>
        <el-select
          v-model="selectedSource"
          class="source-select"
          :disabled="saving"
          @change="changeSelectedSource"
        >
          <el-option
            v-for="item in allSources"
            :key="item.url"
            :label="`${item.url}${item.builtin ? '（内置）' : ''}`"
            :value="item.url"
          />
        </el-select>
      </div>

      <div class="option-item">
        <span class="label">内置加速源（锁定）</span>
        <div class="source-list">
          <div v-for="source in builtinSources" :key="source" class="source-line">
            <span class="value">{{ source }}</span>
            <el-tag size="small" effect="plain" type="info">锁定</el-tag>
          </div>
        </div>
      </div>

      <div class="option-item">
        <span class="label">自定义加速源</span>

        <div class="add-row">
          <el-input
            v-model="newCustomSource"
            placeholder="请输入 https:// 开头的加速源"
            :disabled="saving"
          />
          <el-button type="primary" :loading="saving" @click="addCustomSource">新增</el-button>
        </div>

        <div v-if="hasCustomSources" class="source-list custom-list">
          <div v-for="(source, index) in customSources" :key="`${index}-${source}`" class="custom-row">
            <el-input v-model="customSources[index]" :disabled="saving" />
            <el-button :loading="saving" @click="saveCustomSource(index)">保存</el-button>
            <el-button type="danger" plain :loading="saving" @click="removeCustomSource(index)">
              删除
            </el-button>
          </div>
        </div>
        <el-empty v-else description="暂无自定义加速源" :image-size="72" />
      </div>
    </template>
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
  background: var(--surface-settings-clear);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.switch-item {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}

.label {
  color: var(--text-secondary);
  font-weight: 600;
}

.value {
  color: var(--text-primary);
  font-weight: 600;
  word-break: break-all;
}

.source-select {
  width: 100%;
}

.source-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.source-line {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-sm);
  background: var(--surface-settings-clear);
  padding: 8px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.add-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.add-row :deep(.el-input) {
  flex: 1;
  min-width: 220px;
}

.custom-list {
  margin-top: 4px;
}

.custom-row {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto auto;
  gap: 8px;
}

@media (max-width: 768px) {
  .custom-row {
    grid-template-columns: 1fr;
  }
}
</style>
