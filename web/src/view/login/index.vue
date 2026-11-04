<template>
  <!-- 配色和图标跟 App 同一套（sun-50 奶油底 + grass 绿），
       家长 / 商家从 App 点过来要一眼认出是同一个产品。 -->
  <div id="userLayout" class="w-full h-full relative bg-[#fffbeb]">
    <div class="flex w-full h-full md:w-screen md:h-screen">
      <div class="flex-1 flex items-center justify-center px-6">
        <div class="w-full max-w-sm py-10">
          <div class="flex flex-col items-center mb-8">
            <img
              src="/logo.png"
              alt="小任务农场"
              class="w-16 h-16 rounded-2xl shadow-md"
            />
            <p class="text-center text-3xl font-bold mt-4 text-[#14532d]">
              {{ $GIN_VUE_ADMIN.appName }}
            </p>
            <p class="text-center text-sm font-normal text-gray-500 mt-2">
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
            </el-form>
          </div>
        </div>
      <!-- 右边只讲「这是谁的后台」，不放框架的演示图：
           家长 / 商家不关心底层用什么写的，看到陌生品牌只会以为走错站了。 -->
      <div
        class="hidden md:flex w-1/2 items-center justify-center bg-gradient-to-br from-[#22c55e] to-[#15803d]"
      >
        <div class="text-center px-10">
          <img
            src="/logo.png"
            alt="小任务农场"
            class="w-40 h-40 rounded-[32px] shadow-2xl mx-auto"
          />
          <p class="mt-8 text-3xl font-bold text-white">小任务农场</p>
          <p class="mt-3 text-white/85">完成小任务，经营大农场</p>
          <p class="mt-10 text-xs leading-6 text-white/70">
            家长中心 · 商家中心 · 运营后台<br />同一个账号，看到各自的内容
          </p>
        </div>
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

</script>
