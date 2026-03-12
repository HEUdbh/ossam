export const DOWNLOAD_DIR_KEY = "ossam.settings.downloadDirectory";
export const CDN_SETTINGS_KEY = "ossam.settings.cdn";
export const BUILTIN_CDN_SOURCES = ["https://ghproxy.net/", "https://ghfast.top/"];

export function getDownloadDirectory() {
  const stored = window.localStorage.getItem(DOWNLOAD_DIR_KEY);
  if (typeof stored !== "string") {
    return "";
  }

  return stored.trim();
}

export function setDownloadDirectory(path) {
  const normalized = String(path || "").trim();
  if (!normalized) {
    window.localStorage.removeItem(DOWNLOAD_DIR_KEY);
    return "";
  }

  window.localStorage.setItem(DOWNLOAD_DIR_KEY, normalized);
  return normalized;
}

export function clearDownloadDirectory() {
  window.localStorage.removeItem(DOWNLOAD_DIR_KEY);
}

function normalizeSourceUrl(url) {
  const normalized = String(url || "").trim();
  if (!normalized) {
    return "";
  }

  if (!/^https:\/\//i.test(normalized)) {
    return "";
  }

  return normalized.endsWith("/") ? normalized : `${normalized}/`;
}

function dedupeSources(sources) {
  const result = [];
  const seen = new Set();

  sources.forEach((source) => {
    const normalized = normalizeSourceUrl(source);
    if (!normalized || seen.has(normalized)) {
      return;
    }
    seen.add(normalized);
    result.push(normalized);
  });

  return result;
}

export function getCDNSettings() {
  const defaults = {
    enabled: true,
    selected_source: BUILTIN_CDN_SOURCES[0],
    custom_sources: [],
  };

  const raw = window.localStorage.getItem(CDN_SETTINGS_KEY);
  if (!raw) {
    return defaults;
  }

  try {
    const parsed = JSON.parse(raw);
    const customSources = dedupeSources(Array.isArray(parsed?.custom_sources) ? parsed.custom_sources : []);
    const selectedSource = normalizeSourceUrl(parsed?.selected_source);
    const allSources = [...BUILTIN_CDN_SOURCES, ...customSources];

    return {
      enabled: Boolean(parsed?.enabled),
      selected_source: allSources.includes(selectedSource) ? selectedSource : BUILTIN_CDN_SOURCES[0],
      custom_sources: customSources,
    };
  } catch {
    return defaults;
  }
}

export function setCDNSettings(settings) {
  const customSources = dedupeSources(Array.isArray(settings?.custom_sources) ? settings.custom_sources : []);
  const selectedSource = normalizeSourceUrl(settings?.selected_source);
  const allSources = [...BUILTIN_CDN_SOURCES, ...customSources];

  const normalized = {
    enabled: Boolean(settings?.enabled),
    selected_source: allSources.includes(selectedSource) ? selectedSource : BUILTIN_CDN_SOURCES[0],
    custom_sources: customSources,
  };

  window.localStorage.setItem(CDN_SETTINGS_KEY, JSON.stringify(normalized));
  return normalized;
}
