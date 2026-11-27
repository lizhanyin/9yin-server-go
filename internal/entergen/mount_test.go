package entergen

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// TestMountPlanShape 钉死挂载计划（golden 0x14 帧提取）：48 项、4 全局 +
// 44 玩家域、顺序首尾锚、全部记录名在当前方言表可解析。
func TestMountPlanShape(t *testing.T) {
	if len(MountPlan) != 48 {
		t.Fatalf("挂载计划 48 项（golden 实证），得 %d", len(MountPlan))
	}
	global, player := 0, 0
	for _, spec := range MountPlan {
		switch spec.Sub {
		case 2:
			global++
		case 0:
			player++
		default:
			t.Fatalf("sub 域值非法 %d（%s）", spec.Sub, spec.Name)
		}
		clientdata.MustRecordIndexByName(spec.Name) // 缺名 panic
	}
	if global != 4 || player != 44 {
		t.Fatalf("sub 分布 4 全局 + 44 玩家，得 %d/%d", global, player)
	}
	anchors := []struct {
		idx  int
		name string
	}{
		{0, "WeatherRec"}, {1, "Time_Limit_Form_Rec"}, {2, "scene_stall_rec"}, {3, "prep_show_npc_rec"},
		{4, "TiGuanRec"}, {5, "Task_Accepted"}, {7, "Task_Completed"},
		{47, "OffLineJob_Log"},
	}
	for _, a := range anchors {
		if MountPlan[a.idx].Name != a.name {
			t.Fatalf("顺序锚 #%d = %s，应 %s", a.idx, MountPlan[a.idx].Name, a.name)
		}
	}
}

// TestMountSequenceFrames 行推序列（2026-11-22 晚定案：0x14 挂载帧不发
// ——aow/golden 方言遗物，2433 fork 官服对拍零 0x14）。白板建号无任务门/
// 轻功——零行推；有门档（授予链落档形态）带 Task_Completed 位图行、
// Task_Accepted 0x29 批量、QingGongRec 18 行。
func TestMountSequenceFrames(t *testing.T) {
	ctx := testContext(t)
	frames, err := MountSequenceFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mounts, bitmapRows, qgRows, accepted29 := countMountShapes(t, frames)
	if mounts != 0 {
		t.Fatalf("0x14 挂载帧应为 0（fork 方言不发），得 %d", mounts)
	}
	if accepted29 {
		t.Fatal("白板建号无任务门不应有 Task_Accepted 0x29 批量")
	}
	if bitmapRows != 0 || qgRows != 0 {
		t.Fatalf("白板建号零行推，得 位图 %d / 轻功 %d", bitmapRows, qgRows)
	}

	// 有门档：OpenGate 49634 → 位图 final row 1551 → 1552 行（含全零行）。
	ctx.State.TaskAccepted = []uint32{1340}
	ctx.State.TaskCompleted = []uint32{49634, 1346}
	for i := 1; i <= 17; i++ {
		ctx.State.QingGongCategories = append(ctx.State.QingGongCategories, fmt.Sprintf("qinggong_%d", i))
	}
	ctx.State.QingGongCategories = append(ctx.State.QingGongCategories, "qinggong_gaojie")
	frames, err = MountSequenceFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mounts, bitmapRows, qgRows, accepted29 = countMountShapes(t, frames)
	if mounts != 0 {
		t.Fatalf("0x14 挂载帧应为 0（fork 方言不发），得 %d", mounts)
	}
	if !accepted29 {
		t.Fatal("Task_Accepted 0x29 批量缺席")
	}
	if qgRows != 18 {
		t.Fatalf("QingGongRec 18 行，得 %d", qgRows)
	}
	if bitmapRows != 1552 {
		t.Fatalf("位图 0x11 行 1552（final-row 前全零行），得 %d", bitmapRows)
	}
}

// countMountShapes 行推族形状计数：0x14 挂载帧、Task_Completed 位图 0x11
// 行、QingGongRec 0x11 行、Task_Accepted 0x29 批量。
func countMountShapes(t *testing.T, frames [][]byte) (mounts, bitmapRows, qgRows int, accepted29 bool) {
	t.Helper()
	for _, f := range frames {
		switch f[0] {
		case 0x14:
			mounts++
		case 0x11:
			if f[1] != 0 {
				break
			}
			switch rec := binary.LittleEndian.Uint16(f[10:]); rec {
			case clientdata.MustRecordIndexByName("Task_Completed"):
				bitmapRows++
			case clientdata.MustRecordIndexByName("QingGongRec"):
				qgRows++
			}
		case 0x29:
			accepted29 = true
		}
	}
	return mounts, bitmapRows, qgRows, accepted29
}

// mountName 从 0x14 帧反查记录名（测试用）。
func mountName(f []byte) string {
	idx := binary.LittleEndian.Uint16(f[10:])
	schema, _ := clientdata.RecordSchemaByIndex(int(idx))
	return schema.Name
}

// countFollowingRows 从 from 起数连续同记录 0x11 行数。
func countFollowingRows(frames [][]byte, from int) int {
	n := 0
	for i := from; i < len(frames) && frames[i][0] == 0x11; i++ {
		n++
	}
	return n
}
