package fssoloworld

// switchsync.go 进场功能开关表的取值（entergen m414 ["sync", …] 的输入）。
//
// 缺省＝官服 CN 2026 全表（entergen.OfficialSwitchIDs，641 项）：客户端引擎
// SwitchManager 的开关树在「初出茅庐/再战江湖」黑幕窗、m414 gamestep 之外还
// 门控一批功能（NPC 对话语音 0x5e5、新版人物界面 2919、各活动入口…），官服每次
// 完整进场在 0x0B 之后下发一次。NINEYIN_SWITCH_SYNC 供真机 A/B 与联调定位。

import (
	"os"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// switchSyncIDs 进场功能开关表：
//
//	未设/空   官服全表
//	off       不发（CheckSwitchEnable 门控的功能全灭——对照用）
//	其余      逗号分隔的开关号表（如 "1509,2919"），非法项忽略；全部非法＝不发
func switchSyncIDs() []int32 {
	raw := strings.TrimSpace(os.Getenv("NINEYIN_SWITCH_SYNC"))
	switch {
	case raw == "":
		return entergen.OfficialSwitchIDs
	case strings.EqualFold(raw, "off"):
		return nil
	}
	var ids []int32
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		id, err := strconv.ParseUint(field, 0, 31)
		if err != nil || id == 0 {
			continue
		}
		ids = append(ids, int32(id))
	}
	return ids
}
