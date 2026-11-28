package fssoloworld

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// TestSwitchSyncIDsDefault 未设环境变量＝官服全表。
func TestSwitchSyncIDsDefault(t *testing.T) {
	t.Setenv("NINEYIN_SWITCH_SYNC", "")
	got := switchSyncIDs()
	if len(got) != len(entergen.OfficialSwitchIDs) || got[0] != entergen.OfficialSwitchIDs[0] ||
		got[len(got)-1] != entergen.OfficialSwitchIDs[len(got)-1] {
		t.Fatalf("缺省应取官服全表, got %d 项 %v…", len(got), got[:3])
	}
}

// TestSwitchSyncIDsEnv off 不发；逗号表覆盖（非法项忽略、全非法＝不发）。
func TestSwitchSyncIDsEnv(t *testing.T) {
	t.Setenv("NINEYIN_SWITCH_SYNC", "off")
	if got := switchSyncIDs(); got != nil {
		t.Fatalf("off 应不发, got %v", got)
	}
	t.Setenv("NINEYIN_SWITCH_SYNC", " 1509, 0x5e5, 2919 ")
	got := switchSyncIDs()
	if len(got) != 3 || got[0] != 1509 || got[1] != 0x5e5 || got[2] != 2919 {
		t.Fatalf("逗号表解析错误: %v", got)
	}
	t.Setenv("NINEYIN_SWITCH_SYNC", "abc, 0, -3")
	if got := switchSyncIDs(); got != nil {
		t.Fatalf("全非法应不发, got %v", got)
	}
}
