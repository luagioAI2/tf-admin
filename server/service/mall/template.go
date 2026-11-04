package mall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================ 模板 CRUD ============================
//
// 后台和 tf-platform-api **共用同一张 `template` / `template_item`**，
// 没有「后台表 + 同步」这一层（见 model/mall/models.go 的包注释）。
//
// ## 数据范围（用户选的口径：商家自己配自己的）
//
//   - 超管 888：能看 / 改 / 建**全部**模板（官方 + 任意商家）；
//   - 商家 1002 / 1003：只能看 / 改 / 建 `scope=merchant AND owner_uid=自己`；
//   - 家长 1001：接口层就不给权限，服务层也不放行。
//
// ★ casbin 只回答「这个角色能不能调这个地址」，**不回答「能看谁的模板」**。
//   所以每个方法都要先过 `loadOwned`，SQL 里必须带 scope / owner_uid。
//   少这一层，商家 A 改一下请求体里的 id 就能改商家 B 的模板，而界面完全正常。

const (
	TemplateKindTask   = "task"
	TemplateKindRedeem = "redeem"

	TemplateScopeOfficial = "official"
	TemplateScopeMerchant = "merchant"
)

// TemplateActor 调用者的数据范围。**由 API 层从 token 推导，不接受前端传参**。
type TemplateActor struct {
	IsAdmin  bool
	OwnerUID uint // 商家账号的 App uid（非管理员必 > 0）
}

// scopeWhere 给「读列表」套数据范围。
func (a TemplateActor) scopeWhere(db *gorm.DB) *gorm.DB {
	if a.IsAdmin {
		return db
	}
	return db.Where("scope = ? AND owner_uid = ?", TemplateScopeMerchant, a.OwnerUID)
}

// canOwn 这份模板是不是这个调用者能改的。
func (a TemplateActor) canOwn(t *mall.Template) bool {
	if a.IsAdmin {
		return true
	}
	return t.Scope == TemplateScopeMerchant && t.OwnerUID == a.OwnerUID
}

// loadOwned 按 id 读一份模板，并**立刻**校验数据范围。
//
// 不存在 / 不是自己的返回**同一个错误**：区分开等于给对方一个
// 「这份模板存在但不是你的」的探测接口。
func loadOwned(ctx context.Context, a TemplateActor, id uint) (*mall.Template, error) {
	var t mall.Template
	err := global.GVA_DB.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("模板不存在或不在你名下")
	}
	if err != nil {
		return nil, err
	}
	if !a.canOwn(&t) {
		return nil, errors.New("模板不存在或不在你名下")
	}
	return &t, nil
}

type TemplateService struct{}

// List 模板列表（带条目数）
func (s *TemplateService) List(ctx context.Context, a TemplateActor, info mallReq.TemplateSearch) (list []mall.TemplateRow, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.Template{})
	db = a.scopeWhere(db)
	if info.Kind != "" {
		db = db.Where("kind = ?", info.Kind)
	}
	if info.Scope != "" && a.IsAdmin {
		db = db.Where("scope = ?", info.Scope)
	}
	if info.OwnerUID != nil && a.IsAdmin {
		db = db.Where("owner_uid = ?", *info.OwnerUID)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	var heads []mall.Template
	if err = db.Order("sort ASC, id DESC").
		Scopes(paginate(info.Page, info.PageSize)).Find(&heads).Error; err != nil {
		return
	}
	list = make([]mall.TemplateRow, 0, len(heads))
	if len(heads) == 0 {
		return
	}
	ids := make([]uint, 0, len(heads))
	for _, h := range heads {
		ids = append(ids, h.ID)
	}
	// 一次 GROUP BY 查完这一页的条目数，别在循环里 count（N+1）。
	type cnt struct {
		TemplateID uint  `gorm:"column:template_id"`
		N          int64 `gorm:"column:n"`
	}
	var counts []cnt
	if err = global.GVA_DB.WithContext(ctx).Table("template_item").
		Select("template_id, count(*) AS n").
		Where("template_id IN ? AND deleted_at IS NULL", ids).
		Group("template_id").Scan(&counts).Error; err != nil {
		return
	}
	byID := make(map[uint]int64, len(counts))
	for _, c := range counts {
		byID[c.TemplateID] = c.N
	}
	for _, h := range heads {
		list = append(list, mall.TemplateRow{Template: h, ItemCount: byID[h.ID]})
	}
	return
}

// Detail 模板详情（头 + 全部条目）
func (s *TemplateService) Detail(ctx context.Context, a TemplateActor, id uint) (*mall.TemplateDetail, error) {
	t, err := loadOwned(ctx, a, id)
	if err != nil {
		return nil, err
	}
	var items []mall.TemplateItem
	if err = global.GVA_DB.WithContext(ctx).
		Where("template_id = ?", id).
		Order("sort ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &mall.TemplateDetail{Template: *t, Items: items}, nil
}

// normalizeScope 把前端传的 scope / owner_uid 收敛成合法且**在权限内**的一组值。
//
// 商家账号一律强制 merchant + 自己，前端传什么都不算数 —— 这是
// 「商家只能配自己的模板」这条口径唯一真正落地的地方。
func normalizeScope(a TemplateActor, req *mallReq.TemplateCreateReq) error {
	if !a.IsAdmin {
		req.Scope = TemplateScopeMerchant
		req.OwnerUID = a.OwnerUID
	}
	if req.Scope == "" {
		req.Scope = TemplateScopeOfficial
	}
	switch req.Scope {
	case TemplateScopeOfficial:
		if req.OwnerUID != 0 {
			return errors.New("官方模板的归属必须是平台（owner_uid = 0）")
		}
	case TemplateScopeMerchant:
		if req.OwnerUID == 0 {
			return errors.New("商家模板必须指定归属商家")
		}
	default:
		return errors.New("模板范围只能是 official / merchant")
	}
	return nil
}

// merchantApproved 这个 uid 是不是已通过审核的商家。
//
// 后台给某个商家预置模板时，挂在一个「还没通过 / 已被驳回」的 uid 上
// 等于把内容挂在一个永远进不来的入口上，而且列表看上去完全正常。
func merchantApproved(ctx context.Context, uid uint) (bool, error) {
	var n int64
	err := global.GVA_DB.WithContext(ctx).Model(&mall.MerchantApplication{}).
		Where("uid = ? AND role = ? AND status = ?", uid, "merchant", "approved").
		Count(&n).Error
	return n > 0, err
}

// Create 新建模板头（条目随后在 item/save 里加）
func (s *TemplateService) Create(ctx context.Context, a TemplateActor, req mallReq.TemplateCreateReq) (uint, error) {
	if strings.TrimSpace(req.Name) == "" {
		return 0, errors.New("模板名不能为空")
	}
	if req.Kind != TemplateKindTask && req.Kind != TemplateKindRedeem {
		return 0, errors.New("模板种类只能是 task / redeem")
	}
	if err := normalizeScope(a, &req); err != nil {
		return 0, err
	}
	if req.Scope == TemplateScopeMerchant {
		ok, err := merchantApproved(ctx, req.OwnerUID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, errors.New("这个商家还没通过审核，不能给它配模板")
		}
	}

	row := mall.Template{
		Name:      strings.TrimSpace(req.Name),
		Kind:      req.Kind,
		Scope:     req.Scope,
		OwnerUID:  req.OwnerUID,
		IsDefault: false,
		Note:      req.Note,
		Sort:      req.Sort,
		Source:    "manual",
	}
	now := nowUnix()
	row.CreatedAt = now
	row.UpdatedAt = now

	// 「新建并设为默认」和「清掉同组旧默认」必须原子：
	// 拆开会有一瞬间同组两个 is_default=1，平台按 id DESC 取，谁生效看运气。
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if req.IsDefault {
			return setDefaultInGroup(tx, row.Kind, row.Scope, row.OwnerUID, row.ID)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// setDefaultInGroup 把同组其它默认清掉，再把这个设为默认。
//
// 分组是 (kind, scope, owner_uid)：官方任务和官方兑换互不影响，
// 商家 A 的默认也不该清掉商家 B 的。
func setDefaultInGroup(tx *gorm.DB, kind, scope string, ownerUID, id uint) error {
	now := nowUnix()
	if err := tx.Model(&mall.Template{}).
		Where("kind = ? AND scope = ? AND owner_uid = ? AND id != ? AND is_default = ?",
			kind, scope, ownerUID, id, true).
		Updates(map[string]interface{}{"is_default": false, "updated_at": now}).Error; err != nil {
		return err
	}
	return tx.Model(&mall.Template{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_default": true, "updated_at": now}).Error
}

// Update 改模板头（名字 / 备注 / 排序）
func (s *TemplateService) Update(ctx context.Context, a TemplateActor, req mallReq.TemplateUpdateReq) error {
	t, err := loadOwned(ctx, a, req.ID)
	if err != nil {
		return err
	}
	fields := map[string]interface{}{"updated_at": nowUnix()}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return errors.New("模板名不能为空")
		}
		fields["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Note != nil {
		fields["note"] = *req.Note
	}
	if req.Sort != nil {
		fields["sort"] = *req.Sort
	}
	return global.GVA_DB.WithContext(ctx).Model(&mall.Template{}).
		Where("id = ?", t.ID).Updates(fields).Error
}

// SetDefault 设为该分组的默认模板
func (s *TemplateService) SetDefault(ctx context.Context, a TemplateActor, id uint) error {
	t, err := loadOwned(ctx, a, id)
	if err != nil {
		return err
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return setDefaultInGroup(tx, t.Kind, t.Scope, t.OwnerUID, t.ID)
	})
}

// Delete 删除模板（连带条目**硬删**）
//
// ⚠️ 用 Unscoped 硬删，不用软删：`template_item` 上有
// `uk_tpl_item(template_id, item_id)` 唯一索引，软删的行还占着名字，
// 运营「删掉再加回来」会撞索引而**写不进去、界面只说保存失败**。
func (s *TemplateService) Delete(ctx context.Context, a TemplateActor, id uint) error {
	t, err := loadOwned(ctx, a, id)
	if err != nil {
		return err
	}
	// 官方默认模板是 App 首启的兜底，删掉就没有默认了 ——
	// 平台 Resolve 会报 OperationFailed，所有新用户都铺不出任务。
	// 除非同 kind 还有另一份官方模板，否则不许删。
	if t.Scope == TemplateScopeOfficial && t.IsDefault {
		var others int64
		if err := global.GVA_DB.WithContext(ctx).Model(&mall.Template{}).
			Where("kind = ? AND scope = ? AND owner_uid = 0 AND id != ?",
				t.Kind, TemplateScopeOfficial, t.ID).Count(&others).Error; err != nil {
			return err
		}
		if others == 0 {
			return errors.New("这是该种类唯一的官方默认模板，删了新用户就没有默认内容了；先把另一份设为默认")
		}
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("template_id = ?", t.ID).
			Delete(&mall.TemplateItem{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("id = ?", t.ID).Delete(&mall.Template{}).Error
	})
}

// typedExtraKeys 下发时由 typed 列提供的键。Extra 里不许再出现它们，
// 否则 `template_srv.go` 的合并会把运营刚改的 price / base_points 盖回旧值。
var typedExtraKeys = map[string]bool{
	"item_id": true, "name": true, "emoji": true, "category": true, "cycle": true,
	"planned_minutes": true, "base_points": true, "price": true, "note": true,
	"limit_per_day": true, "stock": true, "once": true, "sort": true,
}

// cleanExtra 校验并清理 Extra：必须是 JSON 对象，且剔掉 typed 列的键。
func cleanExtra(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", errors.New("玩法参数必须是 JSON 对象")
	}
	for k := range typedExtraKeys {
		delete(m, k)
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// SaveItem 新增 / 改一条模板条目
func (s *TemplateService) SaveItem(ctx context.Context, a TemplateActor, req mallReq.TemplateItemSaveReq) (uint, error) {
	t, err := loadOwned(ctx, a, req.TemplateID)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return 0, errors.New("条目名不能为空")
	}
	if req.Price < 0 || req.BasePoints < 0 || req.PlannedMinutes < 0 || req.LimitPerDay < 0 {
		return 0, errors.New("积分 / 价格 / 时长 / 限购不能是负数")
	}
	if req.Stock < -1 {
		return 0, errors.New("库存只能是 -1（不限）或 >= 0")
	}
	extra, err := cleanExtra(req.Extra)
	if err != nil {
		return 0, err
	}

	now := nowUnix()
	fields := map[string]interface{}{
		"name":            strings.TrimSpace(req.Name),
		"emoji":           req.Emoji,
		"category":        req.Category,
		"cycle":           req.Cycle,
		"planned_minutes": req.PlannedMinutes,
		"base_points":     req.BasePoints,
		"price":           req.Price,
		"note":            req.Note,
		"limit_per_day":   req.LimitPerDay,
		"stock":           req.Stock,
		"once":            req.Once,
		"sort":            req.Sort,
		"extra":           extra,
		"updated_at":      now,
	}

	if req.ID == 0 {
		itemID := strings.TrimSpace(req.ItemID)
		if itemID == "" {
			itemID = "it_" + uuid.NewString()[:8]
		}
		var dup int64
		if err := global.GVA_DB.WithContext(ctx).Model(&mall.TemplateItem{}).
			Where("template_id = ? AND item_id = ?", t.ID, itemID).Count(&dup).Error; err != nil {
			return 0, err
		}
		if dup > 0 {
			return 0, fmt.Errorf("条目 id「%s」已存在，换一个", itemID)
		}
		row := mall.TemplateItem{TemplateID: t.ID, ItemID: itemID, Extra: extra}
		row.Name = strings.TrimSpace(req.Name)
		row.Emoji = req.Emoji
		row.Category = req.Category
		row.Cycle = req.Cycle
		row.PlannedMinutes = req.PlannedMinutes
		row.BasePoints = req.BasePoints
		row.Price = req.Price
		row.Note = req.Note
		row.LimitPerDay = req.LimitPerDay
		row.Stock = req.Stock
		row.Once = req.Once
		row.Sort = req.Sort
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := global.GVA_DB.WithContext(ctx).Create(&row).Error; err != nil {
			return 0, err
		}
		return row.ID, nil
	}

	res := global.GVA_DB.WithContext(ctx).Model(&mall.TemplateItem{}).
		Where("id = ? AND template_id = ?", req.ID, t.ID).Updates(fields)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, errors.New("条目不存在或不属于这份模板")
	}
	return req.ID, nil
}

// DeleteItem 硬删一条模板条目（理由同 Delete）
func (s *TemplateService) DeleteItem(ctx context.Context, a TemplateActor, req mallReq.TemplateItemDeleteReq) error {
	if _, err := loadOwned(ctx, a, req.TemplateID); err != nil {
		return err
	}
	res := global.GVA_DB.WithContext(ctx).Unscoped().
		Where("id = ? AND template_id = ?", req.ID, req.TemplateID).
		Delete(&mall.TemplateItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("条目不存在或不属于这份模板")
	}
	return nil
}

// MakeInviteCode 商家的推广邀请码。
//
// ⚠️ 必须和 tf-platform-api `entities.MakeInviteCode` **逐字节一致**：
// 后台把这个码印在物料上，平台按同一规则校验和归属。改了一边，
// 症状是「码能生成、但绑不上 / 绑到别人身上」，两端都不报错。
// `service/mall/template_test.go` 钉了两个已知值，改公式前先看它。
func MakeInviteCode(uid uint) string {
	if uid == 0 {
		return ""
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("tf-invite-v1:%d", uid)))
	return "TF" + strings.ToUpper(strconv.FormatUint(uint64(uid), 36)) +
		"-" + strings.ToUpper(hex.EncodeToString(h[:2]))
}
