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

      <p class="text-sm leading-7 text-slate-600">
        孩子在家里完成小任务（收拾玩具、按时睡觉……）拿到积分，用积分在兑换商城里换奖品；
        家长负责许愿和确认兑现，商家负责把自己的商品上架，运营负责审核和上架。
      </p>

      <el-divider content-position="left">四个入口</el-divider>
      <el-table :data="entrances" size="small" border>
        <el-table-column prop="who" label="谁" width="140" />
        <el-table-column prop="where" label="在哪儿用" width="180" />
        <el-table-column prop="what" label="能做什么" />
      </el-table>

      <el-divider content-position="left">账号</el-divider>
      <ul class="list-disc pl-5 text-sm leading-7 text-slate-600">
        <li>家长 / 商家<b>不另开账号</b>：用的就是 App 的手机号和密码，先在 App 里提交申请，通过审核后就能登录这里。</li>
        <li>登录后看到哪些页面由<b>账号的权限</b>决定，登录页不需要选身份。</li>
        <li>运营账号由后台的「用户管理」创建。</li>
      </ul>

      <el-divider content-position="left">数据</el-divider>
      <p class="text-sm leading-7 text-slate-600">
        孩子、积分、兑换、心愿这些数据都在平台库里，和 App 是同一份；
        后台自己的账号、菜单、权限是另一套表。所以「用户管理」里的人数
        不等于 App 的家长人数 —— App 用户请看「商城运营 → App 用户」。
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
