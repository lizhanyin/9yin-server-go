package fssoloworld

// neigongpersist_test.go 运行内功/当前血蓝持久化单测：登录书 varprop
// 刷新（2433 refreshAllModernEffects 口径）、战斗生存态随运行书与存档
// 持久值初始化、存档写回、215 换书战斗权威刷新。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/neigongdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// writeNeiGongTree 夹具 share 树（neigongdata_test 同款）：ng_wd_001
// （StaticData 5）30 级档行 + ng_wd_002（StaticData 6）1 级档行（带
// proppack 资源位）+ buff StaticData。
func writeNeiGongTree(t *testing.T) *neigongdata.Catalog {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"share/faculty/wuxuebaseinfo.ini":         "[ng_wd_001]\nnWuXing=5\n[ng_yh_001]\nnWuXing=6\n",
		"share/skill/neigong/neigong_static.ini":  "[5]\nMinVarPropNo=100\nMaxVarPropNo=100\n[6]\nMinVarPropNo=200\nMaxVarPropNo=200\n",
		"share/skill/neigong/neigong_varprop.ini": "[100]\nLevel=30\nBufferID=buf_fixture\nBufferLevel=4\nStrAdd=77\nStaAdd=66\nSpiAdd=9\n@PropModifyPackRec=300\n[200]\nLevel=1\nStrAdd=8\n@PropModifyPackRec=301\n",
		"share/skill/buff_new.ini":                "[buf_fixture]\nStaticData=4242\n",
		"share/modifypack/proppack.ini":           "[300]\nr=,MaxHPAdd,999,0\nr=,MaxMPAdd,999,0\n[301]\nr=,MaxMPAdd,10,0\nr=,MinMagicDefAdd,35,0\n",
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := neigongdata.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

// TestRefreshNeiGongBooksFromCatalog 登录刷新：五维 Add/BuffID/BuffLevel
// 按（StaticData, Level）档行重取、BuffStaticData 补全、幂等。白板建号
// 无书——书布局注入（授予链落档形态：ng_wd_001@30 带快照资源 Add、
// ng_wd_002@1 零快照）。
func TestRefreshNeiGongBooksFromCatalog(t *testing.T) {
	catalog := writeNeiGongTree(t)
	state, err := rolestate.BuildRoleState(rolestate.DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	state.NeiGong = []rolestate.NeiGongBook{
		{ConfigID: "ng_wd_001", StaticData: 5, ItemType: 1002, Level: 30, MaxLevel: 36,
			MaxHPAdd: 1630, MaxMPAdd: 490},
		{ConfigID: "ng_wd_002", StaticData: 6, ItemType: 1002, Level: 1, MaxLevel: 36},
	}
	if !refreshNeiGongBooks(catalog, &state) {
		t.Fatal("首刷应报变化")
	}
	book := equipBookOf(&state, "ng_wd_001")
	if book == nil || book.StrAdd != 77 || book.StaAdd != 66 || book.SpiAdd != 9 {
		t.Fatalf("五维 Add 未按档行刷新: %+v", book)
	}
	if book.BuffID != "buf_fixture" || book.BuffLevel != 4 || book.BuffStaticData != 4242 {
		t.Fatalf("buff 三元组未刷新: %+v", book)
	}
	// 资源 Add 补零不覆盖：ng_wd_001 快照 MaxHPAdd=1630（官服建号对拍值）
	// 不被 pack 999 顶掉；零快照的 ng_wd_002 补 pack 位 10/35。
	if book.MaxHPAdd != 1630 || book.MaxMPAdd != 490 {
		t.Fatalf("非零快照资源 Add 被覆盖: %+v", book)
	}
	l1 := equipBookOf(&state, "ng_wd_002")
	if l1 == nil || l1.StrAdd != 8 || l1.MaxMPAdd != 10 || l1.MinMagicDefAdd != 35 || l1.MaxHPAdd != 0 {
		t.Fatalf("零快照资源 Add 未补 pack 位: %+v", l1)
	}
	if refreshNeiGongBooks(catalog, &state) {
		t.Fatal("二刷应幂等（无变化）")
	}
	if refreshNeiGongBooks(nil, &state) {
		t.Fatal("缺目录应整体 no-op")
	}
}

// TestEnsureLiveUsesRunningBookAndPersistedHP 战斗生存态初始化：血上限 =
// 白板 + 五维系数推导 Add（entergen DerivedMaxHP，书跟 CurNeiGong——
// 旧版恒首本 ng_wd_001）、当前血取存档持久值钳上限。白板建号无书——
// 运行书注入（换书落档形态）。
func TestEnsureLiveUsesRunningBookAndPersistedHP(t *testing.T) {
	s, pl := newCustomTestServer(t)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, rolestate.NeiGongBook{
		ConfigID: "ng_yh_001", StaticData: 114, ItemType: 1002, Level: 30, MaxLevel: 71,
		StrAdd: 55, StaAdd: 57, DexAdd: 70, IngAdd: 97, SpiAdd: 67,
		MaxHPAdd: 4050, MaxMPAdd: 1130, MaxParryAdd: 5040, MinMagicDefAdd: 100,
	})
	st.CurNeiGong = "ng_yh_001"
	book := equipBookOf(st, "ng_yh_001")
	if book == nil {
		t.Fatal("注入运行书缺席")
	}
	wantMax := int(entergen.DerivedMaxHP(*st, *book))
	st.CurHP = wantMax / 3
	combat := &pl.combat
	combat.mu.Lock()
	s.ensureLiveLocked(combat, pl)
	live := combat.live
	combat.mu.Unlock()
	if live.maxHP != wantMax {
		t.Fatalf("maxHP=%d, want 系数推导 %d", live.maxHP, wantMax)
	}
	if live.hp != st.CurHP {
		t.Fatalf("hp=%d, want 存档持久值 %d", live.hp, st.CurHP)
	}
}

// TestPlayerMaxMPFollowsRunningBook MP 上限随运行书（系数推导）；ensureVitals
// 读存档持久蓝钳上限。白板建号无书——运行书注入（换书落档形态）。
func TestPlayerMaxMPFollowsRunningBook(t *testing.T) {
	s, pl := newCustomTestServer(t)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, rolestate.NeiGongBook{
		ConfigID: "ng_yh_001", StaticData: 114, ItemType: 1002, Level: 30, MaxLevel: 71,
		StrAdd: 55, StaAdd: 57, DexAdd: 70, IngAdd: 97, SpiAdd: 67,
		MaxHPAdd: 4050, MaxMPAdd: 1130, MaxParryAdd: 5040, MinMagicDefAdd: 100,
	})
	st.CurNeiGong = "ng_yh_001"
	book := equipBookOf(st, "ng_yh_001")
	if book == nil {
		t.Fatal("注入运行书缺席")
	}
	wantMax := int(entergen.DerivedMaxMP(*st, *book))
	if got := playerMaxMP(pl); got != wantMax {
		t.Fatalf("playerMaxMP=%d, want 系数推导 %d", got, wantMax)
	}
	st.CurMP = wantMax / 4
	combat := &pl.combat
	combat.mu.Lock()
	combat.ensureVitals(pl)
	mp := combat.mp
	combat.mu.Unlock()
	if mp != st.CurMP {
		t.Fatalf("会话 MP=%d, want 存档持久值 %d", mp, st.CurMP)
	}
	_ = s
}

// TestSyncLiveVitalsWriteBack 存档写回：会话血蓝 → roleState.CurHP/CurMP；
// 会话态未建（未进战斗/未用蓝）保持存档原值、零值不写。
func TestSyncLiveVitalsWriteBack(t *testing.T) {
	s, pl := newCustomTestServer(t)
	combat := &pl.combat
	combat.mu.Lock()
	combat.live = playerLive{ready: true, hp: 1234, maxHP: 2630}
	combat.mp, combat.mpReady = 321, true
	combat.mu.Unlock()
	s.syncLiveVitals(pl)
	if pl.roleState.CurHP != 1234 || pl.roleState.CurMP != 321 {
		t.Fatalf("写回失败: CurHP=%d CurMP=%d", pl.roleState.CurHP, pl.roleState.CurMP)
	}
	// 会话态清空（重连首拍）：存档值保持，不被零值冲掉。
	combat.mu.Lock()
	combat.live, combat.mp, combat.mpReady = playerLive{}, 0, false
	combat.mu.Unlock()
	pl.roleState.CurHP, pl.roleState.CurMP = 555, 222
	s.syncLiveVitals(pl)
	if pl.roleState.CurHP != 555 || pl.roleState.CurMP != 222 {
		t.Fatalf("会话态未建不应冲存档: CurHP=%d CurMP=%d", pl.roleState.CurHP, pl.roleState.CurMP)
	}
}

// TestUseNeiGongRefreshesCombatAuthority 215 换书后战斗权威跟新书：
// 血/蓝/怒上限随新书（五维系数推导），当前值**保留钳新上限**（高换低
// =新低值 103；低换高=旧值 95/30 不回满）并重推 HPRatio/HP/MP/SP 帧、
// 面板缓存作废。
func TestUseNeiGongRefreshesCombatAuthority(t *testing.T) {
	s, pl := newCustomTestServer(t)
	resetNeiGongCatalog()
	t.Cleanup(resetNeiGongCatalog)
	st := &pl.roleState
	// 白板建号无书——先落一本运行书（换书前形态），再追加换书目标 ng_wd_003。
	st.NeiGong = append(st.NeiGong, rolestate.NeiGongBook{
		ConfigID: "ng_wd_001", StaticData: 5, ItemType: 1002, Level: 30, MaxLevel: 36,
		MaxHPAdd: 1630, MaxMPAdd: 490,
	})
	st.CurNeiGong = "ng_wd_001"
	st.NeiGong = append(st.NeiGong, rolestate.NeiGongBook{
		ConfigID: "ng_wd_003", StaticData: 5459, ItemType: 1100,
		Level: 1, MaxLevel: 36, Total: 200, MaxPower: 100, BuffID: "buf_ng_wd_003_1",
	})
	combat := &pl.combat
	combat.mu.Lock()
	combat.live = playerLive{ready: true, hp: 900, maxHP: st.HP + st.NeiGong[0].MaxHPAdd, sp: 95, maxSP: 90}
	combat.mp, combat.mpReady = 30, true
	combat.mu.Unlock()
	pl.panel = map[string]float64{"Str": 1} // 旧面板缓存
	frames := captureFrames(t)

	s.dispatchCustom(pl, customMsg(215, customStr("ng_wd_003")))

	book3 := equipBookOf(st, "ng_wd_003")
	wantMaxHP := int(entergen.DerivedMaxHP(*st, *book3)) // 白板+系数推导（书无资源分量）
	combat.mu.Lock()
	live, mp := combat.live, combat.mp
	combat.mu.Unlock()
	if live.maxHP != wantMaxHP {
		t.Fatalf("换书后 maxHP=%d, want 系数推导 %d", live.maxHP, wantMaxHP)
	}
	if live.hp != want2(wantMaxHP, 900) { // 高值换低值：以新低值为准
		t.Fatalf("换书后 hp=%d, want 钳新上限 %d", live.hp, want2(wantMaxHP, 900))
	}
	if mp != 30 { // 30 < 新上限：低值换高值保留旧值
		t.Fatalf("换书后 mp=%d, want 保留旧值 30", mp)
	}
	if live.maxSP != 110 || live.sp != 95 { // 三内档 80+30；sp 95<110 保留
		t.Fatalf("换书后 SP 权威异常: max=%d sp=%d", live.maxSP, live.sp)
	}
	if pl.panel != nil {
		t.Fatal("换书后面板缓存应作废重建")
	}
	// 资源重推帧：HPRatio/HP/MP/SP 四单属性（MaxSP 专帧之后）。
	got := *frames
	if len(got) != 1+4+1+8+1 {
		t.Fatalf("换书帧数=%d, want 15（MaxSP+资源4+资源批+面板8+m169）: ops=% x", len(got), opOf(got))
	}
	for i := 1; i <= 4; i++ {
		if got[i][0] != 0x10 {
			t.Fatalf("资源重推第 %d 帧非 0x10: % x", i, got[i])
		}
	}
}

func want2(a, b int) int {
	if b > a {
		return a
	}
	return b
}
