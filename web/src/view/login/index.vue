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

            <!-- 一个后台，**一个输入框**。
                 家长 / 商家不是另一套账号，也不是另一种「身份」——他们用的就是
                 App 的手机号和密码，登录后看到什么由 `sys_authorities` 决定。
                 所以这里没有「管理员 / 家长 / 商家」的切换：
                 那个开关等于让用户自己声明权限，选错的表现是「菜单少了一半」，
                 用户会以为自己的申请没过。 -->
            <el-form
              ref="loginForm"
              :model="loginFormData"
              :rules="rules"
              :validate-on-rule-change="false"
              @keyup.enter="submitForm"
            >
              <el-form-item prop="account" class="mb-6">
                <el-input
                  v-model="loginFormData.account"
                  size="large"
                  placeholder="用户名 / App 手机号"
                  suffix-icon="user"
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
                   账号填的是手机号（App 那条路）时不显示：那条路已经有服务端的
                   bcrypt + 审核闸门，再加一层只是让家长多输一次。 -->
              <el-form-item
                v-if="!isMobileInput && loginFormData.openCaptcha"
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

              <div class="text-xs leading-5 text-gray-500 mb-4">
                <p>· 家长 / 商家直接用 <b>App 的手机号和密码</b>，不用另记账号。</p>
                <p>
                  · 能看到哪些页面由<b>账号的权限</b>决定；家长 / 商家要先在 App 里申请并通过审核。
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

  const submitting = ref(false)
  const subtitle = computed(() => '运营后台 · 家长 / 商家用 App 账号登录')

  // 验证函数
  //
  // 一个框装两种账号：后台用户名（`admin`）和 App 手机号（`13800138000`）。
  // 这里只挡「明显不是账号」的输入 —— 手机号的口径在服务端 `normalizeMobile`，
  // 前端再判一遍只会把「138-0013-8000」这种正常写法判错。
  const checkAccount = (rule, value, callback) => {
    const v = (value || '').trim()
    if (v.length < 3) {
      return callback(new Error('请输入用户名或手机号'))
    }
    callback()
  }
  const checkPassword = (rule, value, callback) => {
    if (value.length < 6) {
      return callback(new Error('请输入正确的密码'))
    } else {
      callback()
    }
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
    /** 后台用户名 **或** App 手机号 —— 同一个框，按内容分路 */
    account: 'admin',
    password: '',
    captcha: '',
    captchaId: '',
    openCaptcha: false
  })

  /** 「像不像手机号」= 去掉空格 / 横线后是 8 位以上数字（可带 +）。
   *  像手机号就走 App 那条登录路，否则走后台用户名。 */
  const isMobileInput = computed(() =>
    /^\+?\d{8,20}$/.test((loginFormData.account || '').replace(/[\s-]/g, ''))
  )

  const rules = {
    account: [{ validator: checkAccount, trigger: 'blur' }],
    password: [{ validator: checkPassword, trigger: 'blur' }],
    captcha: [{ validator: checkCaptcha, trigger: 'blur' }]
  }

  const userStore = useUserStore()
  const login = async () => {
    const account = (loginFormData.account || '').trim()
    if (isMobileInput.value) {
      /* App 账号那条路：服务端按手机号找 `user`，再按**审核记录**定角色。
         这里不传角色 —— 传了就等于让前端替服务端决定权限。 */
      return await userStore.PortalLoginIn({
        mobile: account,
        password: loginFormData.password
      })
    }
    return await userStore.LoginIn({
      username: account,
      password: loginFormData.password,
      captcha: loginFormData.captcha,
      captchaId: loginFormData.captchaId
    })
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
          if (!isMobileInput.value) await loginVerify()
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
