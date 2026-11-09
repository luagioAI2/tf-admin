package mall

// 游戏包的**上传转发**（2026-11-07）。
//
// ## 为什么是「转发」而不是「自己存」
//
// 后台和平台共用同一个库，所以版本记录后台自己写就行。但**包文件**不归后台：
// 平台用 `StaticPath/games/<id>/<ver>/` 托管，再由 `PackageBaseURL`（将来是 CDN）
// 对外给地址。后台一旦也往自己盘上写包，客户端拿到的 `url` 就取决于
// 「哪台机器发的版」—— 换个部署全 404，而且**上线之后才发现**。
//
// 所以这里只做三件事：签一个平台令牌 → 把 multipart 原样转发 → 把结果翻成人话。
// 落盘、算 SHA-256、写版本行、签名目录、失效缓存，全部由平台那一次调用完成。
//
// ## 为什么不复用后台自己的 `Publish`
//
// 那样会变成「后台写版本行 + 平台写文件」，两边都要成功；中间失败就是
// 「库里有一个指向空文件的版本」。一次调用只有一个成功点。
//
// ⚠️ 平台不可用（没配 `tf-api.base-url` / 连不上）时，这个动作**直接失败**，
// 不做任何降级 —— 降级只能是自己写库，那正是上面说的那种坏状态。

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
)

// releaseUploadTimeout 转发超时。
//
// 包最大 64MB（平台侧限制），家用宽带上行 5MB/s 也要十几秒，
// 所以不能沿用 GVA 默认的短超时。
const releaseUploadTimeout = 10 * time.Minute

// UploadAndPublish 把一个包文件转给平台，由平台落盘并发布。
//
// 返回平台回的业务码与提示（成功时 `code == 200`）。
func (s *GameReleaseService) UploadAndPublish(
	ctx context.Context,
	req request.PublishUploadReq,
	fh *multipart.FileHeader,
) (code int, msg string, data map[string]any, err error) {
	cfg := global.GVA_CONFIG.TFAPI
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return 0, "", nil, fmt.Errorf("没有配置平台地址（config.yaml 的 tf-api.base-url），上传功能不可用")
	}
	if cfg.AdminSignKey == "" {
		return 0, "", nil, fmt.Errorf("没有配置 tf-api.admin-sign-key，签不出平台令牌")
	}
	if fh == nil {
		return 0, "", nil, fmt.Errorf("没有收到文件")
	}

	src, err := fh.Open()
	if err != nil {
		return 0, "", nil, err
	}
	defer src.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	// 字段名必须和平台 `GameReleaseAPI.Upload` 的 Form 名一致
	fields := map[string]string{
		"game_id":       req.GameID,
		"version":       req.Version,
		"channel":       req.Channel,
		"asset_base":    req.AssetBase,
		"release_notes": req.ReleaseNotes,
		"min_host_api":  req.MinHostApi,
		"forced":        req.Forced,
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := mw.WriteField(k, v); err != nil {
			return 0, "", nil, err
		}
	}
	part, err := mw.CreateFormFile("file", fh.Filename)
	if err != nil {
		return 0, "", nil, err
	}
	// 流式拷贝：64MB 的包不要先读进内存再写一遍
	if _, err := io.Copy(part, src); err != nil {
		return 0, "", nil, err
	}
	if err := mw.Close(); err != nil {
		return 0, "", nil, err
	}

	cctx, cancel := context.WithTimeout(ctx, releaseUploadTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		cctx, http.MethodPost, base+"/api/admin/game/release/upload", &body,
	)
	if err != nil {
		return 0, "", nil, err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("X-Api-Token", mintAdminToken(cfg.AdminSignKey, cfg.AdminTokenTTL))

	res, err := (&http.Client{Timeout: releaseUploadTimeout}).Do(httpReq)
	if err != nil {
		return 0, "", nil, fmt.Errorf("连不上平台（%s）：%w", base, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return 0, "", nil, err
	}

	// 平台的统一回包：{ code, error, data }
	//
	// ⚠️ 失败时文案字段叫 **`error`**，不是 `msg`（`ginx.Resp` 的 json tag 就是
	//    `error`）。照 `msg` 解析的话成功没事、**失败时文案恒为空** ——
	//    界面上只剩一句「平台拒绝（10080201）：」，运营不知道该改什么。
	var out struct {
		Code  int            `json:"code"`
		Data  map[string]any `json:"data"`
		Msg   string         `json:"msg"`
		Error string         `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		// 不是 JSON ⇒ 多半是被网关/反代拦了。把前 120 字带出来，
		// 否则界面上只有一句「上传失败」，排查时不知道对面是谁。
		return 0, "", nil, fmt.Errorf("平台回了非 JSON（HTTP %d）：%s", res.StatusCode, head(raw, 120))
	}
	text := out.Error
	if text == "" {
		text = out.Msg
	}
	/* 业务码非 200 是**预期内**的失败（版本重复 / 包不合法），不当 error 返回 ——
	   调用方要把这句话原样显示给运营。 */
	return out.Code, text, out.Data, nil
}

// mintAdminToken 与平台 `middleware.TokenManager.GenerateToken` **逐字节一致**：
//
//	base64url( "<ts>:<userID>:<hex hmac-sha256(key, "userID:ts")>" )
//
// 后台这边没有「后台用户」概念，userID 固定 `admin`（平台只把它记进日志）。
func mintAdminToken(key string, ttl int) string {
	if ttl <= 0 {
		ttl = 300
	}
	ts := time.Now().Unix()
	const userID = "admin"
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(fmt.Sprintf("%s:%d", userID, ts)))
	return base64.URLEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%d:%s:%s", ts, userID, hex.EncodeToString(h.Sum(nil)))),
	)
}

// head 截断一段字节用于报错（按字符边界，不切坏 UTF-8）
func head(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return strings.TrimSpace(s)
	}
	r := []rune(s)
	if len(r) <= n {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
