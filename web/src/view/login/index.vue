<template>
  <!-- 后台入口：单屏居中。
       以「之前」那版的暖色性格为底（奶油底 / 大圆角 logo / 绿标题 / 标语 / 白卡 / 绿按钮），
       补上扁平设计、分层阴影与一点科技感：柔光、细点阵、状态点、渐变按钮。
       不再有右半屏绿块，也不留任何解释账号 / 权限的提示语。 -->
  <div id="userLayout" class="login-bg">
    <!-- 氛围层用「纯色圆 + 滤镜模糊」，不用渐变：渐变一铺开就成了那块难看的绿。 -->
    <div class="login-orb login-orb--green" aria-hidden="true"></div>
    <div class="login-orb login-orb--amber" aria-hidden="true"></div>
    <div class="login-grid" aria-hidden="true"></div>

    <div class="login-col">
      <header class="login-head">
        <div class="login-logo-wrap">
          <img src="/logo.png" alt="小任务农场" class="login-logo" />
        </div>
        <p class="login-kicker">
          <span class="login-kicker-dot" aria-hidden="true"></span>运营后台
        </p>
        <h1 class="login-title">小任务农场</h1>
        <p class="login-slogan">完成小任务，经营大农场</p>
      </header>

      <div class="login-card">
        <div class="login-card-head">
          <span class="login-card-line" aria-hidden="true"></span>
          <span class="login-card-label">欢迎回来</span>
          <span class="login-card-line" aria-hidden="true"></span>
        </div>

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

      <p class="login-foot">
        <span class="login-foot-dot" aria-hidden="true"></span>
        小任务农场 · 运营后台
      </p>
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
    position: relative;
    min-height: 100vh;
    overflow: hidden;
    background: #fffcf3;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px 24px;
  }

  /* 柔光：纯色 + blur。刻意不用渐变 —— 渐变铺开就是那块被否掉的半屏绿。 */
  .login-orb {
    position: absolute;
    border-radius: 50%;
    filter: blur(72px);
    pointer-events: none;
  }
  .login-orb--green {
    width: 420px;
    height: 420px;
    top: -140px;
    left: -120px;
    background: #34d399;
    opacity: 0.2;
  }
  .login-orb--amber {
    width: 380px;
    height: 380px;
    right: -120px;
    bottom: -150px;
    background: #fbbf24;
    opacity: 0.16;
  }
  /* 细点阵：SVG 平铺（background-image 是 url，不是 gradient），给一点科技感。 */
  .login-grid {
    position: absolute;
    inset: 0;
    opacity: 0.07;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='26' height='26'%3E%3Ccircle cx='2' cy='2' r='1.1' fill='%2315803d'/%3E%3C/svg%3E");
    pointer-events: none;
  }

  .login-col {
    position: relative;
    z-index: 1;
    width: 100%;
    max-width: 400px;
  }
  .login-head {
    text-align: center;
    margin-bottom: 22px;
  }
  .login-logo-wrap {
    width: 76px;
    height: 76px;
    margin: 0 auto 18px;
    border-radius: 22px;
    background: #ffffff;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow:
      0 1px 2px rgba(20, 83, 45, 0.06),
      0 10px 24px -10px rgba(20, 83, 45, 0.35),
      0 0 0 1px rgba(21, 128, 61, 0.1);
  }
  .login-logo {
    width: 52px;
    height: 52px;
    border-radius: 14px;
    object-fit: contain;
    display: block;
  }
  .login-kicker {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    margin: 0 0 10px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.24em;
    color: #15803d;
  }
  .login-kicker-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.18);
    animation: login-pulse 2.4s ease-in-out infinite;
  }
  @keyframes login-pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.45;
    }
  }
  .login-title {
    margin: 0;
    font-size: 30px;
    font-weight: 800;
    letter-spacing: 0.04em;
    color: #14532d;
    line-height: 1.25;
  }
  .login-slogan {
    margin: 9px 0 0;
    font-size: 13px;
    letter-spacing: 0.08em;
    color: #4b7a5c;
  }

  .login-card {
    background: #ffffff;
    border: 1px solid rgba(21, 128, 61, 0.1);
    border-radius: 20px;
    padding: 26px 26px 22px;
    /* 分层阴影：近处一层薄边、中景一层主影、远景一层扩散 —— 扁平但有厚度 */
    box-shadow:
      0 1px 2px rgba(20, 83, 45, 0.05),
      0 14px 34px -14px rgba(20, 83, 45, 0.28),
      0 30px 70px -40px rgba(20, 83, 45, 0.22);
  }
  .login-card-head {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 22px;
  }
  .login-card-line {
    flex: 1;
    height: 1px;
    background: #e8efe9;
  }
  .login-card-label {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.16em;
    color: #6b7c6f;
    white-space: nowrap;
  }

  .login-form .el-form-item {
    margin-bottom: 18px;
  }
  .login-form .el-form-item__content {
    line-height: normal;
  }
  .login-form .el-input__wrapper {
    background: #f7faf6 !important;
    box-shadow: none !important;
    border: 1px solid #e3ece4 !important;
    border-radius: 11px !important;
    padding: 1px 12px !important;
    transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
  }
  .login-form .el-input__wrapper:hover {
    border-color: #b9d8c2 !important;
  }
  .login-form .el-input__wrapper.is-focus {
    background: #ffffff !important;
    border-color: #15803d !important;
    box-shadow: 0 0 0 3px rgba(21, 128, 61, 0.12) !important;
  }
  .login-form .el-input__inner {
    height: 44px;
    font-size: 14px;
    color: #14532d;
  }
  .login-form .el-input__inner::placeholder {
    color: #9aa8a0;
  }
  .login-captcha {
    width: 33%;
    height: 46px;
    border: 1px solid #e3ece4;
    border-radius: 11px;
    overflow: hidden;
    background: #f7faf6;
  }
  .login-actions {
    margin-top: 26px;
    margin-bottom: 0 !important;
  }
  .login-submit.el-button {
    width: 100%;
    height: 46px;
    border: 0 !important;
    border-radius: 12px !important;
    background: linear-gradient(135deg, #22a55b 0%, #15803d 100%) !important;
    background-color: #15803d !important;
    box-shadow:
      0 8px 20px -6px rgba(21, 128, 61, 0.55),
      inset 0 1px 0 rgba(255, 255, 255, 0.22) !important;
    font-size: 15px;
    font-weight: 600;
    letter-spacing: 0.16em;
    color: #ffffff !important;
    transition: transform 0.18s, box-shadow 0.18s;
  }
  .login-submit.el-button:hover {
    transform: translateY(-1px);
    box-shadow:
      0 12px 26px -6px rgba(21, 128, 61, 0.65),
      inset 0 1px 0 rgba(255, 255, 255, 0.22) !important;
    color: #ffffff !important;
  }

  .login-foot {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin: 20px 0 0;
    font-size: 11px;
    letter-spacing: 0.12em;
    color: #9aa99e;
  }
  .login-foot-dot {
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: #a7c4ae;
  }

  @media (max-width: 480px) {
    .login-bg {
      padding: 28px 16px;
    }
    .login-orb--green {
      width: 260px;
      height: 260px;
    }
    .login-orb--amber {
      width: 240px;
      height: 240px;
    }
    .login-card {
      padding: 22px 20px 18px;
    }
    .login-title {
      font-size: 26px;
    }
  }
</style>
