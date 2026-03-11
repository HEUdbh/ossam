export const DOWNLOAD_DIR_KEY = "ossam.settings.downloadDirectory";

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
