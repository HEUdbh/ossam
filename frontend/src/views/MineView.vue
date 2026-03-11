<template>
  <section class="mine-page">
    <el-card class="mine-card" shadow="never">
      <template #header>
        <div class="card-header">
          <h2>我的</h2>
          <el-tag type="success" effect="plain" round>阶段占位</el-tag>
        </div>
      </template>

      <p class="card-desc">后续将接入登录态、个人偏好同步与常用应用清单。</p>

      <div class="feature-grid">
        <div class="feature-item">账号信息</div>
        <div class="feature-item">偏好配置</div>
        <div class="feature-item">下载历史</div>
      </div>
    </el-card>
  </section>
</template>

<style scoped>
.mine-page {
  height: 100%;
  padding: 18px;
  overflow: auto;
}

.mine-card {
  border-radius: var(--radius-lg);
  border-color: var(--line-color);
}

.mine-card :deep(.el-card__body) {
  background: linear-gradient(180deg, #ffffff, #f8fdf9);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.card-header h2 {
  margin: 0;
  font-size: 22px;
}

.card-desc {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.6;
}

.feature-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}

.feature-item {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-md);
  background: var(--surface-2);
  padding: 14px;
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
