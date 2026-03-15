export const CATEGORY_ORDER = [
  "Security",
  "DevTools",
  "AI",
  "System",
  "Network",
  "Database",
  "Terminal",
  "Utility",
];

export const CATEGORY_LABELS = {
  Security: "安全工具",
  DevTools: "开发工具",
  AI: "AI工具",
  System: "系统增强",
  Network: "网络插件",
  Database: "数据管理",
  Terminal: "终端效率",
  Utility: "效率办公",
};

export function getCategoryDisplayName(category) {
  const normalized = String(category || "").trim();
  return CATEGORY_LABELS[normalized] || normalized;
}

export function sortCategories(categories) {
  return [...categories].sort((left, right) => {
    const leftIndex = CATEGORY_ORDER.indexOf(left);
    const rightIndex = CATEGORY_ORDER.indexOf(right);

    if (leftIndex === -1 && rightIndex === -1) {
      return left.localeCompare(right);
    }
    if (leftIndex === -1) {
      return 1;
    }
    if (rightIndex === -1) {
      return -1;
    }
    return leftIndex - rightIndex;
  });
}
