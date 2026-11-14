package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestPlayerMotionPropsMatchActiveTable 运动属性族表一致闸：
// baseMotion/qinggongMotion 全部条目的索引与值线型必须与当前表（唯一
// official2433 方言）的玩家域行一致。2026-11-10 真机教训：fork 字面量
// 索引在换表方言下整体错位——玩家 0x0D 166 属性行在 CurNeiGong(655)
// 处按 string 读 f32 流即失步，客户端丢弃/截断玩家档案（武学面板与
// NPC 可见性连坐）——索引一律按名解析的回归锚。
func TestPlayerMotionPropsMatchActiveTable(t *testing.T) {
	props := append(baseMotionProperties(playerMotion{}), qinggongMotionProperties()...)
	if len(props) < 20 {
		t.Fatalf("motion props=%d, want >=20", len(props))
	}
	for _, p := range props {
		entry, ok := clientdata.PlayerProperty(p.Name)
		if !ok {
			t.Fatalf("%q absent from active table", p.Name)
		}
		if entry.Index != p.Index {
			t.Fatalf("%q index=%d, table=%d", p.Name, p.Index, entry.Index)
		}
		if entry.Type != p.Value.Type {
			t.Fatalf("%q wire=%v, table=%v", p.Name, p.Value.Type, entry.Type)
		}
	}
}
