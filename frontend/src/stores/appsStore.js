import { computed, reactive } from "vue";
import { GetAppReleaseDetail, GetAppsConfig } from "../../wailsjs/go/main/App";

const state = reactive({
  config: null,
  loading: false,
  loaded: false,
  error: "",
  detailCache: {},
});

export async function loadAppsConfig(force = false) {
  if (state.loading) {
    return;
  }
  if (state.loaded && !force) {
    return;
  }

  state.loading = true;
  state.error = "";

  try {
    state.config = await GetAppsConfig();
    state.loaded = true;
  } catch (error) {
    const message = error?.message || String(error);
    state.error = message || "加载应用配置失败";
    state.config = null;
    state.loaded = false;
  } finally {
    state.loading = false;
  }
}

function buildDetailKey(appInfo) {
  const repo = String(appInfo?.repo || "").trim();
  const match = String(appInfo?.match || "").trim();
  if (!repo || !match) {
    return "";
  }
  return `${repo}::${match}`;
}

function ensureDetailEntry(detailKey) {
  if (!state.detailCache[detailKey]) {
    state.detailCache[detailKey] = reactive({
      data: null,
      loading: false,
      loaded: false,
      error: "",
    });
  }
  return state.detailCache[detailKey];
}

export function getAppDetailState(appInfo) {
  const detailKey = buildDetailKey(appInfo);
  if (!detailKey) {
    return {
      data: null,
      loading: false,
      loaded: false,
      error: "",
    };
  }
  return ensureDetailEntry(detailKey);
}

export async function loadAppDetail(appInfo, force = false) {
  const detailKey = buildDetailKey(appInfo);
  if (!detailKey) {
    return null;
  }

  const entry = ensureDetailEntry(detailKey);
  if (entry.loading) {
    return entry.data;
  }
  if (entry.loaded && !force) {
    return entry.data;
  }

  entry.loading = true;
  entry.error = "";

  try {
    entry.data = await GetAppReleaseDetail(appInfo.repo, appInfo.match);
    entry.loaded = true;
    return entry.data;
  } catch (error) {
    const message = error?.message || String(error);
    entry.error = message || "Failed to load app detail.";
    entry.loaded = false;
    entry.data = null;
    return null;
  } finally {
    entry.loading = false;
  }
}

export function useAppsStore() {
  const categories = computed(() => {
    const apps = state.config?.apps;
    if (!apps || typeof apps !== "object") {
      return [];
    }

    return Object.keys(apps).sort((left, right) => left.localeCompare(right));
  });

  const appsByCategory = (category) => {
    if (!category || !state.config?.apps?.[category]) {
      return [];
    }
    return state.config.apps[category];
  };

  const findApp = (category, name) => {
    const apps = appsByCategory(category);
    return apps.find((app) => app.name === name);
  };

  return {
    state,
    categories,
    appsByCategory,
    findApp,
    getAppDetailState,
  };
}
