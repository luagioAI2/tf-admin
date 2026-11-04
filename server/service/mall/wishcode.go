package mall

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
)

/* ============================================================
   激活码（后台）
   ------------------------------------------------------------
   家长在线下买了一张卡，孩子把卡上的 6 位码敲进 App，那款游戏就进他的库。
   核销在 C 端（`tf-platform-api` 的 `/api/wish/redeem-code`），
   这一页只负责「生成 / 看 / 作废」。

   ## 三个必须和 C 端对齐的口径

   1. **字母表**：Crockford Base32，32 个字符，去掉 `I L O U`。
      前三个和 `1` `0` 在手写体里分不清；`U` 是 Crockford 的老规矩。
   2. **长度 6**：手输的，短一点少错位。
   3. **哈希**：`sha256("tf-wish:" + 归一化后的码)`。
      ★ 归一化必须和 C 端 `domain/gameCode.ts` 的 `normalizeGameCode`
      以及服务端 `hashWishCode` **逐条一致** —— 三处任何一处不一样，
      症状都是「后台生成的码，App 说没有这个码」，而且两边都不报错。

   ## 明文只在这里出现一次

   表里只有哈希。生成接口把明文返回给运营去印卡片，**之后再也拿不回来**。
   所以界面上必须让运营当场复制走，并且明确说这件事。
   ============================================================ */

/** Crockford Base32：32 个字符，去掉了 I L O U */
const wishCodeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

/** 码长（不含分隔符） */
const wishCodeLen = 6

/** 哈希前缀。**必须和 `tf-platform-api` 的 `hashWishCode` 一致** */
const wishCodeHashSalt = "tf-wish:"

/** 单次最多生成多少个 */
const wishCodeMaxBatch = 500

// mintWishCode 生成一个码。
//
// 用 `crypto/rand` 而不是 `math/rand` —— 后者可预测，而码就是「谁拿到谁能用」。
// 256 % 32 == 0，所以 `% 32` 不引入偏差。
func mintWishCode() (string, error) {
	buf := make([]byte, wishCodeLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, x := range buf {
		b.WriteByte(wishCodeAlphabet[int(x)%len(wishCodeAlphabet)])
	}
	return b.String(), nil
}

// hashWishCode 与 C 端同口径的哈希。
//
// ⚠️ 归一化规则（和 `tf-platform-api/internal/app/service/wish_srv.go` 的
// `normalizeWishCode` 逐条对应）：大写、去掉空白与 `- _ · ・`、
// 把 `I` `L` 纠正成 `1`、`O` 纠正成 `0`。
func hashWishCode(code string) string {
	normalized := normalizeWishCode(code)
	sum := sha256.Sum256([]byte(wishCodeHashSalt + normalized))
	return hex.EncodeToString(sum[:])
}

func normalizeWishCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(code) {
		switch r {
		case ' ', '\t', '\n', '\r', '-', '_', '·', '・':
			continue
		case 'I', 'L':
			b.WriteRune('1')
		case 'O':
			b.WriteRune('0')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// codeTail 后 4 位，印在列表上让家长核对「是不是这张卡」。
//
// 取**明文**的后 4 位（不是哈希的）—— 家长手里拿的是明文，
// 给他看哈希的一截等于什么都没说。
func codeTail(code string) string {
	if len(code) <= 4 {
		return code
	}
	return code[len(code)-4:]
}

// WishCodeService 激活码
type WishCodeService struct{}

// CodeList 激活码列表
func (s *WishCodeService) CodeList(ctx context.Context, info request.WishCodeSearch) (list []mall.WishCode, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.WishCode{})
	if info.ItemID != "" {
		db = db.Where("item_id = ?", info.ItemID)
	}
	if info.Batch != "" {
		db = db.Where("batch = ?", info.Batch)
	}
	if info.Used != nil {
		db = db.Where("used = ?", *info.Used)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	// 新的在前：刚生成的一批要能立刻看到
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// GenerateCodes 批量生成激活码，返回**明文**（只此一次）。
//
// ## 为什么要先查商品
//
// 码是「解锁某件商品」的：核销时 C 端按 `item_id` 找到商品、再取 `game_id`。
// 如果挂在一件没有 `game_id` 的商品上（实物、或者 kind 写错的盲盒），
// 核销会**成功**（码被标记已用）却什么都不解锁 —— 那张卡就白买了。
// 所以在这里就拦住。
func (s *WishCodeService) GenerateCodes(ctx context.Context, req request.GenerateWishCodeReq) ([]string, error) {
	if req.Count <= 0 {
		return nil, fmt.Errorf("数量要大于 0")
	}
	if req.Count > wishCodeMaxBatch {
		return nil, fmt.Errorf("一次最多生成 %d 个", wishCodeMaxBatch)
	}

	var item mall.WishItem
	if err := global.GVA_DB.WithContext(ctx).
		Where("item_id = ?", req.ItemID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("商品不存在：%s", req.ItemID)
	}
	if strings.TrimSpace(item.GameID) == "" {
		return nil, fmt.Errorf("「%s」没有绑定具体游戏，码核销后解锁不出东西；请先把商品分类改成「单款游戏」并选好那一款", item.Name)
	}

	batch := strings.TrimSpace(req.Batch)
	if batch == "" {
		batch = "B" + time.Now().Format("20060102-1504")
	}
	expireAt := int64(0)
	if req.ExpireDays > 0 {
		expireAt = time.Now().AddDate(0, 0, req.ExpireDays).Unix()
	}

	now := nowUnix()
	plain := make([]string, 0, req.Count)
	rows := make([]mall.WishCode, 0, req.Count)
	seen := map[string]struct{}{}

	for len(plain) < req.Count {
		code, err := mintWishCode()
		if err != nil {
			return nil, err
		}
		h := hashWishCode(code)
		if _, ok := seen[h]; ok {
			continue /* 本批内撞了（概率极低），重摇 */
		}
		/* 和库里已有的撞：撞的概率是 10 亿分之一，但撞上就是「两张卡一个码」，
		   第二个人输进去会看到「这个码已经用过了」—— 所以还是查一次。
		   批量插入前一次查出可能冲突的集合更省，但 500 次单查在后台页面上
		   也就几十毫秒，不值得为它加复杂度。 */
		var dup int64
		if err := global.GVA_DB.WithContext(ctx).Model(&mall.WishCode{}).
			Where("code_hash = ?", h).Count(&dup).Error; err != nil {
			return nil, err
		}
		if dup > 0 {
			continue
		}
		seen[h] = struct{}{}
		plain = append(plain, code)
		rows = append(rows, mall.WishCode{
			BaseModel: mall.BaseModel{CreatedAt: now, UpdatedAt: now},
			CodeHash:  h,
			CodeTail:  codeTail(code),
			ItemID:    req.ItemID,
			Batch:     batch,
			Used:      false,
			ExpireAt:  expireAt,
		})
	}

	if err := global.GVA_DB.WithContext(ctx).CreateInBatches(&rows, 100).Error; err != nil {
		return nil, err
	}
	return plain, nil
}

// VoidCode 作废。
//
// ★ 只能作废**没用过**的：用过的那个孩子的游戏已经到账了，
// 删掉这一行只是把「谁用的」这条记录抹掉，游戏并不会收回 —— 那是最糟的组合
// （既收不回东西，又丢了凭据）。
//
// 给 `batch` 时作废整批里没用过的；给 `id` 时只作废那一张。
func (s *WishCodeService) VoidCode(ctx context.Context, req request.VoidWishCodeReq) (int64, error) {
	if req.ID == 0 && strings.TrimSpace(req.Batch) == "" {
		return 0, fmt.Errorf("要么给 id，要么给批次号")
	}
	db := global.GVA_DB.WithContext(ctx).Where("used = ?", false)
	if req.ID != 0 {
		db = db.Where("id = ?", req.ID)
	} else {
		db = db.Where("batch = ?", strings.TrimSpace(req.Batch))
	}

	/* 先数一下有没有「用过的」被挡住，好把话说清楚 —— 不然运营点完
	   发现数量对不上，会以为作废失败了。 */
	var usedCount int64
	q := global.GVA_DB.WithContext(ctx).Model(&mall.WishCode{}).Where("used = ?", true)
	if req.ID != 0 {
		q = q.Where("id = ?", req.ID)
	} else {
		q = q.Where("batch = ?", strings.TrimSpace(req.Batch))
	}
	if err := q.Count(&usedCount).Error; err != nil {
		return 0, err
	}
	if req.ID != 0 && usedCount > 0 {
		return 0, fmt.Errorf("这张码已经被用过了，不能作废（那个孩子的游戏已经到账）")
	}

	res := db.Delete(&mall.WishCode{})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 && usedCount == 0 {
		return 0, fmt.Errorf("没有找到可作废的码")
	}
	return res.RowsAffected, nil
}

// CodeBatches 已有的批次号（给筛选下拉用）
func (s *WishCodeService) CodeBatches(ctx context.Context) ([]string, error) {
	var out []string
	err := global.GVA_DB.WithContext(ctx).Model(&mall.WishCode{}).
		Where("batch <> ?", "").
		Distinct("batch").Order("batch DESC").Pluck("batch", &out).Error
	return out, err
}
