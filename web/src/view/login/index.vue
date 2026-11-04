<template>
  <div id="userLayout" class="w-full h-full relative">
    <div
      class="rounded-lg flex items-center justify-evenly w-full h-full md:w-screen md:h-screen md:bg-[#194bfb] bg-white"
    >
      <div class="md:w-3/5 w-10/12 h-full flex items-center justify-evenly">
        <div
          class="oblique h-[130%] w-3/5 bg-white dark:bg-slate-900 transform -rotate-12 absolute -ml-52"
        />
        <!-- 分割斜块 -->
        <div
          class="z-[999] pt-12 pb-10 md:w-96 w-full rounded-lg flex flex-col justify-between box-border"
        >
          <div>
            <div class="flex items-center justify-center">
              <Logo :size="6" />
            </div>
            <div class="mb-6">
              <p class="text-center text-4xl font-bold">
                {{ $GIN_VUE_ADMIN.appName }}
              </p>
              <p class="text-center text-sm font-normal text-gray-500 mt-2.5">
                {{ subtitle }}
              </p>
            </div>

            <!-- 三种身份，一个后台。
                 家长 / 商家**不是另一套账号**：用的是 App 的手机号和密码，
                 登录时换出对应的角色；准入看 `merchant_application` 有没有通过审核。 -->
            <div class="flex justify-center mb-6">
              <el-radio-group v-model="mode" size="large" @change="onModeChange">
                <el-radio-button value="admin">管理员</el-radio-button>
                <el-radio-button value="parent">家长</el-radio-button>
                <el-radio-button value="merchant">商家</el-radio-button>
              </el-radio-group>
            </div>

            <el-form
              ref="loginForm"
              :model="loginFormData"
              :rules="rules"
              :validate-on-rule-change="false"
              @keyup.enter="submitForm"
            >
              <el-form-item v-if="!isPortal" prop="username" class="mb-6">
                <el-input
                  v-model="loginFormData.username"
                  size="large"
                  placeholder="请输入用户名"
                  suffix-icon="user"
                />
              </el-form-item>
              <el-form-item v-else prop="mobile" class="mb-6">
                <el-input
                  v-model="loginFormData.mobile"
                  size="large"
                  placeholder="请输入手机号（App 登录用的那个）"
                  suffix-icon="iphone"
                />
              </el-form-item>

              <el-form-item prop="password" class="mb-6">
                <el-input
                  v-model="loginFormData.password"
                  show-password
                  size="large"
                  type="password"
                  placeholder="请输入密码"
                />
              </el-form-item>

              <!-- 验证码只挡「用户名 + 密码」这条最容易被撞的路。
                   App 账号那条路已经有服务端的 bcrypt + 审核闸门，
                   再加一层验证码只是让家长多输一次。 -->
              <el-form-item
                v-if="!isPortal && loginFormData.openCaptcha"
                prop="captcha"
                class="mb-6"
              >
                <div class="flex w-full justify-between">
                  <el-input
                    v-model="loginFormData.captcha"
                    placeholder="请输入验证码"
                    size="large"
                    class="flex-1 mr-5"
                  />
                  <div class="w-1/3 h-11 bg-[#c3d4f2] rounded">
                    <img
                      v-if="picPath"
                      class="w-full h-full"
                      :src="picPath"
                      alt="请输入验证码"
                      @click="loginVerify()"
                    />
                  </div>
                </div>
              </el-form-item>

              <el-form-item class="mb-4">
                <el-button
                  class="shadow shadow-active h-11 w-full"
                  type="primary"
                  size="large"
                  :loading="submitting"
                  @click="submitForm"
                  >登 录</el-button
                >
              </el-form-item>

              <div v-if="isPortal" class="text-xs leading-5 text-gray-500 mb-4">
                <p>· 用 App 注册的<b>手机号</b>和<b>密码</b>登录，不用另记账号。</p>
                <p>
                  · 需要先在 App 里提交「{{ mode === 'parent' ? '家长' : '商家' }}」申请并通过审核。
                </p>
              </div>

              <el-form-item v-if="isDev" class="mb-6">
                <el-button
                  class="shadow shadow-active h-11 w-full"
                  type="primary"
                  size="large"
                  @click="checkInit"
                  >前往初始化</el-button
                >
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>
      <div class="hidden md:block w-1/2 h-full float-right bg-[#194bfb]">
        <img
          class="h-full"
          src="@/assets/login_right_banner.jpg"
          alt="banner"
        />
      </div>
    </div>

    <BottomInfo class="left-0 right-0 absolute bottom-3 mx-auto w-full z-20">
      <div class="links items-center justify-center gap-2 hidden md:flex">
        <a href="https://www.gin-vue-admin.com/" target="_blank">
          <img src="@/assets/docs.png" class="w-8 h-8" alt="文档" />
        </a>
        <a href="https://support.qq.com/product/371961" target="_blank">
          <img src="@/assets/kefu.png" class="w-8 h-8" alt="客服" />
        </a>
        <a
          href="https://github.com/flipped-aurora/gin-vue-admin"
          target="_blank"
        >
          <img src="@/assets/github.png" class="w-8 h-8" alt="github" />
        </a>
        <a href="https://space.bilibili.com/322210472" target="_blank">
          <img src="@/assets/video.png" class="w-8 h-8" alt="视频站" />
        </a>
      </div>
    </BottomInfo>
  </div>
</template>

<script setup>
  import { captcha } from '@/api/user'
  import { checkDB } from '@/api/initdb'
  import BottomInfo from '@/components/bottomInfo/bottomInfo.vue'
  import { computed, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useRouter } from 'vue-router'
  import { useUserStore } from '@/pinia/modules/user'
  import Logo from '@/components/logo/index.vue'
  import { isDev } from '@/utils/env.js'

  defineOptions({
    name: 'Login'
  })

  const router = useRouter()
  const captchaRequiredLength = ref(6)

  // admin = 后台自己的账号；parent / merchant = App 账号 + 审核通过
  const mode = ref('admin')
  const isPortal = computed(() => mode.value !== 'admin')
  const submitting = ref(false)
  const subtitle = computed(() => {
    if (mode.value === 'parent') return '家长中心 · 用 App 账号登录'
    if (mode.value === 'merchant') return '商家中心 · 用 App 账号登录'
    return 'A management platform using Golang and Vue'
  })

  // 验证函数
  const checkUsername = (rule, value, callback) => {
    if (value.length < 5) {
      return callback(new Error('请输入正确的用户名'))
    } else {
      callback()
    }
  }
  const checkPassword = (rule, value, callback) => {
    if (value.length < 6) {
      return callback(new Error('请输入正确的密码'))
    } else {
      callback()
    }
  }
  // 手机号：只做「像不像」的校验，真正的口径在服务端（`normalizeMobile`）。
  // 这里放宽到「8 位以上数字、可带 +」—— 号码里带空格 / 横线的由服务端清，
  // 前端提前拦会把「138-0013-8000」这种正常输入判错。
  const checkMobile = (rule, value, callback) => {
    const v = (value || '').replace(/[\s-]/g, '')
    if (!/^\+?\d{8,20}$/.test(v)) {
      return callback(new Error('请输入正确的手机号'))
    }
    callback()
  }
  const checkCaptcha = (rule, value, callback) => {
    if (!loginFormData.openCaptcha) {
      return callback()
    }
    const sanitizedValue = (value || '').replace(/\s+/g, '')
    if (!sanitizedValue) {
      return callback(new Error('请输入验证码'))
    }
    if (!/^\d+$/.test(sanitizedValue)) {
      return callback(new Error('验证码须为数字'))
    }
    if (sanitizedValue.length < captchaRequiredLength.value) {
      return callback(
        new Error(`请输入至少${captchaRequiredLength.value}位数字验证码`)
      )
    }
    if (sanitizedValue !== value) {
      loginFormData.captcha = sanitizedValue
    }
    callback()
  }

  // 获取验证码
  const loginVerify = async () => {
    const ele = await captcha()
    captchaRequiredLength.value = Number(ele.data?.captchaLength) || 0
    picPath.value = ele.data?.picPath
    loginFormData.captchaId = ele.data?.captchaId
    loginFormData.openCaptcha = ele.data?.openCaptcha
  }
  loginVerify()

  // 登录相关操作
  const loginForm = ref(null)
  const picPath = ref('')
  const loginFormData = reactive({
    username: 'admin',
    mobile: '',
    password: '',
    captcha: '',
    captchaId: '',
    openCaptcha: false
  })

  // 规则跟着身份走：切到家长 / 商家时不能再要求「用户名 5 位以上」
  const rules = computed(() => {
    if (isPortal.value) {
      return {
        mobile: [{ validator: checkMobile, trigger: 'blur' }],
        password: [{ validator: checkPassword, trigger: 'blur' }]
      }
    }
    return {
      username: [{ validator: checkUsername, trigger: 'blur' }],
      password: [{ validator: checkPassword, trigger: 'blur' }],
      captcha: [{ validator: checkCaptcha, trigger: 'blur' }]
    }
  })

  // 切身份时把上一种身份的红字清掉（否则「用户名太短」会挂在手机号那一栏旁边）
  const onModeChange = () => {
    loginForm.value?.clearValidate()
  }

  const userStore = useUserStore()
  const login = async () => {
    if (isPortal.value) {
      return await userStore.PortalLoginIn({
        mobile: loginFormData.mobile,
        password: loginFormData.password,
        role: mode.value
      })
    }
    return await userStore.LoginIn(loginFormData)
  }
  const submitForm = () => {
    if (submitting.value) return
    loginForm.value.validate(async (v) => {
      if (!v) {
        // 未通过前端静态验证
        ElMessage({
          type: 'error',
          message: '请正确填写登录信息',
          showClose: true
        })
        return false
      }

      submitting.value = true
      try {
        // 通过验证，请求登陆
        const flag = await login()

        // 登陆失败，刷新验证码
        if (!flag) {
          if (!isPortal.value) await loginVerify()
          return false
        }

        // 登陆成功
        return true
      } finally {
        submitting.value = false
      }
    })
  }

  // 跳转初始化
  const checkInit = async () => {
    const res = await checkDB()
    if (res.code === 0) {
      if (res.data?.needInit) {
        userStore.NeedInit()
        await router.push({ name: 'Init' })
      } else {
        ElMessage({
          type: 'info',
          message: '已配置数据库信息，无法初始化'
        })
      }
    }
  }
</script>
