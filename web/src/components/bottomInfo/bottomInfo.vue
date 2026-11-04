<template>
  <!-- 页脚只留产品自己的话。原来这里是「Powered by Gin-Vue-Admin /
       Copyright flipped-aurora团队」，家长和商家看到只会以为走错站了。 -->
  <div
    class="flex flex-col md:flex-row gap-2 items-center text-xs text-slate-400 dark:text-slate-500 justify-center py-3"
  >
    <span>小任务农场 · 运营后台</span>
    <slot />
  </div>
</template>

<script setup>
  defineOptions({
    name: 'BottomInfo'
  })
</script>
