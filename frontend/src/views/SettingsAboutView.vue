<template>
  <section class="settings-block">
    <h3>关于作者</h3>
    <p>欢迎使用 ossam，以下是作者与项目联系信息。</p>

    <el-descriptions :column="1" border class="about-table">
      <el-descriptions-item label="作者">夜天炫安全-HEUdbh</el-descriptions-item>
      <el-descriptions-item label="仓库主页">
        <a
          class="about-link"
          href="https://ghproxy.net/https://github.com/HEUdbh/ossam"
          target="_blank"
          rel="noreferrer"
        >
          https://github.com/HEUdbh/ossam
        </a>
      </el-descriptions-item>
      <el-descriptions-item label="反馈途径">
        <a class="about-link" href="mailto:heuxry@outlook.com">heuxry@outlook.com</a>
      </el-descriptions-item>
    </el-descriptions>

    <el-alert class="contribution-tip" type="info" :closable="false" show-icon>
      <template #title>社区补充说明</template>
      <p>
        欢迎广大网友补充 GitHub 开源项目连接加入应用市场，请按标准 JSON 格式书写配置文件，可通过邮件发送，或在
        GitHub 发起请求合并（PR）进入项目仓库。
      </p>
    </el-alert>
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
  line-height: 1.65;
}

.about-table {
  margin-top: 14px;
}

.about-table :deep(.el-descriptions__body) {
  background: #ffffff;
}

.about-link {
  color: var(--brand-color);
  text-decoration: none;
  word-break: break-all;
}

.about-link:hover {
  text-decoration: underline;
}

.contribution-tip {
  margin-top: 14px;
}

.contribution-tip :deep(.el-alert__content) {
  width: 100%;
}

.contribution-tip :deep(p) {
  margin: 6px 0 0;
  line-height: 1.65;
}
</style>
