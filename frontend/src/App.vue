<script setup lang="ts">
</script>

<template>
  <view class="app-root" />
</template>

<style>
page {
  background: #f5f6f8;
  color: #1a1a2e;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  -webkit-user-select: text;
  user-select: text;
}
.app-root {
  min-height: 100vh;
}
/* uni-app H5：view/text 默认可选中，避免整页无法复制 */
page,
view,
uni-view,
uni-text,
text,
div,
span,
p,
label {
  -webkit-user-select: text;
  user-select: text;
}
input,
textarea,
select,
.uni-input-input {
  -webkit-text-security: none !important;
  text-security: none !important;
  color: #334155 !important;
  -webkit-user-select: text;
  user-select: text;
}
/* 仅按钮/芯片等交互控件禁用选中，避免影响点击 */
button,
uni-button,
.chip,
.btn,
.btn-add-lib,
.btn-save-lib,
.risk-link {
  -webkit-user-select: none;
  user-select: none;
  pointer-events: auto !important;
  cursor: pointer;
}
</style>
