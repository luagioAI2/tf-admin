package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// ============================ 登录桥 ============================

// AppLoginReq 家长 / 商家用 **App 账号** 登录后台。
//
// ⚠️ 这里没有 `username`：App 账号的身份就是手机号，后台的 `sys_user`
// 是登录时**惰性生成**的影子行（`app_{uid}`），不是给人记的账号。
// 让家长多记一个「后台用户名」等于多一个会忘、会丢的东西。
//
// ⚠️ 这里也**没有 `role`**：登录页只有一个输入框，不让人自己选身份。
// 「你是家长还是商家」这件事，`merchant_application` 里已经写着（审核通过的那几条），
// 让用户再选一次只会选错 —— 而且选错的表现是「菜单少了一半」，
// 用户会以为自己的申请没通过。角色由服务端按审核记录取并集。
type AppLoginReq struct {
	Mobile   string `json:"mobile" binding:"required"`   // 手机号（带不带 +86 都行）
	Password string `json:"password" binding:"required"` // App 的登录密码
}

// ============================ 家长端 ============================

// ParentWishSearch 家长端心愿单列表
type ParentWishSearch struct {
	request.PageInfo
	// Status 只看某一档（wished / committed / done / cancelled），空 = 全部
	Status string `json:"status" form:"status"`
}

// ParentRecordSearch 家长端兑换记录
type ParentRecordSearch struct {
	request.PageInfo
}

// ============================ 商家端 ============================

// MerchantItemSearch 商家端「我的商品」
type MerchantItemSearch struct {
	request.PageInfo
	// Archived nil = 全部；false = 在架；true = 已下架
	Archived *bool `json:"archived"`
}

// MerchantSaveItemReq 商家上架 / 改价 / 下架自己的一件商品。
//
// ⚠️ **故意只有这几个字段**。商家**不能**改：
//   - `kind`（把实物改成盲盒 ⇒ 扣的是券、发的是游戏）
//   - `game_id` / `pool` / `unlock_count`（那是盲盒的池子）
//   - `merchant_uid`（改归属 = 把别人的东西搬到自己名下）
//
// 所以不复用运营用的 `UpdateWishItemReq` —— 那份是全字段的，
// 少写一行判断就等于把上面三个也开给商家了。
//
// ⚠️ **整条覆盖**：没传的字段会被写成零值（不是「保持不变」）。
// 因为请求体里区分不出「没传」和「传了空串」——`BuyURL: ""` 既可能是
// 「清空链接」也可能是「这次没改」。商家那边的编辑弹窗**每次都提交全部字段**，
// 所以覆盖语义是对的；新加调用方时必须整份提交，否则会静默抹掉数据。
type MerchantSaveItemReq struct {
	// ItemID 留空 = 新建；填了 = 改这一件（必须是自己名下的）
	ItemID      string `json:"itemId"`
	Name        string `json:"name" binding:"required"`
	Emoji       string `json:"emoji"`
	Tint        string `json:"tint"`
	PartnerName string `json:"partnerName"`
	PriceCents  int64  `json:"priceCents"`
	Note        string `json:"note"`
	Notice      string `json:"notice"`
	BuyURL      string `json:"buyUrl"`
	BuyLabel    string `json:"buyLabel"`
	SortOrder   int    `json:"sortOrder"`
	// Archived 上架 / 下架（nil = 不改）
	Archived *bool `json:"archived"`
}
