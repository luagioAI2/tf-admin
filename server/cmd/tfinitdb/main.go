// tfinitdb 给共享 SQLite 补建 gin-vue-admin 的 sys_* 表与初始数据。
//
// # 为什么需要它
//
// `/init/initdb` 这个 HTTP 接口在 `global.GVA_DB != nil` 时直接返回
// 「已存在数据库配置」。而 tf-admin 与 tf-platform-api **共用同一个 tf.db**，
// 平台早就把库建好了 ⇒ 那个接口永远不会执行，后台永远没有 sys_* 表。
// 这里绕过 HTTP 层的守卫，直接调 service 层的 `InitDB`（它本身没有这个判断）。
//
// # 安全性
//
//   - `bizModel()` 是空实现（`db.AutoMigrate()` 不接模型），`RegisterTables` 也只列
//     `sys_*` / `exa_*`，所以平台那 30 张业务表不会被 AutoMigrate 碰到。
//   - `SqliteEmptyDsn()` 只是拼出文件路径去 open **已有**文件，不会清空。
//   - **幂等**：`sys_users` 已存在就直接退出，什么都不做。
//     这条守卫很重要 —— `WriteConfig` 会重新生成 `jwt.signing-key`，
//     重复跑会把现有登录态全部作废。
//
// # 用法
//
//	cd tf-admin/server
//	TF_ADMIN_PASSWORD='...' go run ./cmd/tfinitdb
//
// 密码不设则用 `TfAdmin@2026`。库路径 / 库名从 config.yaml 的 sqlite 段读。
package main

import (
	"fmt"
	"os"

	"github.com/flipped-aurora/gin-vue-admin/server/core"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	_ "github.com/flipped-aurora/gin-vue-admin/server/initialize" // 触发 source/* 的 RegisterInit
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	sysService "github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/glebarez/sqlite" // 已经是直接依赖，不再为探表多引一个包
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultAdminPassword = "TfAdmin@2026"

func main() {
	global.GVA_VP = core.Viper()
	global.GVA_LOG = core.Zap()
	zap.ReplaceGlobals(global.GVA_LOG)

	conf := request.InitDB{
		AdminPassword: envOr("TF_ADMIN_PASSWORD", defaultAdminPassword),
		DBType:        "sqlite",
		DBName:        global.GVA_CONFIG.Sqlite.Dbname,
		DBPath:        global.GVA_CONFIG.Sqlite.Path,
	}
	if conf.DBName == "" || conf.DBPath == "" {
		fmt.Println("config.yaml 的 sqlite 段没配 path / db-name，先补上再跑")
		os.Exit(1)
	}

	// 幂等守卫：已经初始化过就什么都不做。
	// 必须放在 InitDB 之前 —— 它末尾的 WriteConfig 会重新生成 jwt.signing-key。
	if hasTable(conf.DBPath+"/"+conf.DBName+".db", "sys_users") {
		fmt.Println("sys_users 已存在 —— 这个库已经初始化过，什么都没做。")
		return
	}

	fmt.Printf("开始初始化：%s/%s.db\n", conf.DBPath, conf.DBName)
	if err := (&sysService.InitDBService{}).InitDB(conf); err != nil {
		panic(fmt.Errorf("initdb failed: %w", err))
	}
	fmt.Println("\n=== INITDB OK ===")
	fmt.Printf("管理员密码：%s\n", conf.AdminPassword)
}

// hasTable 直接开文件探表。这里刻意**不调 initialize.Gorm()** ——
// 那条路会跑 AutoMigrate，而本程序要判的正是「该不该 migrate」。
func hasTable(dsn, table string) bool {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return false
	}
	defer func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	}()
	return db.Migrator().HasTable(table)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
