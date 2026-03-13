import { computed, reactive } from "vue";
import { sortCategories } from "../utils/marketMeta";
import {
  GetAppReleaseDetail,
  GetAppsConfig,
  GetRepoStars,
} from "../../wailsjs/go/main/App";

const state = reactive({
  config: null,
  loading: false,
  loaded: false,
  error: "",
  detailCache: {},
  repoStars: {},
  repoStarsLoading: false,
  repoStarsLoaded: false,
  repoStarsError: "",
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

function collectAllRepos() {
  const apps = state.config?.apps;
  if (!apps || typeof apps !== "object") {
    return [];
  }

  const unique = new Set();
  Object.values(apps).forEach((appList) => {
    if (!Array.isArray(appList)) {
      return;
    }
    appList.forEach((app) => {
      const repo = String(app?.repo || "").trim();
      if (repo) {
        unique.add(repo);
      }
    });
  });

  return Array.from(unique);
}

export async function loadRepoStars(force = false) {
  if (!state.loaded || !state.config) {
    return state.repoStars;
  }

  if (state.repoStarsLoading) {
    return state.repoStars;
  }

  if (state.repoStarsLoaded && !force) {
    return state.repoStars;
  }

  const repos = collectAllRepos();
  if (!repos.length) {
    state.repoStars = {};
    state.repoStarsLoaded = true;
    state.repoStarsError = "";
    return state.repoStars;
  }

  state.repoStarsLoading = true;
  state.repoStarsError = "";

  try {
    const repoStars = await GetRepoStars(repos);
    state.repoStars = repoStars || {};
    state.repoStarsLoaded = true;
    return state.repoStars;
  } catch (error) {
    const message = error?.message || String(error);
    state.repoStarsError = message || "Failed to load repo stars.";
    state.repoStarsLoaded = false;
    return state.repoStars;
  } finally {
    state.repoStarsLoading = false;
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

    return sortCategories(Object.keys(apps));
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

  const getRepoStarCount = (repo) => {
    const normalizedRepo = String(repo || "").trim();
    if (!normalizedRepo) {
      return null;
    }

    const value = state.repoStars?.[normalizedRepo];
    if (typeof value !== "number" || Number.isNaN(value)) {
      return null;
    }
    return value;
  };

  return {
    state,
    categories,
    appsByCategory,
    findApp,
    getAppDetailState,
    getRepoStarCount,
  };
}
