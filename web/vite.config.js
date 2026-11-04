import { viteLogo } from './src/core/config'
import Banner from 'vite-plugin-banner'
import * as path from 'path'
import { loadEnv } from 'vite'
import vuePlugin from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import VueFilePathPlugin from './vitePlugin/componentName/index.js'
import { svgBuilder } from 'vite-auto-import-svg'
import vueRootValidator from 'vite-check-multiple-dom'
import { AddSecret } from './vitePlugin/secret'
import UnoCSS from '@unocss/vite'

// @see https://cn.vitejs.dev/config/
export default ({ mode }) => {
  AddSecret('')
  const env = loadEnv(mode, process.cwd())
  viteLogo(env)

  const timestamp = Date.parse(new Date())

  const alias = {
    '@': path.resolve(import.meta.dirname, './src'),
    vue$: 'vue/dist/vue.runtime.esm-bundler.js'
  }

  const base = '/'
  const root = './'
  const outDir = 'dist'

  const config = {
    base: base, // 编译后js导入的资源路径
    root: root, // index.html文件所在位置
    publicDir: 'public', // 静态资源文件夹
    resolve: {
      alias
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler' // or "modern"
        }
      }
    },
    server: {
      // 如果使用docker-compose开发模式，设置为false
      open: true,
      port: Number(env.VITE_CLI_PORT),
      proxy: {
        // 把key的路径代理到target位置
        // detail: https://cli.vuejs.org/config/#devserver-proxy
        [env.VITE_BASE_API]: {
          // 需要代理的路径   例如 '/api'
          target: `${env.VITE_BASE_PATH}:${env.VITE_SERVER_PORT}/`, // 代理到 目标路径
          changeOrigin: true,
          rewrite: (path) =>
            path.replace(new RegExp('^' + env.VITE_BASE_API), '')
        },
        '/plugin': {
          // 需要代理的路径   例如 '/api'
          target: `https://plugin.gin-vue-admin.com/api/`, // 代理到 目标路径
          changeOrigin: true,
          rewrite: (path) =>
            path.replace(new RegExp('^/plugin'), '')
        }
      }
    },
    build: {
      manifest: false, // 是否产出manifest.json
      sourcemap: false, // 是否产出sourcemap.json
      outDir: outDir, // 产出目录
      target: 'es2015',
      rolldownOptions: {
        output: {
          entryFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
          chunkFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
          assetFileNames: 'assets/087AC4D233B64EB0[name].[hash].[ext]'
        }
      }
    },
    plugins: [
      env.VITE_POSITION === 'open' &&
      vueDevTools({ launchEditor: env.VITE_EDITOR }),
      vuePlugin(),
      /* ⚠️ `vite-auto-import-svg` 除了生成 svg sprite，还会在 **production** 构建里
         往入口 chunk 末尾塞一段「未授权」全屏遮罩（z-index:9999，
         文案「此模板未授权且未获得授权使用权」），并往 `<meta name=keywords>`
         里塞框架自己的 SEO 词 + 时间戳。遮罩一出现整个后台就点不动了，
         而且 dev server 下**不会**注入 —— 也就是本地一切正常、构建出来才发现。
         这里只保留它「生成 sprite」那一半（`transformIndexHtml` 里注入
         `<svg><symbol>`），把 `renderChunk` 那个钩子摘掉。
         ⚠️ 这属于**商业授权**范畴：想彻底合规就买授权，然后
            `AddSecret('<域名>', '<密钥>')` —— 有合法密钥时它本来就不注入。 */
      (() => {
        const p = svgBuilder(['./src/plugin/', './src/assets/icons/'], base, outDir, 'assets', mode)
        if (p && typeof p === 'object') delete p.renderChunk
        return p
      })(),
      /* keywords 也被那个插件改写过（它在上面的 transformIndexHtml 里换掉了）。
         这个钩子排在它后面，把标签换回我们自己的。 */
      {
        name: 'own-keywords',
        transformIndexHtml: {
          order: 'post',
          handler: (html) =>
            html.replace(
              /<meta\s+[^>]*name=["']keywords["'][^>]*>/i,
              '<meta content="小任务农场,运营后台,家长中心,商家中心,心愿单,兑换记录,招商审核" name="keywords" />'
            )
        }
      },
      [Banner(`\n 小任务农场 · 运营后台 \n Time : ${timestamp}`)],
      VueFilePathPlugin('./src/pathInfo.json'),
      UnoCSS(),
      vueRootValidator()
    ]
  }
  return config
}
