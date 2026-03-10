<template>
  <section class="settings-block">
    <h3>下载地址设置</h3>
    <p>阶段1占位：后续在此配置下载目录、镜像源优先级与下载策略。</p>

    <div class="option-grid">
      <div class="option-item">
        <span class="label">默认目录</span>
        <span class="value">未配置</span>
      </div>
      <div class="option-item">
        <span class="label">下载源</span>
        <span class="value">待接入</span>
      </div>
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

.option-grid {
  margin-top: 14px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}

.option-item {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-sm);
  background: var(--surface-1);
  padding: 12px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.label {
  color: var(--text-secondary);
}

.value {
  color: var(--text-primary);
  font-weight: 600;
}
</style>
