package rolestate

import (
	"slices"
	"strings"
	"testing"
)

// TestStarterShortcutPosSeedsShortcutBar 建号快捷栏基线：StarterSet 的
// ShortcutPos（1 基栏位序号）→ shortcut_rec 行（索引 = 位置 - 1）。官服
// 新号登录 0x29 批同构（建号.pcapng 42.430s 等）：栏位 0..3 = 普攻 +
// 三式江湖武学；没写位置的 hw_normal_fd 不上栏。
func TestStarterShortcutPosSeedsShortcutBar(t *testing.T) {
	state, err := BuildRoleState(DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	want := []ShortcutRow{
		{Index: 0, Kind: "skill", ID: "zs_normal_00"},
		{Index: 1, Kind: "skill", ID: "CS_jh_cqgf01"},
		{Index: 2, Kind: "skill", ID: "CS_jh_cqgf04"},
		{Index: 3, Kind: "skill", ID: "CS_jh_cqgf06"},
	}
	if !slices.Equal(state.Shortcuts, want) {
		t.Fatalf("建号快捷栏基线 = %+v，want %+v", state.Shortcuts, want)
	}
	if state.ShortcutsSet {
		t.Fatal("建号基线不得标记已编辑——Set 留给玩家首次编辑")
	}
	// 建号基线不合成打坐条目（再战江湖跳线才授予，fssoloworld guide.go）。
	for _, row := range state.Shortcuts {
		if row.ID == "zs_default_01" || row.ID == "hw_normal_fd" {
			t.Fatalf("打坐条目/无位置的武学不得上栏： %+v", state.Shortcuts)
		}
	}

	// JSON 往返：预置行随 role_state_json。
	raw, err := MarshalRoleState(state)
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalRoleState(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.Shortcuts, want) || back.ShortcutsSet {
		t.Fatalf("回读不一致： %+v set=%t", back.Shortcuts, back.ShortcutsSet)
	}
	// 首次编辑物化后以档为准：删掉的预置行不复活（无合成条目可复活）。
	back.ShortcutsSet = true
	back.Shortcuts = slices.DeleteFunc(back.Shortcuts, func(row ShortcutRow) bool {
		return row.Index == 0
	})
	if len(back.Shortcuts) != 3 || back.Shortcuts[0].ID != "CS_jh_cqgf01" {
		t.Fatalf("已编辑基线应原样生效： %+v", back.Shortcuts)
	}
}

// TestParseShortcutPos ShortcutPos 口径：1 基栏位序号（索引 = 位置 - 1；
// 下方数字栏每页 10 格、上方 Ctrl 栏从 201 起），空/缺省不上栏，其余坏值
// 报错。
func TestParseShortcutPos(t *testing.T) {
	cases := []struct {
		raw   string
		index int32
		ok    bool
		bad   bool
	}{
		{"", 0, false, false},
		{"   ", 0, false, false},
		{"1", 0, true, false},
		{" 4 ", 3, true, false},
		{"10", 9, true, false},
		{"11", 10, true, false},
		{"201", 200, true, false},
		{"4097", shortcutIndexMax, true, false},
		{"0", 0, false, true},
		{"-1", 0, false, true},
		{"4098", 0, false, true},
		{"x", 0, false, true},
		{"1.0", 0, false, true},
	}
	for _, c := range cases {
		index, ok, err := parseShortcutPos("CS_x", c.raw)
		switch {
		case c.bad:
			if err == nil || ok {
				t.Fatalf("ShortcutPos %q 应报错： index=%d ok=%t err=%v", c.raw, index, ok, err)
			}
			if !strings.Contains(err.Error(), "CS_x") {
				t.Fatalf("报错应点名武学： %v", err)
			}
		case err != nil || index != c.index || ok != c.ok:
			t.Fatalf("ShortcutPos %q = (%d, %t, %v)，want (%d, %t)", c.raw, index, ok, err, c.index, c.ok)
		}
	}
}

// TestStarterShortcutRowsOrderAndConflicts 预置行按栏位升序（与表内书写
// 顺序无关）；全无位置不产行（omitempty 不落 JSON）；两个武学写同一位置
// 即资产自相矛盾，报错点名两者。
func TestStarterShortcutRowsOrderAndConflicts(t *testing.T) {
	rows, err := starterShortcutRows([]starterSkillXML{
		{ConfigID: "c", ShortcutPos: "3"},
		{ConfigID: "none"},
		{ConfigID: "a", ShortcutPos: "1"},
		{ConfigID: "b", ShortcutPos: "2"},
		{ConfigID: "blank", ShortcutPos: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []ShortcutRow{
		{Index: 0, Kind: "skill", ID: "a"},
		{Index: 1, Kind: "skill", ID: "b"},
		{Index: 2, Kind: "skill", ID: "c"},
	}
	if !slices.Equal(rows, want) {
		t.Fatalf("rows = %+v，want %+v", rows, want)
	}
	if rows, err := starterShortcutRows([]starterSkillXML{{ConfigID: "none"}}); err != nil || rows != nil {
		t.Fatalf("全无位置应不产行： %+v %v", rows, err)
	}
	_, err = starterShortcutRows([]starterSkillXML{
		{ConfigID: "a", ShortcutPos: "2"},
		{ConfigID: "b", ShortcutPos: "2"},
	})
	if err == nil || !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
		t.Fatalf("位置重复应报错并点名两者： %v", err)
	}
}

// TestBuildRoleStateRejectsBadShortcutPos 资产里的坏位置/重复位置：建号
// 报错（建号即败好过静默不上栏）。
func TestBuildRoleStateRejectsBadShortcutPos(t *testing.T) {
	saved := roleStateAssets
	t.Cleanup(func() { roleStateAssets = saved })
	cases := []struct {
		rows []starterSkillXML
		want string
	}{
		{[]starterSkillXML{{ConfigID: "a", ShortcutPos: "first"}}, `ShortcutPos "first"`},
		{[]starterSkillXML{{ConfigID: "a", ShortcutPos: "2"}, {ConfigID: "b", ShortcutPos: "2"}}, "重复"},
	}
	for _, c := range cases {
		injected := saved
		injected.starter.Skills.Skill = c.rows
		roleStateAssets = injected
		if _, err := BuildRoleState(DefaultBook, 1, "", 0); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("rows %+v 应报错含 %q： %v", c.rows, c.want, err)
		}
	}
}

// TestLegacyArchiveShortcutsUntouched 存量档（建号早于 ShortcutPos 预置）：
// 不回填——快捷栏照旧为空（打坐条目不再无条件合成）。
func TestLegacyArchiveShortcutsUntouched(t *testing.T) {
	legacy, err := UnmarshalRoleState(`{"sex":1,"scene":"s","resource":"r","position":[0,0,0,1]}`)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Shortcuts != nil || legacy.ShortcutsSet {
		t.Fatalf("旧档不应凭空出现快捷栏行： %+v set=%t", legacy.Shortcuts, legacy.ShortcutsSet)
	}
}
