/*
 * 小任务农场 · 后台前端初始化
 *
 * */
// 加载网站配置文件夹
import { register } from './global'
import packageInfo from '../../package.json'

export default {
  install: (app) => {
    register(app)
    console.log(`
       小任务农场 · 运营后台
       当前版本:v${packageInfo.version}
       家长 / 商家用 App 的手机号和密码登录，看到什么由权限决定
    `)
  }
}
