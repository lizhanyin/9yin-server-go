package entergen

// jingmai_test.go View45 经脉容器行族守卫（再战江湖开脉链的编码面：
// 行属性集/登录整推/行号续编；官服建号-04.pcapng 26.748s #468..#679 定版）。

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func decU16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func decU32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

// decodeViewRowProps 0x18 视图行的属性字典（行布局 [18][vid:2][idx:2]
// [n:2]{[propidx:2][值]}——线型不在线上，宽度按属性表 TypeCode）。
func decodeViewRowProps(t *testing.T, frame []byte) map[string]string {
	t.Helper()
	n := int(decU16(frame, 5))
	off := 7
	out := map[string]string{}
	for i := 0; i < n; i++ {
		idx := decU16(frame, off)
		off += 2
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("未知属性 idx=%d", idx)
		}
		var val string
		switch entry.TypeCode {
		case 1:
			val = strconv.Itoa(int(frame[off]))
			off++
		case 2:
			val = strconv.Itoa(int(decU16(frame, off)))
			off += 2
		case 3:
			val = strconv.Itoa(int(int32(decU32(frame, off))))
			off += 4
		case 4:
			val = strconv.FormatInt(int64(binary.LittleEndian.Uint64(frame[off:])), 10)
			off += 8
		case 5:
			val = strconv.FormatFloat(float64(decU32(frame, off)), 'g', -1, 32)
			off += 4
		case 7:
			ln := int(decU32(frame, off))
			off += 4
			val = string(frame[off : off+ln-1])
			off += ln
		default:
			t.Fatalf("属性 %s 未覆盖的表线型 %d", entry.Name, entry.TypeCode)
		}
		out[entry.Name] = val
	}
	if off != len(frame) {
		t.Fatalf("行消费 %d / %d 字节", off, len(frame))
	}
	return out
}

// decodePlayerPropNameValue 玩家/视图对象 0x10 单属性帧的「名=值」
//（布局 [10][x][obj:4][own:4][n:2]{[propidx:2][值]}）。
func decodePlayerPropNameValue(t *testing.T, frame []byte) (string, string) {
	t.Helper()
	entry, ok := clientdata.PropertyByIndex(int(decU16(frame, 12)))
	if !ok {
		t.Fatalf("未知属性 idx=%d", decU16(frame, 12))
	}
	switch entry.TypeCode {
	case 1:
		return entry.Name, strconv.Itoa(int(frame[14]))
	case 3:
		return entry.Name, strconv.Itoa(int(int32(decU32(frame, 14))))
	}
	return entry.Name, "?"
}

// TestJingMaiRowFrameShape 单脉行属性集（jm_dantian 官服 8 列照抄：
// ConfigID/ItemType=1003/VarPropNo=141001/StaticData=41/Level=1/
// TotalFillValue=20100/Quality=1/WuXing=3；末三脉 6 列零省形状）。
func TestJingMaiRowFrameShape(t *testing.T) {
	frame, err := JingMaiRowFrame(0, rolestate.GuideMeridians[0])
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x18 {
		t.Fatalf("op = %#x", frame[0])
	}
	if view := decU16(frame, 1); view != ViewportJingMai {
		t.Fatalf("view = %d, want %d", view, ViewportJingMai)
	}
	if idx := decU16(frame, 3); idx != 1 {
		t.Fatalf("首行 idx = %d, want 1（1 基行号）", idx)
	}
	row := decodeViewRowProps(t, frame)
	want := map[string]string{
		"ConfigID": "jm_dantian", "ItemType": "1003", "VarPropNo": "141001",
		"StaticData": "41", "Level": "1", "TotalFillValue": "20100",
		"Quality": "1", "WuXing": "3",
	}
	for name, value := range want {
		if row[name] != value {
			t.Errorf("%s = %q, want %q", name, row[name], value)
		}
	}
	// 末脉（阳跷维脉）官服 6 列：Quality/WuXing 零省。
	tail, err := JingMaiRowFrame(15, rolestate.GuideMeridians[15])
	if err != nil {
		t.Fatal(err)
	}
	if idx := decU16(tail, 3); idx != 16 {
		t.Fatalf("末行 idx = %d, want 16", idx)
	}
	rowTail := decodeViewRowProps(t, tail)
	if _, has := rowTail["Quality"]; has {
		t.Errorf("末脉应省 Quality 列: %v", rowTail)
	}
	if rowTail["ConfigID"] != "jm_yangqiao_wei" || rowTail["TotalFillValue"] != "15800" {
		t.Errorf("末脉行 = %v", rowTail)
	}
}

// TestJingMaiViewFrames 登录整推：建视图（cap 32）+ 每脉一行；空经脉零行帧。
func TestJingMaiViewFrames(t *testing.T) {
	state := rolestate.RoleState{JingMai: rolestate.GuideMeridians}
	frames, err := JingMaiViewFrames(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1+len(rolestate.GuideMeridians) {
		t.Fatalf("帧数 = %d, want 建视图+16 行", len(frames))
	}
	if frames[0][0] != 0x15 {
		t.Fatalf("首帧 op = %#x, want 0x15 建视图", frames[0][0])
	}
	if capacity := decU16(frames[0], 3); capacity != 32 {
		t.Fatalf("容器容量 = %d, want 32（官服登录建视图）", capacity)
	}
	empty, err := JingMaiViewFrames(rolestate.RoleState{})
	if err != nil || len(empty) != 1 {
		t.Fatalf("空经脉帧数 = %d err=%v, want 仅建视图", len(empty), err)
	}
}

// TestJingMaiMaxLevelFrame 0x10 [View45, 行] MaxLevel=180（官服每脉随行）。
func TestJingMaiMaxLevelFrame(t *testing.T) {
	frame, err := JingMaiMaxLevelFrame(2)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 || frame[1] != 1 {
		t.Fatalf("op/x = %#x/%#x", frame[0], frame[1])
	}
	if obj := decU32(frame, 2); obj != uint32(ViewportJingMai) {
		t.Fatalf("obj = %d, want View45", obj)
	}
	if own := decU32(frame, 6); own != 3 {
		t.Fatalf("own = %d, want 行号 3", own)
	}
	if name, value := decodePlayerPropNameValue(t, frame); name != "MaxLevel" || value != "180" {
		t.Fatalf("属性 = %s=%s, want MaxLevel=180", name, value)
	}
}

// TestNeiGongRowFrame 学书追加行（View43 行号 = 下标+1，官服 #455 idx=1；
// WuXing 真值入行——内修面板属性列的显示源，0 值查表失败显示原码）。
func TestNeiGongRowFrame(t *testing.T) {
	book := rolestate.NeiGongBook{ConfigID: "ng_jh_001", ItemType: 1002, StaticData: 1, Level: 1, MaxLevel: 36, Total: 750, WuXing: 6}
	frame, err := NeiGongRowFrame(0, book)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x18 || decU16(frame, 1) != ViewportNeiGong || decU16(frame, 3) != 1 {
		t.Fatalf("头 = %#x view=%d idx=%d", frame[0], decU16(frame, 1), decU16(frame, 3))
	}
	row := decodeViewRowProps(t, frame)
	if row["ConfigID"] != "ng_jh_001" || row["ItemType"] != "1002" || row["StaticData"] != "1" {
		t.Errorf("书行 = %v", row)
	}
	if row["School"] != "ng_jh" {
		t.Errorf("School 派系串 = %q, want ng_jh", row["School"])
	}
	if row["WuXing"] != "6" {
		t.Errorf("WuXing = %q, want 6（阴柔——官服 ng_jh_001 行值）", row["WuXing"])
	}
}
