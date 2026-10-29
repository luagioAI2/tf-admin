package mall

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// 与 tf-platform-api 相同的错误语义（后台也用它来判「钱不够」这类业务错误）
var (
	// ErrInsufficientBalance 余额不足
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrVersionConflict 乐观锁冲突（并发写同一钱包）
	ErrVersionConflict = errors.New("wallet version conflict")
)

// ServiceGroup 组合本模块所有 Service。
//
// 新增一个 Service 时：在这里加一行，并在 `service/enter.go` 里
// 把 MallServiceGroup 挂到 ServiceGroupApp。
type ServiceGroup struct {
	GameService
	GameReleaseService
	PlatformUserService
	RedeemService
	WishService
	TaskService
	DashboardService
	MerchantService
}

// txOrDB 取事务句柄（没传事务就用普通 DB）。
//
// ⚠️ 这个 helper 的意义：GORM 里「事务内用外层 DB 句柄」会向连接池
// 申请**第二条连接**（第一条已被事务占用），连接池打满时死锁 ——
// tf-platform-api 在回档逻辑上踩过一次（测试挂死 600 秒）。
// 统一走这个 helper 能避免那种错。
func txOrDB(tx *gorm.DB, db *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return db
}

// nowUnix 当前 Unix 秒。
//
// ⚠️ 平台的 `created_at` / `updated_at` 是 **int64**，
// 而且模型的 `autoCreateTime:false` —— GORM **不会**自动填。
// 所以每次写入都必须**手动**赋这两个字段，
// 否则会写进 0（1970 年），而列表按时间倒序时这些记录全在最后。
func nowUnix() int64 {
	return time.Now().Unix()
}
