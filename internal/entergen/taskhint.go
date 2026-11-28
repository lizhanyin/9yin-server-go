package entergen

// taskhint.go 新手引导提示 m371 的帧编码（提示表与下发位次见
// fssoloworld/taskhint.go）。
//
//	S2C m371 [帮助类型]            官服完整引导-女 #2738
//	    1e 02 00 02 73 01 00 00 06 14 00 00 00 "voice_xushijia_help\0"
//	S2C m371 [帮助类型, 配置 ID]   #2367
//	    1e 03 00 02 73 01 00 00 06 0b 00 00 00 "wuxue_help\0"
//	    06 13 00 00 00 "tbook_CS_jh_cqgf03\0"

import "fmt"

// serverCustomNewPlayerGuide 新手引导提示（客户端 handler 表两套同号：
// on_new_player_guid；372 是它的江湖变体 on_jianghu_guide）。
const serverCustomNewPlayerGuide int32 = 371

// TaskHintFrame m371 [帮助类型[, 配置 ID]]：configID 空＝无参。帮助类型是
// 客户端 ini\new_helper\player_helper.ini 的节名，配置 ID 是帮助窗定位控件
// 用的物品／武学 ID。
func TaskHintFrame(helper, configID string) ([]byte, error) {
	if helper == "" {
		return nil, fmt.Errorf("task hint: empty helper")
	}
	values := make([]CustomValue, 0, 2)
	values = append(values, CustomString(helper))
	if configID != "" {
		values = append(values, CustomString(configID))
	}
	return CustomMessageFrame(serverCustomNewPlayerGuide, values...)
}
