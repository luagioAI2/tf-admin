<template>
  <!-- 后台入口。
       上一版是奶油底 + 大圆角白卡 + 饱和绿按钮 + 标语 + 两行说明，
       整套是「母婴 / 营销页」的配方，看着俗。这一版换成中性纸 + 近黑墨 +
       宋体标题，主色只留一小段线；不加投影、不加标语、不加解释。 -->
  <div id="userLayout" class="login-bg">
    <div class="login-col">
      <header class="login-head">
        <img src="/logo.png" alt="小任务农场" class="login-logo" />
        <p class="login-kicker">运营后台</p>
        <h1 class="login-title">小任务农场</h1>
        <span class="login-rule" aria-hidden="true"></span>
      </header>

      <div class="login-panel">
        <!-- 一个输入框按内容分路：手机号走门户登录，其余走后台登录。 -->
        <el-form
          ref="loginForm"
          class="login-form"
          :model="loginFormData"
          :rules="rules"
          :validate-on-rule-change="false"
          @keyup.enter="submitForm"
        >
          <el-form-item prop="account" class="login-field">
            <el-input
              v-model="loginFormData.account"
              size="large"
              placeholder="用户名 / App 手机号"
              suffix-icon="user"
            />
          </el-form-item>

          <el-form-item prop="password" class="login-field">
            <el-input
              v-model="loginFormData.password"
              show-password
              size="large"
              type="password"
              placeholder="请输入密码"
            />
          </el-form-item>

          <!-- 验证码只挡「用户名 + 密码」这条最容易被撞的路。
               账号填的是手机号时不显示：那条路已经有服务端校验。 -->
          <el-form-item
            v-if="!isMobileInput && loginFormData.openCaptcha"
            prop="captcha"
            class="login-field"
          >
            <div class="flex w-full justify-between">
              <el-input
                v-model="loginFormData.captcha"
                placeholder="请输入验证码"
                size="large"
                class="flex-1 mr-5"
              />
              <div class="login-captcha">
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

          <el-form-item class="login-actions">
            <el-button
              class="login-submit"
              type="primary"
              size="large"
              :loading="submitting"
              @click="submitForm"
              >登 录</el-button
            >
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
  import { captcha } from '@/api/user'
  import { computed, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useUserStore } from '@/pinia/modules/user'

  defineOptions({
    name: 'Login'
  })

  const captchaRequiredLength = ref(6)
  const submitting = ref(false)

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

  /** 「像不像手机号」= 去掉空格 / 横线后是 8 位以上数字（可带 +）。 */
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
      /* App 账号那条路：服务端按手机号找 `user`，再按审核记录定角色。 */
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
        ElMessage({
          type: 'error',
          message: '请正确填写登录信息',
          showClose: true
        })
        return false
      }

      submitting.value = true
      try {
        const flag = await login()
        if (!flag) {
          if (!isMobileInput.value) await loginVerify()
          return false
        }
        return true
      } finally {
        submitting.value = false
      }
    })
  }
</script>

<style>
  /* 登录页是独立路由，用全局样式即可；类名都带 login- 前缀，不会漏到别处。 */
  .login-bg {
    min-height: 100vh;
    background: #fafaf9;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
  }
  .login-col {
    width: 100%;
    max-width: 320px;
  }
  .login-head {
    text-align: center;
    margin-bottom: 28px;
  }
  .login-logo {
    width: 40px;
    height: 40px;
    border-radius: 4px;
    display: block;
    margin: 0 auto 22px;
  }
  .login-kicker {
    font-size: 11px;
    letter-spacing: 0.22em;
    color: #8a8a85;
    margin: 0 0 10px;
  }
  .login-title {
    font-family: 'Songti SC', 'Noto Serif SC', 'SimSun', serif;
    font-size: 27px;
    font-weight: 400;
    letter-spacing: 0.02em;
    color: #16181a;
    margin: 0;
  }
  .login-rule {
    display: block;
    width: 22px;
    height: 2px;
    background: #2f5d4a;
    margin: 18px auto 0;
  }
  .login-panel {
    background: #ffffff;
    border: 1px solid #e7e5e4;
    border-radius: 2px;
    padding: 26px 26px 20px;
  }
  .login-form .el-form-item {
    margin-bottom: 20px;
  }
  .login-form .el-form-item__content {
    line-height: normal;
  }
  .login-form .el-input__wrapper {
    background: transparent !important;
    box-shadow: none !important;
    border: 0 !important;
    border-bottom: 1px solid #d6d3d1 !important;
    border-radius: 0 !important;
    padding: 0 2px !important;
  }
  .login-form .el-input__wrapper.is-focus {
    border-bottom-color: #16181a !important;
  }
  .login-form .el-input__inner {
    height: 44px;
    font-size: 14px;
    color: #16181a;
  }
  .login-form .el-input__inner::placeholder {
    color: #a8a29e;
  }
  .login-captcha {
    width: 33%;
    height: 44px;
    border: 1px solid #e7e5e4;
    overflow: hidden;
  }
  .login-actions {
    margin-top: 30px;
    margin-bottom: 0 !important;
  }
  .login-submit.el-button {
    width: 100%;
    height: 42px;
    background: #16181a !important;
    border-color: #16181a !important;
    border-radius: 2px !important;
    box-shadow: none !important;
    font-size: 14px;
    font-weight: 400;
    letter-spacing: 0.2em;
    color: #ffffff;
  }
  .login-submit.el-button:hover {
    background: #303336 !important;
    border-color: #303336 !important;
  }
</style>
