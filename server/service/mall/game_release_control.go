package mall

// 版本状态变更（删除 / 上架 / 下架）的**转发**。
//
// ## 为什么不能直接写库
//
// 后台和平台共用同一个 `tf.db`，所以后台一句 `GVA_DB.Delete(...)` 就能把行改掉 ——
// 但平台那边**目录缓存还在内存里**（`CatalogCacheSeconds`，默认 300 秒）。
//
// 结果：运营点了「下架」，客户端最长还能再装 5 分钟；如果包目录也被清了，
// 用户拿到的是 404。「下架」这个动作到底算不算数，取决于平台有没有机会失效缓存。
//
// 所以这三个动作和上传发版走同一条路：**转发给平台**，由平台改库 + 失效缓存。
//
// ## 转发失败就直接失败
//
// 不做「平台连不上就自己写库」的降级 —— 那样会得到「库里下架了、目录里还在」
// 的坏状态，而且只有等用户投诉才会发现。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// releaseControlTimeout 状态变更的转发超时。
// 这类请求只改一行 + 失效缓存，秒级就回；给 30 秒是留够平台重启时的重试窗口。
const releaseControlTimeout = 30 * time.Second

// forwardToPlatform 把一次 JSON 调用转给平台管理端，返回平台的业务码与提示。
//
// 业务码非 200 属于**预期内的失败**（比如版本不存在），不算 error；
// 只有「连不上 / 回包不是 JSON」才是 error。
func (s *GameReleaseService) forwardToPlatform(
	ctx context.Context,
	path string,
	payload any,
) (code int, msg string, err error) {
	cfg := global.GVA_CONFIG.TFAPI
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return 0, "", fmt.Errorf("没有配置平台地址（config.yaml 的 tf-api.base-url），这个操作不可用")
	}
	if cfg.AdminSignKey == "" {
		return 0, "", fmt.Errorf("没有配置 tf-api.admin-sign-key，签不出平台令牌")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}

	cctx, cancel := context.WithTimeout(ctx, releaseControlTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(cctx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Token", mintAdminToken(cfg.AdminSignKey, cfg.AdminTokenTTL))

	res, err := (&http.Client{Timeout: releaseControlTimeout}).Do(httpReq)
	if err != nil {
		return 0, "", fmt.Errorf("连不上平台（%s）：%w", base, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return 0, "", err
	}

	// 平台统一回包 { code, error, data } —— 失败文案在 `error`，不是 `msg`
	var out struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
		Msg   string `json:"msg"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, "", fmt.Errorf("平台回了非 JSON（HTTP %d）：%s", res.StatusCode, head(raw, 120))
	}
	text := out.Error
	if text == "" {
		text = out.Msg
	}
	return out.Code, text, nil
}
