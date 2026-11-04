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
    `)
  }
}
