package entergen

// guide.go 新手引导流的帧编码（官服女号「完整引导」/ 男号「跳过引导」两份
// 抓包逐帧原字节定版；流程编排见 fssoloworld/guide.go）：
//
//	引导记录    0x10 FinishFreshmanRecStr=初值（idx225；待选态进场簇首位，
//	            建号.pcapng 男号 #134 照抄）——引擎侧只当新手进度条的
//	            数据源，不是弹窗条件
//	黑幕窗      m346 [2, 0] → m427 []（on_query_fresh_man：客户端 lua 弹
//	            「初出茅庐/再战江湖」选择窗 form_single_blackback）→ 锁输入；
//	            官服五个弹窗会话恒此序，427 只在待选号上发、每场一帧
//	追踪栏刷新  m390 [5, 0]（女号 #1508 / 男号 #1604、#2258）：引擎原生
//	            handler（lua 的 custom_refresh_track_info 是 m52 [1, 服务器
//	            时间]）——追踪栏窗体进场预载后是隐藏的，官服在这一帧之后
//	            才显示并按已接表追踪标志重建追踪列表。
//	引导阶段    m414 ["gamestep", 主步, 子步]（#437 女 4/5、#547 男 5/5）
//	            ＋ 0x10 BornServerGameStep=主步×10+子步（#183=45、#293=55）
//	新手帮助    0x10 HelpStepID=<步骤>（#466 "chengdu_00"、#1792 "chengdu_02"）
//	输入锁      0x10 CanntOperator=1/0（待选态 #881 锁、点选后 #1532 解）
//	跳过簇      （男号 #1612..#1989，再战江湖）GiveItemRec 行 → 历练四帧
//	            （m1 8055 → m24 [玩家,1,点数] → m24 [玩家,40,"huodelilian"]
//	            → 0x10 LiveGroove_N）→ Repute_Record 行（标志 2）→ m142 生活
//	            新手帮助 ×4 → 入江湖属性（PowerLevel/YinLevel=1、解锁）。

import (
	"bytes"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// 服务器自定义消息号＝客户端引擎 handler 表的下标（2026-11-27 逐号对齐：
// AOW 客户端脱壳镜像 .data 静态表 738 项、CN 2026 引擎 dump 堆表 765 项
// 同序同义——名字串为 lua 全局函数名，空名＋函数指针＝引擎原生）。
const (
	serverCustomTaskTrackRefresh int32 = 390 // 引擎原生 handler（虚拟化，语义未解）
	serverCustomGameStep         int32 = 414 // SwitchManager gamestep（引擎原生）
	serverCustomSingleMark       int32 = 543 // on_addbtn_lead_info_open：跳过簇里 1346 置位后的空参消息（#1683）
	serverCustomProcessSingle    int32 = 346 // process_single_msg [子型, 值]：2＝form_single_check.custom_single_set_form
	serverCustomQueryFreshMan    int32 = 427 // on_query_fresh_man：弹「初出茅庐/再战江湖」黑幕窗，零参
)

var recordGiveItem = clientdata.MustRecordIndexByName("GiveItemRec")

// TaskTrackRefreshFrame 任务追踪栏刷新 m390 [5, 0]：首参 >0 客户端即显示
// 追踪栏（官服两号恒 5），次参是服务器时间（官服恒 0）。
func TaskTrackRefreshFrame() ([]byte, error) {
	return CustomMessageFrame(serverCustomTaskTrackRefresh, CustomInt(5), CustomInt(0))
}

// GameStepFrame 引导阶段 m414 ["gamestep", 主步, 子步]。
func GameStepFrame(main, sub int32) ([]byte, error) {
	return CustomMessageFrame(serverCustomGameStep, CustomString("gamestep"), CustomInt(main), CustomInt(sub))
}

// guideStepSubOfficial 官服两抓包（女 #437 / 男建号.pcapng 42.43s）的
// m414 子步恒 5。
const guideStepSubOfficial int32 = 5

// GuideEntryFrames 进场簇的引导阶段帧（PreReady m655 后、0x0B 前）：
// 待选态先补 FinishFreshmanRecStr 初值（官服 #134 位，照抄的引导记录）
// → idx1309 BornServerGameStep=主步×10+子步 → m414 ["gamestep",
// 主步, 子步] → idx129 HelpStepID。GuideStep=0（从未进引导流）零帧；
// HelpStep 空时省略 idx129。弹窗帧（m346→m427）、锁输入与 m390 不在此处
// ——它们在就绪簇收尾（官服 49.62s、50.24s，均在 NPC 物化之后）。
func GuideEntryFrames(ctx Context) ([][]byte, error) {
	var builders []func() ([]byte, error)
	if ctx.GuideChoicePending {
		builders = append(builders, func() ([]byte, error) {
			return FreshmanRecFrame(ctx.Identity)
		})
	}
	if ctx.GuideStep == 0 {
		if len(builders) == 0 {
			return nil, nil
		}
	} else {
		builders = append(builders,
			func() ([]byte, error) {
				return BornGameStepFrame(ctx.Identity, int32(ctx.GuideStep)*10+guideStepSubOfficial)
			},
			func() ([]byte, error) { return GameStepFrame(int32(ctx.GuideStep), guideStepSubOfficial) },
		)
		if ctx.HelpStep != "" {
			builders = append(builders, func() ([]byte, error) {
				return HelpStepFrame(ctx.Identity, ctx.HelpStep)
			})
		}
	}
	return buildFrames(builders...)
}

// SingleStepMarkFrame 跳过簇里的空参消息 m543（#1683，1346 置位之后）。
func SingleStepMarkFrame() ([]byte, error) {
	return CustomMessageFrame(serverCustomSingleMark)
}

// FreshmanQueryFrames 待选态弹窗帧 m346 [2, 0] → m427 []：官服五个弹窗会话
// 恒此序（建号-01 #898/#899 首登、建号-02 #71/#72 重登——重登没有锁输入帧
// 也照弹），锁输入由调用方紧随其后发。427 的 handler 是客户端 lua
// custom_handler.on_query_fresh_man：util_get_form("form_single_blackback")
// ＋ShowModal。
func FreshmanQueryFrames() ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return CustomMessageFrame(serverCustomProcessSingle, CustomInt(2), CustomInt(0))
		},
		func() ([]byte, error) { return CustomMessageFrame(serverCustomQueryFreshMan) },
	)
}

// freshmanRecInitial 待选新号的 FinishFreshmanRecStr 初值（0x10 idx225，
// 型 7 串线型携裸字节）：45×0x01 尾 09 00——官服建号.pcapng 男号 #134 原字节
// （0x0B 前的进场属性簇）。语义未解（字节疑似按引导步骤的标志位），照抄。
var freshmanRecInitial = append(append(make([]byte, 0, 47),
	bytes.Repeat([]byte{1}, 45)...), 0x09, 0x00)

// FreshmanRecFrame 引导记录 FinishFreshmanRecStr=初值（官服待选新号进场簇
// 照发）。引擎里读它的三处都是 form_main_map 新手进度条的回调，与弹窗
// 无关——弹窗见 FreshmanQueryFrames（旧「黑幕窗门控属性」的说法 2026-11-27
// 作废：只发本属性真机不弹窗，五个官服弹窗会话的共同帧是 m427）。
func FreshmanRecFrame(player Identity) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("FinishFreshmanRecStr", BytesValue(freshmanRecInitial))})
}

// BornGameStepFrame 出生服务端阶段 0x10 BornServerGameStep=<主步×10+子步>。
func BornGameStepFrame(player Identity, value int32) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("BornServerGameStep", Int32Value(value))})
}

// HelpStepFrame 新手帮助步骤 0x10 HelpStepID=<步骤>。
func HelpStepFrame(player Identity, step string) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("HelpStepID", StringValue(step))})
}

// GuideMaterializeFrames 0x0D 之后补推的引导属性：HelpStepID → FinishFreshmanRecStr
// 初值（仅待选态）→ BornServerGameStep，与 GuideEntryFrames 同值同判据。
//
// 进场簇那三条 0x10 在 0x0B 之前发，到不了场内玩家对象——进场会重建玩家对象，
// 只认 0x0D 属性区（官服 CN 2026 实证：0x0B 前单发过的属性在 0x0D 属性区整批
// 重发，HelpStepID/FinishFreshmanRecStr/BornServerGameStep 均在其列，完整引导-
// 女 #487 的 0x28 批内层）；m414 gamestep 是引擎全局态，留在 0x0B 前不动。本服
// 0x0D 沿 fork 样本属性集（187 项）不含这三项，故物化后补推。
//
// 缺 HelpStepID 的症状：客户端 freshman_help.movie_end_callback 首段查
// HelpStepID 为空即返回——接取影像播完客户端不发 m950，引导步骤链不推进
// （2026-11-28 五个真机会话零 m950 定案）。
func GuideMaterializeFrames(ctx Context) ([][]byte, error) {
	var builders []func() ([]byte, error)
	if ctx.HelpStep != "" {
		builders = append(builders, func() ([]byte, error) {
			return HelpStepFrame(ctx.Identity, ctx.HelpStep)
		})
	}
	if ctx.GuideChoicePending {
		builders = append(builders, func() ([]byte, error) {
			return FreshmanRecFrame(ctx.Identity)
		})
	}
	if ctx.GuideStep != 0 {
		builders = append(builders, func() ([]byte, error) {
			return BornGameStepFrame(ctx.Identity, int32(ctx.GuideStep)*10+guideStepSubOfficial)
		})
	}
	return buildFrames(builders...)
}

// InputLockFrame 输入锁 0x10 CanntOperator=1/0。
func InputLockFrame(player Identity, locked bool) ([]byte, error) {
	value := int32(0)
	if locked {
		value = 1
	}
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("CanntOperator", Int32Value(value))})
}

// ReliveFrame 复活点 0x10 RelivePositon=<串>。
func ReliveFrame(player Identity, relive string) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("RelivePositon", StringValue(relive))})
}

// GiveItemRowFrame 获得物品记录 0x11 GiveItemRec row=<行>（配置 ID、数量；
// 男号 #1695）。
func GiveItemRowFrame(objectID, ownerID uint32, row uint16, configID string, num int32) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordGiveItem, row, []RecordCell{
		RecordString(configID), RecordInt(uint32(num)),
	})
}

// LiveGrooveGainFrames 跳过簇的历练四帧：m1 [1,"8055",<点数>] → m24 [玩家,1,
// <点数>] → m24 [玩家,40,"huodelilian"] → 0x10 LiveGroove_<下标>=<新余额>
// （男号 #1700..#1703）。amount 是本次累加值、total 是累加后的余额（点数
// ＝amount/1000）。
func LiveGrooveGainFrames(player Identity, index int, amount, total int32) ([][]byte, error) {
	points := amount / 1000
	property, err := TaskLiveGrooveFrame(player, index, total)
	if err != nil {
		return nil, err
	}
	frames, err := buildFrames(
		func() ([]byte, error) { return TaskLiveGrooveNotice(points) },
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(1), CustomInt(points))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("huodelilian"))
		},
	)
	if err != nil {
		return nil, err
	}
	return append(frames, property), nil
}

// ReputeRowFrameWithFlag 声望记录行 0x11 Repute_Record row=<行>（标志列
// 取调用方给值：任务完成奖励官服恒 1，跳过引导的再战江湖奖励是 2——男号
// #1720）。
func ReputeRowFrameWithFlag(objectID, ownerID uint32, row uint16, name string, count, flag int32) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordRepute, row, []RecordCell{
		RecordString(name), RecordInt(uint32(count)), RecordInt(uint32(flag)),
	})
}

// LifeHelpUnlockFrames 生活新手帮助解锁提示 m142 ×4（采集／制造／文化／
// 视觉）＋收尾 m98 [712]（男号 #1723..#1727；女号完成奖励 #1936..#1937 同
// 形）。
func LifeHelpUnlockFrames() ([][]byte, error) {
	names := []string{"caiji", "zhizao", "wenhua", "shijing"}
	builders := make([]func() ([]byte, error), 0, len(names)+1)
	for _, name := range names {
		key := "life_freshmanhelp_" + name
		builders = append(builders, func() ([]byte, error) {
			return CustomMessageFrame(142, CustomString(key))
		})
	}
	builders = append(builders, func() ([]byte, error) { return CustomMessageFrame(98, CustomInt(712)) })
	return buildFrames(builders...)
}

// FunctionOpenNoticeFrame 功能开启提示 m98 [<功能号>]（男号声望行之后的
// #1721 = 177）。
func FunctionOpenNoticeFrame(function int32) ([]byte, error) {
	return CustomMessageFrame(98, CustomInt(function))
}

// EnterWorldFrames 「进入江湖」属性对 0x10 PowerLevel=1 → 0x10 YinLevel=1
// （男号 #1946..#1947）。
func EnterWorldFrames(player Identity) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
				[]IndexedProperty{NamedProperty("PowerLevel", Int32Value(1))})
		},
		func() ([]byte, error) {
			return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
				[]IndexedProperty{NamedProperty("YinLevel", Int32Value(1))})
		},
	)
}
