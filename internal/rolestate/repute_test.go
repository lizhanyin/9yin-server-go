package rolestate

// repute_test.go 声望/历练累加与登录簇行推。

import (
	"testing"
)

func TestAddReputeAndLiveGroove(t *testing.T) {
	var s RoleState
	row, total, added := s.AddRepute("repute_jianghu", 10)
	if row != 0 || total != 10 || !added {
		t.Fatalf("首次声望应加行: row=%d total=%d added=%v", row, total, added)
	}
	row, total, added = s.AddRepute("repute_jianghu", 20)
	if row != 0 || total != 30 || added {
		t.Fatalf("已有声望应累加: row=%d total=%d added=%v", row, total, added)
	}
	row, total, added = s.AddRepute("repute_school", 5)
	if row != 1 || total != 5 || !added {
		t.Fatalf("另一种声望应另起一行: row=%d total=%d added=%v", row, total, added)
	}
	if total, ok := s.AddLiveGroove(5, 5000); !ok || total != 5000 {
		t.Fatalf("历练累加 = %d %v", total, ok)
	}
	if total, ok := s.AddLiveGroove(5, 10000); !ok || total != 15000 {
		t.Fatalf("历练再累加 = %d %v", total, ok)
	}
	if _, ok := s.AddLiveGroove(0, 1); ok {
		t.Fatal("下标 0 不应累加")
	}
	if _, ok := s.AddLiveGroove(6, 1); ok {
		t.Fatal("下标 6 不应累加")
	}
}

// TestAddLiveGrooveCapClamp 历练槽上限钳制（999 点=999000 毫；2433 线
// livepoint-conversion.md）。
func TestAddLiveGrooveCapClamp(t *testing.T) {
	s := &RoleState{}
	if total, ok := s.AddLiveGroove(1, 1_500_000); !ok || total != LiveGrooveCap {
		t.Fatalf("超限累加 = %d %v, want %d", total, ok, LiveGrooveCap)
	}
	if total, _ := s.AddLiveGroove(1, 500_000); total != LiveGrooveCap {
		t.Fatalf("已满槽再累加 = %d, want %d", total, LiveGrooveCap)
	}
	// 扣减穿底钳 0。
	if total, _ := s.AddLiveGroove(1, -2_000_000); total != 0 {
		t.Fatalf("负向扣减 = %d, want 0", total)
	}
}

// TestAddLivePoints 点历练按序填槽（×1000 毫、单次 1..999 点、满槽溢出
// 丢弃——2433 GM add_livepoint 同形）。
func TestAddLivePoints(t *testing.T) {
	s := &RoleState{LiveGrooves: [6]int32{0, 999_000, 400_000, 0, 0, 0}}
	// 槽 1 已满、槽 2 余 599 点：发 700 点 → 槽 2 填满、槽 3 进 101 点。
	if added := s.AddLivePoints(700); added != (599_000 + 101_000) {
		t.Fatalf("入槽毫数 = %d, want 700000", added)
	}
	if s.LiveGrooves[2] != LiveGrooveCap || s.LiveGrooves[3] != 101_000 {
		t.Fatalf("落槽 = %d/%d, want 999000/101000", s.LiveGrooves[2], s.LiveGrooves[3])
	}
	// 单次上限 999 点。
	s2 := &RoleState{}
	s2.AddLivePoints(5000)
	if s2.LiveGrooves[1] != 999_000 {
		t.Fatalf("单次超 999 点应钳: %d", s2.LiveGrooves[1])
	}
}
