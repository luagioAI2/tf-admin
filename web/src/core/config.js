/**
 * 网站配置文件
 */
import packageInfo from '../../package.json'

const greenText = (text) => `\x1b[32m${text}\x1b[0m`

export const config = {
  appName: '小任务农场',
  showViteLogo: true,
  keepAliveTabs: false,
  logs: []
}

export const viteLogo = (env) => {
  if (config.showViteLogo) {
    console.log(greenText(`> 小任务农场 · 运营后台 v${packageInfo.version}`))
    console.log(greenText(`> 后端:http://127.0.0.1:${env.VITE_SERVER_PORT}`))
    console.log(greenText(`> 前端:http://127.0.0.1:${env.VITE_CLI_PORT}`))
  }
}

export default config
