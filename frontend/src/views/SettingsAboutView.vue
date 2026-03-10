<template>
  <section class="settings-block">
    <h3>关于作者</h3>
    <p>阶段1占位：后续补充作者介绍、项目愿景与联系方式。</p>

    <el-descriptions :column="1" border class="about-table">
      <el-descriptions-item label="作者">待补充</el-descriptions-item>
      <el-descriptions-item label="项目主页">待补充</el-descriptions-item>
      <el-descriptions-item label="反馈渠道">待补充</el-descriptions-item>
    </el-descriptions>
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

.about-table {
  margin-top: 14px;
}
</style>
