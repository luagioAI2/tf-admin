<template>
  <!-- 运行状态：服务器自身的资源占用。
       原版是框架自带的诊断页（Runtime / os / cpu nums / Disk 全英文），
       对家长和商家是纯噪音，所以整页改成中文，并且只留「会不会写不进数据」这件事。 -->
  <div class="p-2">
    <div class="mb-4 text-sm text-slate-500">
      这一页看的是服务器自己的资源占用。日常不用看；<b>磁盘写满会直接导致数据存不进去</b>，
      所以磁盘那一栏留个心。
    </div>

    <el-row :gutter="15" class="py-1">
      <el-col :span="12">
        <el-card v-if="state.os" class="state-card">
          <template #header>
            <div class="state-title">服务运行</div>
          </template>
          <div class="state-body">
            <el-row :gutter="10">
              <el-col :span="12">操作系统</el-col>
              <el-col :span="12">{{ state.os.goos }}</el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="12">CPU 核数</el-col>
              <el-col :span="12">{{ state.os.numCpu }}</el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="12">同时处理的任务</el-col>
              <el-col :span="12">{{ state.os.numGoroutine }}</el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="12">服务端版本</el-col>
              <el-col :span="12">{{ state.os.goVersion }}</el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card
          v-if="state.disk"
          class="state-card"
          :body-style="{ height: '200px', 'overflow-y': 'auto' }"
        >
          <template #header>
            <div class="state-title">磁盘</div>
          </template>
          <div class="state-body">
            <el-row
              v-for="(item, index) in state.disk"
              :key="index"
              :gutter="10"
              class="mb-6"
            >
              <el-col :span="13">
                <el-row :gutter="10">
                  <el-col :span="10">位置</el-col>
                  <el-col :span="14" class="break-all">{{ item.mountPoint }}</el-col>
                </el-row>
                <el-row :gutter="10">
                  <el-col :span="10">总容量</el-col>
                  <el-col :span="14">{{ item.totalGb }} GB</el-col>
                </el-row>
                <el-row :gutter="10">
                  <el-col :span="10">已用</el-col>
                  <el-col :span="14">{{ item.usedGb }} GB</el-col>
                </el-row>
                <el-row :gutter="10">
                  <el-col :span="10">剩余</el-col>
                  <el-col :span="14" :class="left(item) < 1 ? 'text-red-600' : ''">
                    {{ left(item) }} GB
                  </el-col>
                </el-row>
              </el-col>
              <el-col :span="11">
                <el-progress
                  type="dashboard"
                  :percentage="item.usedPercent"
                  :color="colors"
                />
              </el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="15" class="py-1">
      <el-col :span="12">
        <el-card
          v-if="state.cpu"
          class="state-card"
          :body-style="{ height: '200px', 'overflow-y': 'auto' }"
        >
          <template #header>
            <div class="state-title">CPU</div>
          </template>
          <div class="state-body">
            <el-row :gutter="10">
              <el-col :span="12">物理核心</el-col>
              <el-col :span="12">{{ state.cpu.cores }}</el-col>
            </el-row>
            <el-row
              v-for="(item, index) in state.cpu.cpus"
              :key="index"
              :gutter="10"
            >
              <el-col :span="8">第 {{ index + 1 }} 核</el-col>
              <el-col :span="16">
                <el-progress
                  type="line"
                  :percentage="+item.toFixed(0)"
                  :color="colors"
                />
              </el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card v-if="state.ram" class="state-card">
          <template #header>
            <div class="state-title">内存</div>
          </template>
          <div class="state-body">
            <el-row :gutter="10">
              <el-col :span="13">
                <el-row :gutter="10">
                  <el-col :span="10">总容量</el-col>
                  <el-col :span="14">{{ (state.ram.totalMb / 1024).toFixed(1) }} GB</el-col>
                </el-row>
                <el-row :gutter="10">
                  <el-col :span="10">已用</el-col>
                  <el-col :span="14">{{ (state.ram.usedMb / 1024).toFixed(2) }} GB</el-col>
                </el-row>
                <el-row :gutter="10">
                  <el-col :span="10">占用</el-col>
                  <el-col :span="14">{{ state.ram.usedPercent }}%</el-col>
                </el-row>
              </el-col>
              <el-col :span="11">
                <el-progress
                  type="dashboard"
                  :percentage="state.ram.usedPercent"
                  :color="colors"
                />
              </el-col>
            </el-row>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
  import { getSystemState } from '@/api/system'
  import { onUnmounted, ref } from 'vue'

  defineOptions({
    name: 'State'
  })

  const timer = ref(null)
  const state = ref({})
  const colors = ref([
    { color: '#5cb87a', percentage: 20 },
    { color: '#e6a23c', percentage: 40 },
    { color: '#f56c6c', percentage: 80 }
  ])

  // 剩余空间（GB）。后端只给 total/used（MB）和百分比，剩余自己算，
  // 并且用「总 - 已用」而不是「总 × (1 - 百分比)」——百分比是取整过的。
  const left = (d) => {
    const v = (d.totalMb - d.usedMb) / 1024
    return v > 0 ? v.toFixed(1) : 0
  }

  const reload = async () => {
    const { data } = await getSystemState()
    state.value = data.server
  }

  reload()
  timer.value = setInterval(() => {
    reload()
  }, 1000 * 10)

  onUnmounted(() => {
    clearInterval(timer.value)
    timer.value = null
  })
</script>

<style scoped>
  .state-card {
    @apply h-72 bg-white dark:bg-slate-800 rounded m-2;
  }
  .state-title {
    @apply text-base font-medium text-slate-700 dark:text-slate-200;
  }
  .state-body {
    @apply text-sm leading-8 text-slate-600 dark:text-slate-300;
  }
</style>
