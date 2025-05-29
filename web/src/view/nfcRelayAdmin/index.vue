<!--
  NFC中继管理模块父级路由页面
  用于承载子路由的容器页面
-->
<template>
  <div class="nfc-relay-admin-layout">
    <router-view />
  </div>
</template>

<script setup>
defineOptions({
  name: 'NfcRelayAdminLayout'
})
</script>

<style scoped lang="scss">
.nfc-relay-admin-layout {
  width: 100%;
  height: 100%;
}
</style> 