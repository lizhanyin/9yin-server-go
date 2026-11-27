package entergen

// neigongpersist_test.go 登录帧随运行内功/持久化血蓝的单测：0x0D/0x10
// 的 buff 串、BufferInfo4、五行、HP/MP/Ratio 按 ctx 动态取（旧版写死
// ng_wd_001 建号态——换书重登被两仪护心功覆盖的回归锚）。

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// runningNeiGongContext 换书后的进场上下文：CurNeiGong=第二本（易阳护体
// 心法 ng_yh_001，BuffID 带级）、五行阴柔 6、血蓝持久值半血。白板建号
// 无书——书布局注入（授予链/换书落档形态，数值取旧 StarterSet 官方
// 对拍值）。
func runningNeiGongContext(t *testing.T) Context {
	t.Helper()
	state, err := rolestate.BuildRoleState(rolestate.DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatalf("build role state: %v", err)
	}
	state.NeiGong = []rolestate.NeiGongBook{
		{ConfigID: "ng_wd_001", StaticData: 5, ItemType: 1002, Level: 30, MaxLevel: 36,
			StrAdd: 45, StaAdd: 31, DexAdd: 13, IngAdd: 46, SpiAdd: 13,
			MaxHPAdd: 1630, MaxMPAdd: 490, MaxParryAdd: 840, MinMagicDefAdd: 45},
		{ConfigID: "ng_yh_001", StaticData: 114, ItemType: 1002, Level: 30, MaxLevel: 71,
			StrAdd: 55, StaAdd: 57, DexAdd: 70, IngAdd: 97, SpiAdd: 67,
			MaxHPAdd: 4050, MaxMPAdd: 1130, MaxParryAdd: 5040, MinMagicDefAdd: 100},
	}
	state.CurNeiGong = "ng_yh_001"
	state.NeiGong[1].BuffID, state.NeiGong[1].BuffLevel = "buf_ng_yh_001", 2
	// 半血半蓝（上限 = 白板 + 五维系数推导 Add，resourceadds.go 同面）。
	maxHP := int(DerivedMaxHP(state, state.NeiGong[1]))
	maxMP := int(DerivedMaxMP(state, state.NeiGong[1]))
	state.CurHP, state.CurMP = maxHP/2, maxMP/2
	return Context{
		Identity:      Identity{ObjectID: 0x11111111, OwnerID: 0x22222222},
		RoleName:      "persist",
		State:         state,
		NeiGongWuXing: 6,
		CurHP:         state.CurHP,
		CurMP:         state.CurMP,
		BuffInfo4:     "3321,11111111,22222222,2,0,0",
		Stamp:         1795354397233,
		SP:            7,
		GuideStep:     6,
	}
}

// TestAppearanceFrameFollowsRunningNeiGong 0x0D 归档帧随运行书：buff 串/
// BufferInfo4/五行/HP/MP 不再写死 ng_wd_001 建号态。
func TestAppearanceFrameFollowsRunningNeiGong(t *testing.T) {
	ctx := runningNeiGongContext(t)
	frame, err := AppearanceFrame(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(frame, []byte("buf_ng_yh_001,local-4,2,")) {
		t.Fatal("0x0D buff 串未随运行书（buf_ng_yh_001 缺席）")
	}
	if bytes.Contains(frame, []byte("buf_ng_wd_001")) {
		t.Fatal("0x0D buff 串仍写死 buf_ng_wd_001")
	}
	if !bytes.Contains(frame, []byte(ctx.BuffInfo4)) {
		t.Fatal("0x0D BufferInfo4 未用 ctx 动态槽串")
	}
	if !bytes.Contains(frame, []byte("ng_yh_001\x00")) {
		t.Fatal("0x0D CurNeiGong 未随运行书")
	}
	// 五行 6（阴柔）：CurNeiGongWuXing 的 Int32 编码字节应在帧内。
	if !bytes.Contains(frame, mustPropBytes("CurNeiGongWuXing", Int32Value(6))) {
		t.Fatal("0x0D 五行未随运行书（写死 5 或缺省 0）")
	}
	// 半血：HP 与 HPRatio 都不再是满值口径。
	if !bytes.Contains(frame, mustPropBytes("HP", Int32Value(int32(ctx.CurHP)))) {
		t.Fatal("0x0D HP 未带持久值")
	}
	ratio := int32(ctx.CurHP * 100 / int(DerivedMaxHP(ctx.State, ctx.State.NeiGong[1])))
	if !bytes.Contains(frame, mustPropBytes("HPRatio", Int32Value(ratio))) {
		t.Fatalf("0x0D HPRatio 未按持久值折算（want %d）", ratio)
	}
}

// TestVitalBatchFollowsRunningNeiGong 0x10 全量批同源：BufferInfo4 槽/
// 五行/HP/MP 随运行书与持久血蓝。
func TestVitalBatchFollowsRunningNeiGong(t *testing.T) {
	ctx := runningNeiGongContext(t)
	frame, err := PlayerVitalBatchFrame(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(frame, []byte(ctx.BuffInfo4)) {
		t.Fatal("0x10 BufferInfo4 槽未用 ctx 动态槽串（写死 1742 ng_wd_001）")
	}
	if !bytes.Contains(frame, []byte("buf_ng_yh_001,local-4,2,")) {
		t.Fatal("0x10 BufferListStr 未随运行书")
	}
	if !bytes.Contains(frame, mustPropBytes("CurNeiGongWuXing", Int32Value(6))) {
		t.Fatal("0x10 五行未随运行书")
	}
	if !bytes.Contains(frame, mustPropBytes("MP", Int32Value(int32(ctx.CurMP)))) {
		t.Fatal("0x10 MP 未带持久值")
	}
}

// TestBufferListStringNoBuffBook 无 buff 书（BuffID 空）第三段省略、
// 前两段（组标志/报伤）保留。
func TestBufferListStringNoBuffBook(t *testing.T) {
	ctx := runningNeiGongContext(t)
	ctx.State.NeiGong[1].BuffID = ""
	got := bufferListString(ctx)
	if !strings.Contains(got, "buff_group_flag_101,local-1,1,") ||
		!strings.Contains(got, "buf_baosd_02,local-2,0,") {
		t.Fatalf("前两段缺席: %q", got)
	}
	if strings.Contains(got, "local-4") {
		t.Fatalf("无 buff 书不应有 local-4 段: %q", got)
	}
}

// TestCurrentVitalsClampAndRatio 当前血蓝取值面：缺省满值、持久值直取、
// 超上限钳制、Ratio 封顶 100。
func TestCurrentVitalsClampAndRatio(t *testing.T) {
	full := Context{}
	hp, mp, hpR, mpR := currentVitals(full, 1000, 500)
	if hp != 1000 || mp != 500 || hpR != 100 || mpR != 100 {
		t.Fatalf("缺省应满值: %d/%d %d/%d", hp, mp, hpR, mpR)
	}
	partial := Context{CurHP: 250, CurMP: 100}
	hp, mp, hpR, mpR = currentVitals(partial, 1000, 500)
	if hp != 250 || mp != 100 || hpR != 25 || mpR != 20 {
		t.Fatalf("持久值应直取: %d/%d %d/%d", hp, mp, hpR, mpR)
	}
	over := Context{CurHP: 5000, CurMP: 9999}
	hp, mp, hpR, _ = currentVitals(over, 1000, 500)
	if hp != 1000 || mp != 500 || hpR != 100 {
		t.Fatalf("超上限应钳制: %d/%d %d", hp, mp, hpR)
	}
	if _, _, _, mpR := currentVitals(over, 0, 0); mpR != 100 {
		t.Fatal("上限非正 ratio 应 100（零除防护）")
	}
}

// TestStandingRegenFormula 站立回复五维公式（官方逐点表：体魄每 5 秒
// +0.07 生命、罡气每 5 秒 +0.2 蓝）——每秒小数速率，与面板 Add 列同速。
func TestStandingRegenFormula(t *testing.T) {
	if got := HPRegenPerSecond(34); got != 34.0*7/500 {
		t.Fatalf("HPRegenPerSecond(34)=%v", got)
	}
	if got := MPRegenPerSecond(16); got != 16.0/25 {
		t.Fatalf("MPRegenPerSecond(16)=%v", got)
	}
	if per5 := HPRegenPerSecond(34) * 5; per5 != 2.38 {
		t.Fatalf("Sta34 每 5 秒回血 %v, want 2.38", per5)
	}
	if per5 := MPRegenPerSecond(16) * 5; per5 != 3.2 {
		t.Fatalf("Spi16 每 5 秒回蓝 %v, want 3.2", per5)
	}
}

// TestDerivedResourceAddsOfficialAnchor 官方 52626 锚点（docs/official/
// 初始号0x0D外观属性全表.txt）：白板五维 30×5 + ng_wd_001@30（五维
// 45/31/13/46/13、资源 1630/490/840/45）→ 推导 2207/837/840/54——
// Str=75·2+Sta=61·7+1630=2207、Ing=76·4+Spi=43·1+490=837、
// round(0.2·43)+45=54，与官方帧逐值相等（系数层 + 书资源分量双层
// 结构的铁证）。本线建号白板是官服建号口径 3×5，锚点用合成 30×5 态。
func TestDerivedResourceAddsOfficialAnchor(t *testing.T) {
	state := rolestate.RoleState{Str: 30, Sta: 30, Dex: 30, Ing: 30, Spi: 30}
	book := rolestate.NeiGongBook{ConfigID: "ng_wd_001", Level: 30,
		StrAdd: 45, StaAdd: 31, DexAdd: 13, IngAdd: 46, SpiAdd: 13,
		MaxHPAdd: 1630, MaxMPAdd: 490, MaxParryAdd: 840, MinMagicDefAdd: 45}
	adds := DerivedResourceAdds(state, book)
	want := ResourceAdds{MaxHPAdd: 2207, MaxMPAdd: 837, MaxParryAdd: 840, MinMagicDefAdd: 54}
	if adds != want {
		t.Fatalf("官方锚点推导偏差: got %+v want %+v", adds, want)
	}
}

// mustPropBytes 属性编码字节（与帧内同名属性同构——appendProperty 直出）。
func mustPropBytes(name string, value PropertyValue) []byte {
	msg := appendProperty(nil, ap(name, value))
	return msg
}
