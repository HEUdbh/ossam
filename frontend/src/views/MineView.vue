<template>
  <section class="mine-page">
    <el-card class="mine-card" shadow="never">
      <template #header>
        <div class="card-header">
          <h2>我的</h2>
          <el-tag type="info" effect="plain" round>阶段1占位</el-tag>
        </div>
      </template>

      <p>后续将接入登录态、个人偏好同步与常用应用清单。</p>

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
}

.mine-card {
  border-radius: var(--radius-md);
  border-color: var(--line-color);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
}

p {
  margin: 0;
  color: var(--text-secondary);
}

.feature-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 10px;
}

.feature-item {
  border: 1px solid var(--line-color);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  padding: 12px;
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
