package fssoloworld

// taskhint_test.go 新手引导提示 m371 的下发位次（官服「初始账号登陆-完整
// 引导-女」帧序；提示表见 taskhint.go）。帧号是该会话 S2C 流内 0 基序号。

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// hintLines 帧摘要：引导提示（m371）、发书/学会通知（m1 9645/9491）、步骤收尾
// （m24 task_ok、m8）保留，记录格写（0x13）连续折成一行「13×N」，其余不计。
func hintLines(t *testing.T, frames [][]byte) []string {
	t.Helper()
	var out []string
	run := 0
	flush := func() {
		if run > 0 {
			out = append(out, "13×"+strconv.Itoa(run))
			run = 0
		}
	}
	for _, f := range frames {
		if f[0] == 0x13 {
			run++
			continue
		}
		if f[0] != 0x1E {
			continue
		}
		line := taskMessageLine(t, f)
		id := line
		if sp := strings.IndexByte(line, ' '); sp >= 0 {
			id = line[:sp]
		}
		keep := id == "m371" || id == "m8" ||
			(id == "m24" && strings.Contains(line, "task_ok")) ||
			(id == "m1" && (strings.Contains(line, "9645") || strings.Contains(line, "9491")))
		if !keep {
			continue
		}
		flush()
		out = append(out, line)
	}
	flush()
	return out
}

// onlyHints 只留引导提示行。
func onlyHints(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, "m371 ") {
			out = append(out, line)
		}
	}
	return out
}

var taskOKLine = "m24 " + taskTestPlayer + ` 40 "task_ok"`

// TestTaskHintsFollowOfficialGuideScript 引导链 1340→1349 全程的 m371：每条
// 提示的类型、定位参数与在步骤簇里的位置逐项对拍官服。
func TestTaskHintsFollowOfficialGuideScript(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席")
	}
	taskTestNoTimers(t)
	pl := newTaskTestPlayer(t, 23)
	blankTasks(pl)
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcMu.Unlock()
	frames := captureFrames(t)

	// stepDone 走完当前步，返回本次应答的摘要。
	stepDone := func(taskID int) []string {
		t.Helper()
		*frames = nil
		done := false
		s.npcTaskLocked(pl, func() {
			task, track, _, ok := npcTaskCurrent(catalog, pl, uint32(taskID))
			if !ok {
				return
			}
			done = s.npcTaskStepDone(pl, uint32(taskID), npcTaskDialogTail(entergen.Identity{}, task, track.Step-1))
		})
		if !done {
			t.Fatalf("%d 当前步未推进", taskID)
		}
		return hintLines(t, *frames)
	}
	complete := func(taskID int) []string {
		t.Helper()
		*frames = nil
		if !s.completeNPCTask(pl, taskID, npcTaskCompleteTail{closeTalk: true}) {
			t.Fatalf("%d 提交未受理", taskID)
		}
		return hintLines(t, *frames)
	}

	// 1340 剧情（末步自动提交）：全程无提示（官服到 #2367 才出第一条）。
	s.acceptNPCTask(pl, 1340, "进场")
	if got := onlyHints(stepDone(1340)); len(got) != 0 || !pl.taskAcceptedContains(1347) {
		t.Fatalf("1340 不应有提示且应链推 1347: %v accepted=%+v", got, pl.roleState.TaskAccepted)
	}

	// 1347：三本书 + 末段对话。官服 #2362..#2377：入包帧 → m1 9645 →
	// 记录格×4 → m371 → NPC 标记 → task_ok → m8。
	for _, tc := range []struct{ step, book string }{
		{"558", "tbook_CS_jh_cqgf03"}, {"559", "tbook_CS_jh_cqgf02"}, {"560", "tbook_CS_jh_cqgf05"},
	} {
		assertLines(t, stepDone(1347), []string{
			`m1 1 "9645" "` + tc.book + `" 1`,
			"13×4",
			`m371 "wuxue_help" "` + tc.book + `"`,
			taskOKLine,
			"m8",
		}, "1347 步骤 "+tc.step+" 发书提示")
	}
	assertLines(t, stepDone(1347), []string{
		"13×4",
		`m371 "voice_xushijia_help"`,
		taskOKLine,
		"m8",
	}, "1347 步骤 561 对话提示（官服 #2733..#2748）")
	complete(1347)

	// 1341：两只占位，无提示；1342：562（发放前后各一条）→ 特殊步 → 剧情
	// 484 → 570 → 139。
	s.npcTaskStubTick(pl, 1341)
	s.npcTaskStubTick(pl, 1341)
	if !pl.taskAcceptedContains(1342) {
		t.Fatalf("1341 完成后应链推 1342: %+v", pl.roleState.TaskAccepted)
	}
	first := stepDone(1342) // 562 ＋ 特殊步到步即过
	assertLines(t, onlyHints(first[:indexOf(first, taskOKLine)+1]), []string{
		`m371 "add_item_yu_helper"`, // 官服 #5080：早于 m1 9645
		`m371 "jh_tl_dztx_book_help" "book_zs_default_01"`,
	}, "1342 步骤 562 发放前后提示（官服 #5080..#5086）")
	if at := indexOf(first, `m371 "add_item_yu_helper"`); at > indexOf(first, `m1 1 "9645" "book_zs_default_01" 1`) {
		t.Fatalf("add_item_yu_helper 应早于发书通知: %v", first)
	}
	var rest []string
	rest = append(rest, stepDone(1342)...)
	rest = append(rest, stepDone(1342)...)
	rest = append(rest, stepDone(1342)...)
	assertLines(t, onlyHints(rest), []string{
		`m371 "anqi_help_jq02_1" "book_hw_normal"`, // 官服 #5315
		`m371 "anqi_help_jq02_3" "hw_fb_rw_01003"`, // 官服 #5516
	}, "1342 步骤 570/139 发书提示")

	// 1342 提交 → 1343 接取：完成簇（碎银/声望/历练…）之后，提示是接取簇
	// 的第一帧，早于全部记录行与接取通知（官服 #5601 op=29 批 → #5602
	// m371 → … → #5618 m209 9 1343 → #5620 m1 9628）。
	complete(1342)
	completeAt, hintAt, acceptAt := -1, -1, -1
	for i, frame := range *frames {
		switch {
		case bytes.Contains(frame, []byte("title_1342")) && completeAt < 0:
			completeAt = i
		case bytes.Contains(frame, []byte("anqi_help_jq01_4")) && hintAt < 0:
			hintAt = i
		case bytes.Contains(frame, []byte("title_1343")) && acceptAt < 0:
			acceptAt = i
		}
	}
	// 官服在完成簇与接取簇之间还有一个 op=29 记录批（本服不发）：提示紧邻
	// 完成通知，早于接取簇的其余全部帧。
	if completeAt < 0 || hintAt != completeAt+1 || acceptAt <= hintAt {
		t.Fatalf("1343 接取提示位次不对（完成通知 → m371 → 接取簇）: complete=%d hint=%d accept=%d", completeAt, hintAt, acceptAt)
	}
	if !pl.taskAcceptedContains(1343) {
		t.Fatalf("1342 完成后应链推 1343: %+v", pl.roleState.TaskAccepted)
	}

	// 1343：杀怪×3 → 575（发 useitem_museborn_2，无提示）→ 杀怪 → 末步
	// 自动提交 → 1344。
	for i := 0; i < 3; i++ {
		s.npcTaskStubTick(pl, 1343)
	}
	if got := onlyHints(stepDone(1343)); len(got) != 0 {
		t.Fatalf("1343 步骤 575 不应有提示: %v", got)
	}
	s.npcTaskStubTick(pl, 1343)
	s.npcTaskStubTick(pl, 1343)
	if !pl.taskAcceptedContains(1344) {
		t.Fatalf("1343 完成后应链推 1344: %+v", pl.roleState.TaskAccepted)
	}

	// 1344：564 发轻功书 → 565 → 576 → 566 发内功书 → 567。
	assertLines(t, onlyHints(stepDone(1344)), []string{`m371 "yd_qinggong_help" "book_qinggong_4"`}, "1344 步骤 564（官服 #7561）")
	if got := onlyHints(stepDone(1344)); len(got) != 0 {
		t.Fatalf("1344 步骤 565 不应有提示: %v", got)
	}
	if got := onlyHints(stepDone(1344)); len(got) != 0 {
		t.Fatalf("1344 步骤 576 不应有提示: %v", got)
	}
	assertLines(t, onlyHints(stepDone(1344)), []string{`m371 "miji_help" "ng_tbook_jh_001"`}, "1344 步骤 566（官服 #8086）")
	assertLines(t, onlyHints(stepDone(1344)), []string{`m371 "gong_help_xiu_1"`}, "1344 步骤 567（官服 #8822）")
	complete(1344)

	// 1348：270 → 特殊步到步即过自动提交 → 1349：269。
	assertLines(t, onlyHints(stepDone(1348)), []string{`m371 "jingmai_help_two"`}, "1348 步骤 270（官服 #9139）")
	if !pl.taskAcceptedContains(1349) {
		t.Fatalf("1348 应自动提交并链推 1349: %+v", pl.roleState.TaskAccepted)
	}
	assertLines(t, onlyHints(stepDone(1349)), []string{`m371 "jingmai_help_one"`}, "1349 步骤 269（官服 #9378）")
}

func indexOf(lines []string, want string) int {
	for i, line := range lines {
		if line == want {
			return i
		}
	}
	return -1
}

// TestTaskHintsDisabledByEnv NINEYIN_TASK_HINTS=off：发书步骤只剩 m1 9645，
// 不出 m371。
func TestTaskHintsDisabledByEnv(t *testing.T) {
	t.Setenv("NINEYIN_TASK_HINTS", "off")
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席")
	}
	taskTestNoTimers(t)
	pl := newTaskTestPlayer(t, 24)
	blankTasks(pl)
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcMu.Unlock()
	frames := captureFrames(t)
	s.acceptNPCTask(pl, 1340, "进场")
	s.npcTaskLocked(pl, func() {
		task, track, _, ok := npcTaskCurrent(catalog, pl, 1340)
		if ok {
			s.npcTaskStepDone(pl, 1340, npcTaskDialogTail(entergen.Identity{}, task, track.Step-1))
		}
	})
	*frames = nil
	s.npcTaskLocked(pl, func() {
		task, track, _, ok := npcTaskCurrent(catalog, pl, 1347)
		if ok {
			s.npcTaskStepDone(pl, 1347, npcTaskDialogTail(entergen.Identity{}, task, track.Step-1))
		}
	})
	assertLines(t, hintLines(t, *frames), []string{
		`m1 1 "9645" "tbook_CS_jh_cqgf03" 1`, "13×4", taskOKLine, "m8",
	}, "关停后的 1347 步骤 558")
}

// TestTaskHintForStepTable 查表：实际发放的物品优先于对话步表，都没有零值。
func TestTaskHintForStepTable(t *testing.T) {
	if h := taskHintForStep(taskdata.Step{Scene: "DialogID_561"}, []taskdata.PrizeItem{{ID: "tbook_CS_jh_cqgf03", Num: 1}}); h.helper != "wuxue_help" || h.param != "tbook_CS_jh_cqgf03" {
		t.Fatalf("发书提示 = %+v", h)
	}
	if h := taskHintForStep(taskdata.Step{Scene: "DialogID_561"}, nil); h.helper != "voice_xushijia_help" || h.param != "" {
		t.Fatalf("对话步提示 = %+v", h)
	}
	// 物品不在表里（如 1343 的 useitem_museborn_2）且对话号也不在表里：零值。
	if h := taskHintForStep(taskdata.Step{Scene: "DialogID_575"}, []taskdata.PrizeItem{{ID: "useitem_museborn_2", Num: 1}}); h.helper != "" {
		t.Fatalf("无表项应零值: %+v", h)
	}
	// 物品在表里但没发出去（目录缺席/背包满）：按对话步表，没有则零值。
	if h := taskHintForStep(taskdata.Step{Scene: "DialogID_558"}, nil); h.helper != "" {
		t.Fatalf("未发放不应出物品提示: %+v", h)
	}
}

// TestUseItemSkillBookGuideHint 引导链招式书学会后紧跟 m98 860 → m371
// skill_help（官服 #2447..#2452）；非「初出茅庐」角色学书不出提示。
func TestUseItemSkillBookGuideHint(t *testing.T) {
	for _, single := range []bool{true, false} {
		f := newItemUseFixture(t)
		st := &f.pl.roleState
		if single {
			st.GameStep = guideStepSingle
		}
		st.BagItems = []rolestate.BagItem{
			{Slot: 5, ConfigID: "tbook_CS_jh_cqgf03", ItemType: 23, ViewID: 1, Amount: 1, MaxAmount: 1, Seq: 1},
		}
		f.s.dispatchCustom(f.pl, useMsg(2, 5, 1))
		f.take()
		f.finish(t)
		tail := f.take()
		var ids []int32
		var hint []byte
		for _, frame := range tail {
			if frame[0] != 0x1E {
				continue
			}
			msg, err := entergen.ParseCustomMessage(frame)
			if err != nil {
				t.Fatalf("解帧: %v", err)
			}
			ids = append(ids, msg.Values[0].Int32)
			if msg.Values[0].Int32 == 371 {
				hint = frame
			}
		}
		if single {
			// … m141 → m1 9491 → m98 860 → m371 →（扣件 0x19 不在自定义消息里）。
			if n := len(ids); n < 4 || ids[n-4] != 141 || ids[n-3] != 1 || ids[n-2] != 98 || ids[n-1] != 371 {
				t.Fatalf("学会簇尾部消息号应为 141→1→98→371: %v", ids)
			}
			want, err := entergen.TaskHintFrame("skill_help", "CS_jh_cqgf03")
			if err != nil || !bytes.Equal(hint, want) {
				t.Fatalf("skill_help 提示帧不符: % x | % x %v", hint, want, err)
			}
		} else if hint != nil {
			t.Fatalf("非初出茅庐角色学书不应出提示: % x", hint)
		}
	}
}
