package fssoloworld

// operateguide_test.go 操作模式引导 m704 的下发点（2026-11-28 真机反馈「移到
// 剧情演出后，选择再战江湖的也要发」；帧字节对拍官服原帧见
// entergen/operateguide_test.go）：
//
//	选择窗      不发（官服在 m427 之后那一拍发，本服挪后）
//	初出茅庐    新手副本里第一场任务影像（1340 接取影像 473，开场剧情演出）
//	            收尾簇之后发，每角色一次；影像关停时随接取直接发
//	再战江湖    切出副本落到出生地后的第一次就绪收尾发（该线选择后没有
//	            影像）；存量跳线号下次进场补一次

import (
	"bytes"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// operateGuideIndexes 下行帧里 m704 的下标序列（没发＝空）。
func operateGuideIndexes(frames [][]byte) []int {
	want, err := entergen.OperateModeGuideFrame()
	if err != nil {
		panic(err)
	}
	var out []int
	for i, f := range frames {
		if bytes.Equal(f, want) {
			out = append(out, i)
		}
	}
	return out
}

// trackFrameIndexes 下行帧里任务追踪栏刷新 m390 的下标序列。
func trackFrameIndexes(t *testing.T, frames [][]byte) []int {
	t.Helper()
	want, err := entergen.TaskTrackRefreshFrame()
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for i, f := range frames {
		if bytes.Equal(f, want) {
			out = append(out, i)
		}
	}
	return out
}

// singleGuidePlayer 已点选「初出茅庐」、仍在新手副本里的角色（阶段落定）。
func singleGuidePlayer(pl *Player) {
	pl.taskMu.Lock()
	pl.roleState.GameStep, pl.guidePending = guideStepSingle, false
	pl.taskMu.Unlock()
}

// TestOperateModeGuideNotAtChoiceWindow 待选态就绪收尾与点选后到影像开始之间
// 都不发：选择窗那一拍没有操作模式引导。
func TestOperateModeGuideNotAtChoiceWindow(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 52)
	s.readyTail(pl)
	assertLines(t, sink.lines(t), []string{"m346 2 0", "m427", "10 CanntOperator=1", "m390 5 0"}, "待选态就绪收尾")
	sink.reset()
	s.clientActivity(pl, guideC2SSingle)
	if !waitGuideTrack(pl, 1340) {
		t.Fatal("点选后应自动接取引导链根 1340")
	}
	// 接取簇已发完、影像还没开始（本用例影像延迟一小时）。
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("选择窗到剧情演出开始之间不应发 m704: %v", got)
	}
	if pl.roleState.OperateGuideShown {
		t.Fatal("剧情演出结束前不应置已发标记")
	}
}

// TestOperateModeGuideAfterOpeningMovie 初出茅庐线：开场剧情演出（1340 接取
// 影像 473）开始时与演出期间都不发；客户端上报结束 m806 [473] 后，收尾簇
// （标志复位两帧 → m107）之后发一帧 m704，置已发标记；之后的影像结束不再发。
func TestOperateModeGuideAfterOpeningMovie(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 53)
	singleGuidePlayer(pl)
	if !s.acceptNPCTask(pl, 1340, "进场") {
		t.Fatal("1340 接取失败")
	}
	sink.reset()
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("影像开始时不应发 m704（要等演出结束）: %v", got)
	}
	// 演出期间来一次就绪收尾（重登／切图）：不发。
	sink.reset()
	s.readyTail(pl)
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("演出期间的就绪收尾不应发 m704: %v", got)
	}
	// 客户端上报影像结束。
	sink.reset()
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	assertLines(t, sink.lines(t), []string{
		"10 CantBeAttack=0",
		"10 CantMove=0",
		"m107",
		"m704",
	}, "影像收尾簇")
	if !pl.roleState.OperateGuideShown {
		t.Fatal("开场剧情演出结束后应置已发标记")
	}
	// 每角色一次：下一场影像（1341 的 474）结束时不再发。
	sink.reset()
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, taskdata.Task{ID: 1341, AcceptMovie: 474}) })
	s.handleMovieMsg(pl, customMsg(806, customInt(474)), 806)
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("已发过的角色不应再发 m704: %v", got)
	}
	assertLines(t, sink.lines(t), []string{"m8", "m104 obj(0/0) 474 4", "10 CantBeAttack=0", "10 CantMove=0", "m107"}, "第二场影像")
}

// TestOperateModeGuideMovieEndByTimeout 客户端不上报结束、超时自动收尾：同样
// 算演出结束，收尾簇之后发。
func TestOperateModeGuideMovieEndByTimeout(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 54)
	singleGuidePlayer(pl)
	sink.reset()
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	s.npcTaskLocked(pl, func() { s.npcTaskMovieEnd(pl, "测试") })
	assertLines(t, sink.lines(t), []string{
		"m8", "m104 obj(0/0) 473 4",
		"10 CantBeAttack=0",
		"10 CantMove=0",
		"m107",
		"m704",
	}, "超时收尾")
}

// TestOperateModeGuideMovieDisabled 影像关停（NINEYIN_TASK_MOVIE=off）＝没有
// 剧情演出可等：接取簇之后直接发。
func TestOperateModeGuideMovieDisabled(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 55)
	t.Setenv("NINEYIN_TASK_MOVIE", "off")
	singleGuidePlayer(pl)
	sink.reset()
	if !s.acceptNPCTask(pl, 1340, "进场") {
		t.Fatal("1340 接取失败")
	}
	lines := sink.lines(t)
	if len(lines) == 0 || lines[len(lines)-1] != "m704" {
		t.Fatalf("影像关停时接取簇之后应直接发 m704:\n%s", lines)
	}
	if !pl.roleState.OperateGuideShown {
		t.Fatal("应置已发标记")
	}
}

// TestOperateModeGuideOutsideGuideLine 引导线未定（待选态）、或初出茅庐线但
// 已不在新手副本：影像结束不发。
func TestOperateModeGuideOutsideGuideLine(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 56)
	sink.reset()
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	assertLines(t, sink.lines(t), []string{"m8", "m104 obj(0/0) 473 4", "10 CantBeAttack=0", "10 CantMove=0", "m107"}, "待选态影像收尾")
	sink.reset()
	singleGuidePlayer(pl)
	pinSceneBorn03(t, pl)
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("副本外的影像结束不应发 m704: %v", got)
	}
	if pl.roleState.OperateGuideShown {
		t.Fatal("不应置已发标记")
	}
	// 影像关停时的直接发同样只认初出茅庐线：待选态（阶段未定）与副本外的
	// 角色接带影像的任务都不发。
	t.Setenv("NINEYIN_TASK_MOVIE", "off")
	for _, step := range []int{guideStepNone, guideStepSingle} {
		pl.taskMu.Lock()
		pl.roleState.GameStep = step
		pl.taskMu.Unlock()
		blankTasks(pl)
		sink.reset()
		if !s.acceptNPCTask(pl, 1340, "进场") {
			t.Fatalf("1340 接取失败（step=%d）", step)
		}
		if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
			t.Fatalf("副本外／待选态在影像关停时不应发 m704（step=%d）: %v", step, got)
		}
	}
	if pl.roleState.OperateGuideShown {
		t.Fatal("不应置已发标记")
	}
}

// TestOperateModeGuideSkipLineOnArrival 再战江湖：跳过簇里不发（紧接着 0x0B
// 切出副本），落到出生地后的第一次就绪收尾发（m704 → m390），每角色一次。
func TestOperateModeGuideSkipLineOnArrival(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 57)
	guideSwitchGapSoon(t)
	s.readyTail(pl)
	sink.reset()
	s.clientActivity(pl, guideC2SSkip)
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("跳过簇里不应发 m704（随后 0x0B 切场景会重建窗体）: %v", got)
	}
	if pl.roleState.OperateGuideShown {
		t.Fatal("落位前不应置已发标记")
	}
	guideWaitSwitch(t, s, pl)
	sink.reset()
	s.clientReady(pl)
	if guideSceneOf(pl) {
		t.Fatalf("应已切出新手副本: %s", pl.roleState.Resource)
	}
	frames := sink.framesCopy()
	got, tracks := operateGuideIndexes(frames), trackFrameIndexes(t, frames)
	if len(got) != 1 || len(tracks) != 1 || !(got[0] < tracks[0]) {
		t.Fatalf("落位就绪收尾应恰发一帧 m704 且在 m390 之前: m704=%v m390=%v", got, tracks)
	}
	if !pl.roleState.OperateGuideShown {
		t.Fatal("落位后应置已发标记")
	}
	// 每角色一次：之后的就绪收尾不发（追踪列直写行照旧）。
	sink.reset()
	s.readyTail(pl)
	if got := operateGuideIndexes(sink.framesCopy()); len(got) != 0 {
		t.Fatalf("已发过的再战江湖号不应再发 m704: %v", got)
	}
	if tracks := trackFrameIndexes(t, sink.framesCopy()); len(tracks) != 1 {
		t.Fatalf("就绪收尾应照发一帧 m390: %v", tracks)
	}
}

// TestOperateModeGuideStockSkipCharacter 选过再战江湖、还没看过提示的存量号：
// 下次进场的就绪收尾补发一次（选择再战江湖的也要发）。
func TestOperateModeGuideStockSkipCharacter(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 58)
	pinSceneBorn03(t, pl)
	pl.taskMu.Lock()
	pl.roleState.GameStep, pl.guidePending = guideStepSkip, false
	pl.taskMu.Unlock()
	s.readyTail(pl)
	assertLines(t, sink.lines(t), []string{"m704", "m390 5 0"}, "存量再战江湖号")
	sink.reset()
	s.readyTail(pl)
	assertLines(t, sink.lines(t), []string{"m390 5 0"}, "补发过一次之后")
}
