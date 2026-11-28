package fssoloworld

// guidematerial_test.go 进场/切图的引导属性补推与开关表位次（2026-11-28）：
//   - 完整进场：m414 ["sync", 开关表] 紧随 0x0B 之后、0x0D 之前（官服完整引导-
//     女 #476→#479）；HelpStepID 的 0x10 在 0x0D 之后（场内玩家对象只认物化后
//     的属性，0x0B 前单发的那份被进场重建冲掉）；
//   - 切图重入：不重发 sync（连接级一次），HelpStepID 照常补推。

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// storeEnterPayload 0x02 load 回执 {client=7, serial=1, ok=1}（RST1 缺席保留预置
// roleState）。
func storeEnterPayload(pl *Player) []byte {
	p := make([]byte, 12)
	binary.LittleEndian.PutUint32(p[0:], 7)
	binary.LittleEndian.PutUint32(p[4:], 1)
	binary.LittleEndian.PutUint32(p[8:], 1)
	return p
}

// TestEnterPushesSwitchSyncAfterEntry 完整进场恰发一次 m414 sync，位次＝0x0B 的
// 下一帧、0x0D 之前。
func TestEnterPushesSwitchSyncAfterEntry(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)
	if !s.storeLoadEnter(0, storeEnterPayload(pl)) {
		t.Fatal("storeLoadEnter rejected")
	}
	want, err := entergen.SwitchSyncFrame(entergen.OfficialSwitchIDs)
	if err != nil {
		t.Fatal(err)
	}
	entryAt, syncAt, materialAt, syncs := -1, -1, -1, 0
	for i, f := range *frames {
		switch {
		case f[0] == 0x0B && entryAt < 0:
			entryAt = i
		case bytes.Equal(f, want):
			syncs++
			if syncAt < 0 {
				syncAt = i
			}
		case f[0] == 0x0D && materialAt < 0:
			materialAt = i
		}
	}
	if syncs != 1 || entryAt < 0 || syncAt != entryAt+1 || materialAt <= syncAt {
		t.Fatalf("sync 位次不对: syncs=%d entry=%d sync=%d 0x0D=%d ops=% x",
			syncs, entryAt, syncAt, materialAt, opOf(*frames))
	}
}

// TestEnterPushesGuidePropertiesAfterMaterialize 引导链上的角色：HelpStepID 的
// 0x10 在 0x0D 之后（且 0x0B 之前那一份照发）。
func TestEnterPushesGuidePropertiesAfterMaterialize(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.roleState.GameStep = 4
	pl.roleState.HelpStepID = "yanyuzhuang_02"
	frames := captureFrames(t)
	if !s.storeLoadEnter(0, storeEnterPayload(pl)) {
		t.Fatal("storeLoadEnter rejected")
	}
	want, err := entergen.HelpStepFrame(pl.entergenIdentity(), "yanyuzhuang_02")
	if err != nil {
		t.Fatal(err)
	}
	entryAt, materialAt, before, after := -1, -1, false, false
	for i, f := range *frames {
		switch {
		case f[0] == 0x0B && entryAt < 0:
			entryAt = i
		case f[0] == 0x0D && materialAt < 0:
			materialAt = i
		case bytes.Equal(f, want):
			if materialAt < 0 {
				before = true
			} else {
				after = true
			}
		}
	}
	if entryAt < 0 || materialAt < 0 || !before || !after {
		t.Fatalf("HelpStepID 应在 0x0B 前与 0x0D 后各一份: entry=%d 0x0D=%d before=%v after=%v",
			entryAt, materialAt, before, after)
	}
}

// TestSceneSwitchPushesGuidePropertiesWithoutSync 切图重入：HelpStepID 在 0x0D
// 之后补推；开关表不重发（引擎全局态，连接级一次）。
func TestSceneSwitchPushesGuidePropertiesWithoutSync(t *testing.T) {
	s, pl := newCustomTestServer(t)
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })

	pl.roleState.GameStep = 4
	pl.roleState.HelpStepID = "yanyuzhuang_02"
	pl.entryReady = true // 旧场景已就绪
	frames := captureFrames(t)
	dest, _ := resolveSceneDestination("born01")
	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.pendingSceneSwitch(pl) == nil {
		t.Fatal("awaitReady not armed after gap")
	}
	want, err := entergen.HelpStepFrame(pl.entergenIdentity(), "yanyuzhuang_02")
	if err != nil {
		t.Fatal(err)
	}
	sync, err := entergen.SwitchSyncFrame(entergen.OfficialSwitchIDs)
	if err != nil {
		t.Fatal(err)
	}
	materialAt, after := -1, false
	for i, f := range (*frames)[:] {
		if bytes.Equal(f, sync) {
			t.Fatalf("切图重入了开关表（应连接级一次）: frame #%d", i)
		}
		if f[0] == 0x0D {
			materialAt = i
		}
		if materialAt >= 0 && bytes.Equal(f, want) {
			after = true
		}
	}
	if materialAt < 0 || !after {
		t.Fatalf("切图重入后未补推 HelpStepID: 0x0D=%d after=%v ops=% x", materialAt, after, opOf(*frames))
	}
}
