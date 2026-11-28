package entergen

// guidematerial_test.go 0x0D 之后补推的引导属性（HelpStepID 等）：与进场簇
// GuideEntryFrames 同值——官服 CN 2026 的 0x0D 属性区本身就带这三项（完整
// 引导-女 #487 的 0x28 批内层，2026-11-28 用客户端真 LZ 解压器解出：
// HelpStepID=chengdu_00、FinishFreshmanRecStr 初值、BornServerGameStep=45）。

import (
	"bytes"
	"testing"
	"time"
)

// TestGuideMaterializeFramesMirrorEntryCluster 待选新号：三帧与官服 0x0D 属性区
// 里的同属性同形（HelpStepID → FinishFreshmanRecStr → BornServerGameStep）。
func TestGuideMaterializeFramesMirrorEntryCluster(t *testing.T) {
	ctx := testContext(t)
	ctx.Identity = officialPlayer
	ctx.GuideStep = 4
	ctx.HelpStep = "chengdu_00"
	ctx.GuideChoicePending = true
	got, err := GuideMaterializeFrames(ctx)
	if err != nil {
		t.Fatalf("GuideMaterializeFrames: %v", err)
	}
	assertOfficialFrames(t, got[:1], err, []string{
		"10 00 f3 00 00 35 2a 10 00 00 01 00 81 00 0b 00 00 00 63 68 65 6e 67 64 75 5f 30 30 00", // HelpStepID=chengdu_00
	}, "0x0D 后补推的 HelpStepID")
	// FinishFreshmanRecStr 初值：官服 #132 原形（65B：头 18B + 45×0x01 + 09 00）。
	rec := got[1]
	if len(rec) != 65 || !bytes.Equal(rec[:18], spacedHex(t,
		"10 00 f3 00 00 35 2a 10 00 00 01 00 e1 00 2f 00 00 00")) ||
		!bytes.Equal(rec[18:63], bytes.Repeat([]byte{1}, 45)) || !bytes.Equal(rec[63:], []byte{9, 0}) {
		t.Fatalf("FinishFreshmanRecStr 初值帧不符: % x", rec)
	}
	assertOfficialFrames(t, got[2:], err, []string{
		"10 00 f3 00 00 35 2a 10 00 00 01 00 1d 05 2d 00 00 00", // BornServerGameStep=45
	}, "0x0D 后补推的 BornServerGameStep")
}

// TestGuideMaterializeFramesNoneOutsideGuide 不在引导流的角色零帧；已定档
// 非待选号不发 FinishFreshmanRecStr（值随引导进度变化，本服只复现新号初值）。
func TestGuideMaterializeFramesNoneOutsideGuide(t *testing.T) {
	ctx := testContext(t)
	if got, err := GuideMaterializeFrames(ctx); err != nil || len(got) != 0 {
		t.Fatalf("引导流外应零帧, got %d, %v", len(got), err)
	}
	ctx.GuideStep = 5
	ctx.HelpStep = "jimingyi_00"
	got, err := GuideMaterializeFrames(ctx)
	assertOfficialFrames(t, got, err, []string{
		"10 00 01 00 00 00 01 00 00 00 01 00 81 00 0c 00 00 00 6a 69 6d 69 6e 67 79 69 5f 30 30 00",
		"10 00 01 00 00 00 01 00 00 00 01 00 1d 05 37 00 00 00",
	}, "已定档号补推的引导属性")
}

// TestPreReadyFramesCarryGuidePropertiesAfterMaterialize 进场序列里 HelpStepID
// 的 0x10 必须出现在玩家物化帧 0x0D 之后（场内对象只认物化后的属性）。
func TestPreReadyFramesCarryGuidePropertiesAfterMaterialize(t *testing.T) {
	ctx := testContext(t)
	ctx.GuideStep = 4
	ctx.HelpStep = "yanyuzhuang_00"
	frames, err := PreReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatalf("PreReadyFrames: %v", err)
	}
	want, err := HelpStepFrame(ctx.Identity, ctx.HelpStep)
	if err != nil {
		t.Fatalf("HelpStepFrame: %v", err)
	}
	materialized := -1
	for i, frame := range frames {
		if len(frame) > 0 && frame[0] == 0x0D {
			materialized = i
		}
	}
	if materialized < 0 {
		t.Fatal("进场序列缺 0x0D 玩家物化帧")
	}
	found := false
	for _, frame := range frames[materialized+1:] {
		found = found || bytes.Equal(frame, want)
	}
	if !found {
		t.Fatalf("0x0D（第 %d 帧）之后没有 HelpStepID=%q 的 0x10", materialized, ctx.HelpStep)
	}
}
