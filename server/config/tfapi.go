package config

// TFAPI 平台服务（tf-platform-api）的**服务端**接入点。
//
// ## 为什么后台要连平台
//
// 后台和平台共用同一个库，所以「版本记录」后台自己就能写（见
// `service/mall/game.go` 的 Publish）。但**包文件**不归后台管：
// 平台用 `StaticPath/games/<id>/<ver>/` 托管包，再由 `PackageBaseURL`
// （将来是 CDN）对外提供地址。后台没有那棵目录，也不该有 ——
// 一旦两个服务各自往自己盘上写包，客户端拿到的 `url` 就取决于
// 「哪台机器发的版」，换部署就全 404。
//
// 所以后台只做一件事：把文件**转发**给平台的上传接口，让平台决定落哪。
//
// ## 为什么不给浏览器直连平台
//
// 平台的后台接口用 `X-Api-Token`（HMAC，密钥是 AdminSignKey）。
// 那个密钥是**服务端机密**，不能发给浏览器；后台只能自己签、自己发。
type TFAPI struct {
	// BaseURL 平台服务地址（不带结尾斜杠）。空 = 上传功能不可用
	BaseURL string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`
	// AdminSignKey 与平台 `ServiceSettings.AdminSignKey` **逐字节一致**
	AdminSignKey string `mapstructure:"admin-sign-key" json:"admin-sign-key" yaml:"admin-sign-key"`
	// AdminTokenTTL 令牌有效期（秒）。0 = 300
	//
	// 这里只用于「签一次、马上用」的转发，不需要长有效期 ——
	// 有效期越长，密钥泄露后窗口越大。
	AdminTokenTTL int `mapstructure:"admin-token-ttl" json:"admin-token-ttl" yaml:"admin-token-ttl"`
}
