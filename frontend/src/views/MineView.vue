<template>
  <section class="mine-page">
    <header class="mine-header">
      <h2>我的</h2>
      <el-tag type="success" effect="plain">阶段占位</el-tag>
    </header>

    <p class="mine-desc">后续将接入登录态、个人偏好同步与常用应用管理。</p>

    <div class="feature-grid">
      <div class="feature-item">账号信息</div>
      <div class="feature-item">偏好配置</div>
      <div class="feature-item">下载历史</div>
    </div>
  </section>
</template>

<style scoped>
.mine-page {
  height: 100%;
  padding: 18px;
  overflow: auto;
  background: var(--surface-page);
}

.mine-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--line-color);
}

.mine-header h2 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
}

.mine-desc {
  margin: 14px 0 0;
  color: var(--text-secondary);
  line-height: 1.6;
}

.feature-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  border-top: 1px solid var(--line-color);
  border-left: 1px solid var(--line-color);
}

.feature-item {
  border-right: 1px solid var(--line-color);
  border-bottom: 1px solid var(--line-color);
  padding: 14px;
  color: var(--text-secondary);
  font-size: 14px;
  background: var(--surface-container);
}
</style>
