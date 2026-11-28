package entergen

// switchsync_test.go 功能开关表 m414 ["sync", …] 对拍：golden＝官服「初始账号
// 登陆-完整引导-女」#479（op 0x27 压缩批，0x0B 进场之后）的批内层明文，用客户端
// fxnet2.dll 的真 LZ 解压器解出（2026-11-28，tmp/npctalk/reallz.py）。

import (
	"bytes"
	"os"
	"testing"
)

// TestSwitchSyncFrameMatchesOfficialBatch 官服全表编码结果 = 0x1E 外壳 + 官服批
// 内层逐字节相同（官服经 LZ 压进 op 0x27，本服发明文，handler 同一个）。
func TestSwitchSyncFrameMatchesOfficialBatch(t *testing.T) {
	golden, err := os.ReadFile("testdata/official_switch_sync.bin")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got, err := SwitchSyncFrame(OfficialSwitchIDs)
	if err != nil {
		t.Fatalf("SwitchSyncFrame: %v", err)
	}
	want := append([]byte{0x1E}, golden...)
	if !bytes.Equal(got, want) {
		i := 0
		for i < len(got) && i < len(want) && got[i] == want[i] {
			i++
		}
		t.Fatalf("开关表帧与官服批内层不一致（首个差异 @%d）:\n got  len=%d …% x\n want len=%d …% x",
			i, len(got), got[i:min(i+16, len(got))], len(want), want[i:min(i+16, len(want))])
	}
	// 壳内首值=消息号 414、次值="sync"（引擎 SwitchManager handler 按首参分流）。
	if got[0] != 0x1E || !bytes.HasPrefix(got[3:], []byte{2, 0x9e, 1, 0, 0, 6, 5, 0, 0, 0, 's', 'y', 'n', 'c', 0}) {
		t.Fatalf("帧头不是 m414 [\"sync\", …]: % x", got[:24])
	}
}

// TestOfficialSwitchIDsShape 官服表严格升序、全正（引擎 handler 逐值取整插
// std::map；官服表无重复无乱序），并含已知门控项。
func TestOfficialSwitchIDsShape(t *testing.T) {
	if len(OfficialSwitchIDs) != 641 {
		t.Fatalf("官服开关表项数 = %d, want 641", len(OfficialSwitchIDs))
	}
	for i, id := range OfficialSwitchIDs {
		if id <= 0 {
			t.Fatalf("第 %d 项非正: %d", i, id)
		}
		if i > 0 && id <= OfficialSwitchIDs[i-1] {
			t.Fatalf("第 %d 项未严格升序: %d ≤ %d", i, id, OfficialSwitchIDs[i-1])
		}
	}
	// 0x5e5=NPC 对话语音闸门（HeadGame:PlaySoundByFile 入口 CheckSwitchEnable）；
	// 2919=新版人物界面按钮（参考端 2433 记录）。
	for _, want := range []int32{0x5e5, 2919} {
		found := false
		for _, id := range OfficialSwitchIDs {
			found = found || id == want
		}
		if !found {
			t.Fatalf("官服开关表缺门控项 %d", want)
		}
	}
}

// TestSwitchSyncFrameEmptyAndCustom 空表零帧；自定表按 [414, "sync", id…] 编码。
func TestSwitchSyncFrameEmptyAndCustom(t *testing.T) {
	if got, err := SwitchSyncFrame(nil); err != nil || got != nil {
		t.Fatalf("空表应零帧, got % x, %v", got, err)
	}
	got, err := SwitchSyncFrame([]int32{7, 0x5e5})
	if err != nil {
		t.Fatalf("SwitchSyncFrame: %v", err)
	}
	assertOfficial(t, got, nil,
		"1e 04 00 02 9e 01 00 00 06 05 00 00 00 73 79 6e 63 00 02 07 00 00 00 02 e5 05 00 00", "自定开关表帧")
}
