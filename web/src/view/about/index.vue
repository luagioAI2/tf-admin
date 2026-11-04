<template>
  <!-- 「关于」只讲这个产品是什么、谁从哪儿进来。
       原来这一页是框架作者的名片 + GitHub 提交记录（还会去请求 api.github.com），
       对运营和家长都是噪音，网络不通时整页还是空的。 -->
  <div class="mt-2">
    <el-card shadow="never" class="max-w-4xl">
      <template #header>
        <div class="flex items-center gap-3">
          <img src="/logo.png" alt="小任务农场" class="h-10 w-10 rounded-lg" />
          <div>
            <div class="text-base font-semibold text-slate-800">
              小任务农场 · 运营后台
            </div>
            <div class="text-xs text-slate-400">v{{ version }}</div>
          </div>
        </div>
      </template>

      <el-divider content-position="left">四个入口</el-divider>
      <el-table :data="entrances" size="small" border>
        <el-table-column prop="who" label="谁" width="140" />
        <el-table-column prop="where" label="在哪儿用" width="180" />
        <el-table-column prop="what" label="能做什么" />
      </el-table>

      <el-divider content-position="left">数据</el-divider>
      <p class="text-sm leading-7 text-slate-600">
        孩子、积分、兑换、心愿与 App 共用一份库；后台账号、菜单、权限是另一套表。
      </p>
    </el-card>
  </div>
</template>

<script setup>
  import packageInfo from '../../../package.json'

  defineOptions({
    name: 'About'
  })

  const version = packageInfo.version

  const entrances = [
    { who: '孩子', where: 'App', what: '做任务、看积分、逛兑换商城' },
    { who: '家长', where: 'App + 家长中心', what: '发任务、许愿、确认兑现、看兑换记录' },
    { who: '商家', where: '商家中心', what: '上架自己的商品、看入驻状态' },
    { who: '运营', where: '这个后台', what: '审核招商、管商品与定价、看经营数据' }
  ]
</script>
