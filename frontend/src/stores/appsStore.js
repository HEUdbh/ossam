import { computed, reactive } from "vue";
import { GetAppsConfig } from "../../wailsjs/go/main/App";

const state = reactive({
  config: null,
  loading: false,
  loaded: false,
  error: "",
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
  };
}
