package clientdata

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestOfficialPropertyTableShape 校验嵌入表规模与关键锚点
// （c.pcapng 7074 线 0x09 帧提取 + 61 行尾追段，2026-11-14）。
func TestOfficialPropertyTableShape(t *testing.T) {
	if got := OfficialPropertyCount(); got != 2494 {
		t.Fatalf("official property table entries = %d, want 2494", got)
	}
	// 头块换血实证：0=None(不可编码)、1=HomeCount（fork 场景域 0=Type 已退役）。
	if entries := OfficialPropertyEntries(); entries[0].Name != "None" || entries[1].Name != "HomeCount" {
		t.Fatalf("head block changed: [0]=%s [1]=%s", entries[0].Name, entries[1].Name)
	}
}

// TestOfficialPropertyTableDomainResolution 校验同名多处的域解析:
// 场景域首现、玩家域末现(镜像 Java PropertyTable 语义)。
// 锚点全部来自外服私服帧实解（0x0D #1777 / §11.4 View45）。
func TestOfficialPropertyTableDomainResolution(t *testing.T) {
	if _, ok := SceneProperty("NoSuchPropertyName"); ok {
		t.Error("unknown name resolved in scene domain")
	}
	if _, ok := PlayerProperty("NoSuchPropertyName"); ok {
		t.Error("unknown name resolved in player domain")
	}
	// 场景域首现 vs 玩家域末现漂移的实名样本。
	playerAnchors := []struct {
		name  string
		index int
	}{
		{"AutoSelectObject", 417}, // §四-5 外服私服 m431 应答
		{"NpcType", 1566},         // 16-prop 方言 NpcType 行（fork 末现位 1559 已退役）
		{"IDCard", 1602},          // §13.5 外服私服 NPC 0x0D
	}
	for _, a := range playerAnchors {
		entry, ok := PlayerProperty(a.name)
		if !ok {
			t.Fatalf("player property %q absent", a.name)
		}
		if int(entry.Index) != a.index {
			t.Errorf("player %q index = %d, want %d", a.name, entry.Index, a.index)
		}
	}
	// fork 头块缺席名（PosiX/Orient 等）经表尾追复活：场景域首现=官方
	// 2433 行之后的尾位（官方行零改动，fork 客户端渲染头行通道）。
	for name, want := range map[string]int{"PosiX": 2434, "PosiY": 2435, "PosiZ": 2436, "Orient": 2437} {
		entry, ok := SceneProperty(name)
		if !ok || int(entry.Index) != want {
			t.Errorf("tail scene name %q = %+v, want idx %d", name, entry, want)
		}
	}
}

// TestOfficialPropertyTableFrameReassembly 按表重组 0x09 帧体:
// [0x09][u16 count](name\0 u8 type)*,与导出时的捕获帧逐字节一致性的 Go 侧回归。
func TestOfficialPropertyTableFrameReassembly(t *testing.T) {
	entries := OfficialPropertyEntries()
	frame := make([]byte, 3, 64*1024)
	frame[0] = 0x09
	binary.LittleEndian.PutUint16(frame[1:], uint16(len(entries)))
	for _, entry := range entries {
		frame = append(frame, entry.Name...)
		frame = append(frame, 0, byte(entry.TypeCode))
	}
	if count := binary.LittleEndian.Uint16(frame[1:]); int(count) != len(entries) {
		t.Fatalf("frame count %d != entries %d", count, len(entries))
	}
	// 逐行回读:重组体必须能精确还原每一行的 (name, type)。
	offset := 3
	for i, entry := range entries {
		end := bytes.IndexByte(frame[offset:], 0)
		if end < 0 {
			t.Fatalf("entry %d: no NUL terminator", i)
		}
		if string(frame[offset:offset+end]) != entry.Name || frame[offset+end+1] != byte(entry.TypeCode) {
			t.Fatalf("entry %d roundtrip mismatch", i)
		}
		offset = offset + end + 2
	}
	if offset != len(frame) {
		t.Fatalf("reassembly consumed %d of %d bytes", offset, len(frame))
	}
}
