<template>
  <div class="w-full h-screen bg-[#fffcf3] flex items-center justify-center">
    <div class="flex flex-col items-center gap-4 text-center">
      <img class="w-1/3" src="../../assets/404.png" />
      <p class="text-lg font-semibold text-[#14532d]">这个页面找不到了</p>
      <p class="text-sm text-slate-500">可以回首页，或从左边菜单重新进。</p>
      <el-button type="primary" @click="toDashboard">返回首页</el-button>
    </div>
  </div>
</template>

<script setup>
  import { useUserStore } from '@/pinia/modules/user'
  import { useRouter } from 'vue-router'
  import { emitter } from '@/utils/bus'

  defineOptions({
    name: 'Error'
  })

  const userStore = useUserStore()
  const router = useRouter()
  const toDashboard = () => {
    try {
      router.push({ name: userStore.userInfo.authority.defaultRouter })
    } catch (error) {
        emitter.emit('show-error', {
        code: '401',
        message: "检测到其他用户修改了路由权限，请重新登录",
        fn: () => {
          userStore.ClearStorage()
          router.push({ name: 'Login', replace: true })
        }
      })
    }
  }
</script>
