package mall

import "testing"

// MakeInviteCode 必须和 tf-platform-api `entities.MakeInviteCode` 逐字节一致。
//
// 这两个值来自 `tf-platform-api/scripts/check_template.py` 里**独立用 Python
// 实现一遍**的交叉校验（不是从 Go 代码抄的）：改了公式、改了盐、改了大小写，
// 这个测试会红 —— 否则症状是「后台印的码，App 里绑不上」，两端都不报错。
func TestMakeInviteCodeMatchesPlatform(t *testing.T) {
	cases := []struct {
		uid  uint
		want string
	}{
		{990001, "TFL7W1-B65E"},
		{990002, "TFL7W2-AC24"},
		{42, "TF16-8816"},
	}
	for _, c := range cases {
		if got := MakeInviteCode(c.uid); got != c.want {
			t.Fatalf("MakeInviteCode(%d) = %q, want %q（两边公式分叉了）", c.uid, got, c.want)
		}
	}
	if got := MakeInviteCode(0); got != "" {
		t.Fatalf("uid=0 应该返回空串，拿到 %q", got)
	}
}

// cleanExtra 必须剔掉 typed 列的键，否则下发时运营改的值会被 Extra 旧值盖回去。
func TestCleanExtraStripsTypedKeys(t *testing.T) {
	got, err := cleanExtra(`{"price":30,"allowOvertime":true,"base_points":99}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"allowOvertime":true}` {
		t.Fatalf("cleanExtra = %s", got)
	}
	if _, err := cleanExtra(`[1,2,3]`); err == nil {
		t.Fatal("Extra 是数组竟然被放行了")
	}
	if _, err := cleanExtra(`not json`); err == nil {
		t.Fatal("非法 JSON 竟然被放行了")
	}
	if got, err := cleanExtra(""); err != nil || got != "{}" {
		t.Fatalf("空 Extra = %q, err=%v", got, err)
	}
}
