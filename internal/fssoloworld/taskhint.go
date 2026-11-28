package fssoloworld

// taskhint.go 新手引导提示 m371（2026-11-28 定版；帧编码 entergen/taskhint.go）。
//
// 现行客户端的「引导提示」是帮助窗：箭头动画指向控件＋提示文案（客户端
// ini\new_helper\player_helper.ini 的帮助类型 → helper_info.ini 节点：
// focus_control / ani_image / text=desc_xsyd_*，如 wuxue_help＝「右键点击行囊中
// 的技能书学习」指向行囊工具格）。开窗只认 S2C m371（handler 表两套同号：
// on_new_player_guid → 帮助类型，lua 延 0.5s 调
// form_main_helper_manager.open_helper_form(帮助类型, "1"[, 参数])；参数是
// 定位用的配置 ID，如背包里那一本书）。帮助类型 id 是服务端私有配置（客户端
// lua 与任务数据零引用），下面的表照官服「初始账号登陆-完整引导-女」的 23 条
// m371 逐条建成。
//
// 老机制在这个客户端已经死了：freshmanhelp.ini 的 tip/textinfo/tipTime/
// tipphoto/autoshow 键在全部 lua 字节码与引擎 dll 里零引用，
// freshman_help.auto_show_form 没有调用者——HelpStepID/m950 链只推进步骤号，
// 本身不出任何界面。
//
// 官服脚本与帧位（帧号＝完整引导-女 S2C 流内序号）：
//
//	步骤记满   入包帧 → m1 9645（逐件）→ 记录格 0x13×4 → m371 → m209 NPC
//	          标记 → m24 task_ok → m8          （#2362..#2377 / #2733..#2748）
//	  558/559/560  发书 → m371 [wuxue_help, 书]       #2367 / #2466 / #2619
//	  561          → m371 [voice_xushijia_help]      #2738
//	  562          m371 [add_item_yu_helper] 先于 m1 9645（#5080）→ 记录格
//	              → m371 [jh_tl_dztx_book_help, 书]（#5086）
//	  570 / 139    发书 → m371 [anqi_help_jq02_1 / anqi_help_jq02_3, 物品]
//	                                          （#5315 / #5516）
//	  564 / 566    发书 → m371 [yd_qinggong_help, 书]（#7561）/
//	              [miji_help, 内功书]（#8086）
//	  567 / 270 / 269  → m371 [gong_help_xiu_1]（#8822）/ [jingmai_help_two]
//	              （#9139）/ [jingmai_help_one]（#9378）
//	学会武学   … m141 → m1 9491 → m98 860 → m371 → 0x19 扣件
//	          招式书 → [skill_help, 武学]（#2451/#2540/#2676）；zs_default_01 →
//	          [jh_tl_dztx_skill_help]（#5149）；暗器书四连发里 hw_normal_fb 之后
//	          → [anqi_help_jq01_5, hw_normal_fb]（#5427）；轻功 qinggong_4 →
//	          [qinggong_help]（#7631）；内功 ng_jh_001 → [gong_help] 在 m141
//	          之前（#8154；再战江湖跳过簇同形，见 guide.go）
//	接取 1343  m371 [anqi_help_jq01_4] 在簇首（#5602，早于全部记录行）
//
// 未接（官服触发条件不在任务数据里）：voice_zhaojia_help（#3878，1341 木桩
// 练习中）、voice_neigongtiao_help（#6289）/ voice_pofang_help（#6506，1343
// 破防练习）、voice_qizhen_help（#10704，任务 1597 对话 496 发 qizhen_0101）。

import (
	"fmt"
	"os"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// taskHintsEnabled NINEYIN_TASK_HINTS=off 关停引导提示（真机 A/B）。
func taskHintsEnabled() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("NINEYIN_TASK_HINTS"))) != "off"
}

// taskHint 一条引导提示：帮助类型＋定位参数（空＝无参）。
type taskHint struct {
	helper string
	param  string
}

func (h taskHint) frame() ([]byte, error) {
	return entergen.TaskHintFrame(h.helper, h.param)
}

func hintOf(helper, param string) taskHint { return taskHint{helper: helper, param: param} }

// taskHintItemHints 步骤奖励物品 → 帮助类型（m371 [帮助类型, 物品]）。
var taskHintItemHints = map[string]string{
	"tbook_CS_jh_cqgf03": "wuxue_help",
	"tbook_CS_jh_cqgf02": "wuxue_help",
	"tbook_CS_jh_cqgf05": "wuxue_help",
	"book_zs_default_01": "jh_tl_dztx_book_help",
	"book_hw_normal":     "anqi_help_jq02_1",
	"hw_fb_rw_01003":     "anqi_help_jq02_3",
	"book_qinggong_4":    "yd_qinggong_help",
	"ng_tbook_jh_001":    "miji_help",
}

// taskHintStepPre 对话步（DialogID 键）记满时、发放之前的提示。
var taskHintStepPre = map[string]string{
	"DialogID_562": "add_item_yu_helper",
}

// taskHintStepPost 对话步记满时、记录格之后的提示（步骤奖励物品不在
// taskHintItemHints 时才用）。
var taskHintStepPost = map[string]string{
	"DialogID_561": "voice_xushijia_help",
	"DialogID_567": "gong_help_xiu_1",
	"DialogID_270": "jingmai_help_two",
	"DialogID_269": "jingmai_help_one",
}

// taskHintAccept 引导链任务接取簇首的提示。
var taskHintAccept = map[int]string{
	1343: "anqi_help_jq01_4",
}

// taskHintLearned 学会武学后的提示（按武学 ID）。
var taskHintLearned = map[string]taskHint{
	"CS_jh_cqgf03":  hintOf("skill_help", "CS_jh_cqgf03"),
	"CS_jh_cqgf02":  hintOf("skill_help", "CS_jh_cqgf02"),
	"CS_jh_cqgf05":  hintOf("skill_help", "CS_jh_cqgf05"),
	"zs_default_01": hintOf("jh_tl_dztx_skill_help", "zs_default_01"),
	"hw_normal_fb":  hintOf("anqi_help_jq01_5", "hw_normal_fb"),
	"qinggong_4":    hintOf("qinggong_help", ""),
	"ng_jh_001":     hintOf("gong_help", ""),
}

// taskHintForStep 步骤记满时的提示（仅引导链任务）：步骤奖励实际发出的物品
// 优先（按发放顺序逐件查表），没有则查对话步表；都没有零值。granted 是
// npcTaskGrantItems 实际入包的清单（目录缺席/背包满的件不算）。
func taskHintForStep(step taskdata.Step, granted []taskdata.PrizeItem) taskHint {
	for _, item := range granted {
		if helper, ok := taskHintItemHints[item.ID]; ok {
			return hintOf(helper, item.ID)
		}
	}
	if helper, ok := taskHintStepPost[step.Scene]; ok {
		return hintOf(helper, "")
	}
	return taskHint{}
}

// addTaskHint 追加提示帧（零值不加）。
func addTaskHint(out *taskFrames, hint taskHint) {
	if hint.helper == "" {
		return
	}
	out.add(hint.frame())
}

// npcTaskHintEmit 发一条引导提示并留痕（零值不发）。
func (s *Server) npcTaskHintEmit(pl *Player, out *taskFrames, hint taskHint, what string) {
	if hint.helper == "" {
		return
	}
	addTaskHint(out, hint)
	s.trace("npc task hint: %s提示 %s（%s）client=%d", what, hint.helper, hint.param, pl.Conn1)
}

// npcTaskStepHint 步骤记满后的引导提示：记录格之后、NPC 标记之前（官服
// #2367/#2738）；非引导链任务不发。调用方持 taskMu。
func (s *Server) npcTaskStepHint(pl *Player, out *taskFrames, taskID int, step taskdata.Step, granted []taskdata.PrizeItem) {
	if !taskHintsEnabled() {
		return
	}
	hint := taskHintForStep(step, granted)
	if hint.helper == "" {
		return
	}
	s.npcTaskHintEmit(pl, out, hint, fmt.Sprintf("任务 %d 步骤 %s ", taskID, step.Scene))
}

// npcTaskLearnedHint 学会一件武学后的引导提示，紧跟学会簇（官服
// m141 → m1 9491 → m98 860 → m371 → 扣件，#2447..#2452）。内功 ng_jh_001
// 的 gong_help 在学会簇之前（#8154）——由调用方按 before 区分。只对选了
// 「初出茅庐」的角色发：再战江湖线由跳过簇自带，老玩家学书不弹。
func npcTaskLearnedHint(skillID string, single bool) (hint taskHint, before bool, ok bool) {
	if !single || !taskHintsEnabled() {
		return taskHint{}, false, false
	}
	hint, ok = taskHintLearned[skillID]
	before = ok && hint.helper == "gong_help"
	return hint, before, ok
}
