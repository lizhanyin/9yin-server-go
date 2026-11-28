package entergen

// 建号快捷栏基线的下发单测：StarterSet 的 ShortcutPos → rolestate.
// Shortcuts → shortcut_rec 0x11 行（就绪簇与切图重入同一张表）。

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// TestReadyFramesCarryStarterShortcutRows 就绪簇的 shortcut_rec 行 =
// StarterSet 预置行（栏位索引 = ShortcutPos - 1、kind=skill、id=ConfigID）
// ；行帧挂玩家对象、行号恒 0（追加语义）。打坐条目不合成（再战江湖
// 跳线才授予，fssoloworld guide.go）。
func TestReadyFramesCarryStarterShortcutRows(t *testing.T) {
	ctx := testContext(t)
	want := []rolestate.ShortcutRow{
		{Index: 0, Kind: "skill", ID: "zs_normal_00"},
		{Index: 1, Kind: "skill", ID: "CS_jh_cqgf01"},
		{Index: 2, Kind: "skill", ID: "CS_jh_cqgf04"},
		{Index: 3, Kind: "skill", ID: "CS_jh_cqgf06"},
	}
	ready, err := ReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := shortcutRecRows(ready); !slices.Equal(got, want) {
		t.Fatalf("就绪簇 shortcut_rec 行 = %+v，want %+v", got, want)
	}
	// 切图重入（0x0C 清对象后 PlayerObjectConfigFrames 重推）同一张表。
	reentry, err := PlayerObjectConfigFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := shortcutRecRows(reentry); !slices.Equal(got, want) {
		t.Fatalf("重入簇 shortcut_rec 行 = %+v，want %+v", got, want)
	}
	record := clientdata.MustRecordIndexByName("shortcut_rec")
	for _, frame := range ready {
		if frame[0] != 0x11 || binary.LittleEndian.Uint16(frame[10:]) != record {
			continue
		}
		if binary.LittleEndian.Uint32(frame[2:]) != ctx.Identity.ObjectID ||
			binary.LittleEndian.Uint32(frame[6:]) != ctx.Identity.OwnerID ||
			binary.LittleEndian.Uint16(frame[12:]) != 0 || binary.LittleEndian.Uint16(frame[14:]) != 1 {
			t.Fatalf("shortcut_rec 行帧头异常: %x", frame[:16])
		}
	}

	// 玩家编辑物化后的档：以档为准，只重放档内行（打坐条目删了不复活）。
	ctx.State.Shortcuts = []rolestate.ShortcutRow{{Index: 0, Kind: "item", ID: "it_potion"}}
	ctx.State.ShortcutsSet = true
	edited, err := PlayerObjectConfigFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := shortcutRecRows(edited); !slices.Equal(got, ctx.State.Shortcuts) {
		t.Fatalf("已编辑档 shortcut_rec 行 = %+v，want %+v", got, ctx.State.Shortcuts)
	}
}

// shortcutRecRows 抽出帧序里的 0x11 shortcut_rec 行（栏位索引/kind/id）。
func shortcutRecRows(frames [][]byte) []rolestate.ShortcutRow {
	record := clientdata.MustRecordIndexByName("shortcut_rec")
	var rows []rolestate.ShortcutRow
	for _, frame := range frames {
		if frame[0] != 0x11 || len(frame) < 16 || binary.LittleEndian.Uint16(frame[10:]) != record {
			continue
		}
		row := rolestate.ShortcutRow{Index: int32(binary.LittleEndian.Uint32(frame[16:]))}
		kind, off := recordStringCell(frame, 20)
		id, _ := recordStringCell(frame, off)
		row.Kind, row.ID = kind, id
		rows = append(rows, row)
	}
	return rows
}

// recordStringCell 记录行里的窄串格（u32 长度含尾 NUL）→ 值与下一格偏移。
func recordStringCell(frame []byte, off int) (string, int) {
	size := int(binary.LittleEndian.Uint32(frame[off:]))
	return string(bytes.TrimRight(frame[off+4:off+4+size], "\x00")), off + 4 + size
}
