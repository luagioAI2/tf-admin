import { login, getUserInfo, appLogin } from '@/api/user'
import { jsonInBlacklist } from '@/api/jwt'
import router from '@/router/index'
import { ElLoading, ElMessage } from 'element-plus'
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouterStore } from './router'
import { useCookies } from '@vueuse/integrations/useCookies'
import { useStorage } from '@vueuse/core'

import { useAppStore } from '@/pinia'

export const useUserStore = defineStore('user', () => {
  const appStore = useAppStore()
  const loadingInstance = ref(null)

  const userInfo = ref({
    uuid: '',
    nickName: '',
    headerImg: '',
    authority: {}
  })
  const token = useStorage('token', '')
  const xToken = useCookies()
  const currentToken = computed(() => token.value || xToken.get('x-token') || '')

  const setUserInfo = (val) => {
    userInfo.value = val
    if (val.originSetting) {
      Object.keys(appStore.config).forEach((key) => {
        if (val.originSetting[key] !== undefined) {
          appStore.config[key] = val.originSetting[key]
        }
      })
    }
  }

  const setToken = (val) => {
    token.value = val
    xToken.value = val
  }

  const NeedInit = async () => {
    await ClearStorage()
    await router.push({ name: 'Init', replace: true })
  }

  const ResetUserInfo = (value = {}) => {
    userInfo.value = {
      ...userInfo.value,
      ...value
    }
  }
  /* 获取用户信息*/
  const GetUserInfo = async () => {
    const res = await getUserInfo()
    if (res.code === 0) {
      setUserInfo(res.data.userInfo)
    }
    return res
  }
  /* 登录成功之后要做的事：存用户、存 token、装动态路由、跳到首页
   *
   * ★ 抽出来的原因：家长 / 商家走的是**另一个登录接口**（`/base/appLogin`），
   * 但登录成功之后的一切（菜单、默认页、路由注册）必须和管理员**完全一样**。
   * 抄一份的话，两边迟早分叉 —— 而分叉的表现是「家长登录成功、停在空白页」。
   */
  const afterLogin = async (res) => {
    // 登陆成功，设置用户信息和权限相关信息
    setUserInfo(res.data.user)
    setToken(res.data.token)

    // 初始化路由信息
    const routerStore = useRouterStore()
    await routerStore.SetAsyncRouter()
    const asyncRouters = routerStore.asyncRouters

    // 注册到路由表里
    asyncRouters.forEach((asyncRouter) => {
      router.addRoute(asyncRouter)
    })

    if (router.currentRoute.value.query.redirect) {
      await router.replace(router.currentRoute.value.query.redirect)
      return true
    }

    // defaultRouter 是**路由 name**，和菜单 name 对不上时这里只会弹一句提示，
    // 界面停在空白 —— 排查时先看 `sys_authorities.default_router`
    if (!router.hasRoute(userInfo.value.authority.defaultRouter)) {
      ElMessage.error('不存在可以登陆的首页，请联系管理员进行配置')
    } else {
      await router.replace({ name: userInfo.value.authority.defaultRouter })
    }

    const isWindows = /windows/i.test(navigator.userAgent)
    window.localStorage.setItem('osType', isWindows ? 'WIN' : 'MAC')

    // 全部操作均结束，关闭loading并返回
    return true
  }

  /* 登录（管理员）*/
  const LoginIn = async (loginInfo) => {
    try {
      loadingInstance.value = ElLoading.service({
        fullscreen: true,
        text: '登录中，请稍候...'
      })

      const res = await login(loginInfo)

      if (res.code !== 0) {
        return false
      }
      return await afterLogin(res)
    } catch (error) {
      console.error('LoginIn error:', error)
      return false
    } finally {
      loadingInstance.value?.close()
    }
  }

  /* 登录（家长 / 商家）：用 **App 的账号**，不是后台用户名
   *
   * 闸门在服务端：`merchant_application` 里这个角色必须是 approved，
   * 没申请 / 待审 / 被驳回都会在登录这一步被拒（错误文案由服务端给，
   * 请求拦截器已经弹过一次，这里不用再弹）。
   */
  const PortalLoginIn = async (loginInfo) => {
    try {
      loadingInstance.value = ElLoading.service({
        fullscreen: true,
        text: '登录中，请稍候...'
      })

      const res = await appLogin(loginInfo)

      if (res.code !== 0) {
        return false
      }
      return await afterLogin(res)
    } catch (error) {
      console.error('PortalLoginIn error:', error)
      return false
    } finally {
      loadingInstance.value?.close()
    }
  }
  /* 登出*/
  const LoginOut = async () => {
    const res = await jsonInBlacklist()

    // 登出失败
    if (res.code !== 0) {
      return
    }

    await ClearStorage()

    // 把路由定向到登录页，无需等待直接reload
    router.push({ name: 'Login', replace: true })
    window.location.reload()
  }
  /* 清理数据 */
  const ClearStorage = async () => {
    token.value = ''
    // 使用remove方法正确删除cookie
    xToken.remove()
    sessionStorage.clear()
    // 清理所有相关的localStorage项
    localStorage.removeItem('originSetting')
    localStorage.removeItem('token')
  }

  return {
    userInfo,
    token: currentToken,
    NeedInit,
    ResetUserInfo,
    GetUserInfo,
    LoginIn,
    PortalLoginIn,
    LoginOut,
    setToken,
    loadingInstance,
    ClearStorage
  }
})
