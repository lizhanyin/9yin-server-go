package entergen

// taskhint_test.go 新手引导提示 m371 对拍官服「初始账号登陆-完整引导-女」的
// 原帧字节（期望值由 tmp/npctalk 从抓包程序化导出，帧号是该会话 S2C 流内
// 的 0 基序号）。

import "testing"

// TestTaskHintFrameOfficial 有参／无参两种形态逐字节等于官服帧。
func TestTaskHintFrameOfficial(t *testing.T) {
	cases := []struct {
		helper, configID string
		want             string
	}{
		{"wuxue_help", "tbook_CS_jh_cqgf03", // #2367：发书后指向行囊里那本书
			"1e 03 00 02 73 01 00 00 06 0b 00 00 00 77 75 78 75 65 5f 68 65 6c 70 00 06 13 00 00 00 74 62 6f 6f 6b 5f 43 53 5f 6a 68 5f 63 71 67 66 30 33 00"},
		{"skill_help", "CS_jh_cqgf03", // #2451：学会招式后
			"1e 03 00 02 73 01 00 00 06 0b 00 00 00 73 6b 69 6c 6c 5f 68 65 6c 70 00 06 0d 00 00 00 43 53 5f 6a 68 5f 63 71 67 66 30 33 00"},
		{"voice_xushijia_help", "", // #2738：对话步记满
			"1e 02 00 02 73 01 00 00 06 14 00 00 00 76 6f 69 63 65 5f 78 75 73 68 69 6a 69 61 5f 68 65 6c 70 00"},
		{"add_item_yu_helper", "", // #5080：发书前
			"1e 02 00 02 73 01 00 00 06 13 00 00 00 61 64 64 5f 69 74 65 6d 5f 79 75 5f 68 65 6c 70 65 72 00"},
		{"jh_tl_dztx_book_help", "book_zs_default_01", // #5086
			"1e 03 00 02 73 01 00 00 06 15 00 00 00 6a 68 5f 74 6c 5f 64 7a 74 78 5f 62 6f 6f 6b 5f 68 65 6c 70 00 06 13 00 00 00 62 6f 6f 6b 5f 7a 73 5f 64 65 66 61 75 6c 74 5f 30 31 00"},
		{"anqi_help_jq01_4", "", // #5602：1343 接取簇首
			"1e 02 00 02 73 01 00 00 06 11 00 00 00 61 6e 71 69 5f 68 65 6c 70 5f 6a 71 30 31 5f 34 00"},
		{"qinggong_help", "", // #7631
			"1e 02 00 02 73 01 00 00 06 0e 00 00 00 71 69 6e 67 67 6f 6e 67 5f 68 65 6c 70 00"},
		{"gong_help", "", // #8154
			"1e 02 00 02 73 01 00 00 06 0a 00 00 00 67 6f 6e 67 5f 68 65 6c 70 00"},
		{"jingmai_help_two", "", // #9139
			"1e 02 00 02 73 01 00 00 06 11 00 00 00 6a 69 6e 67 6d 61 69 5f 68 65 6c 70 5f 74 77 6f 00"},
	}
	for _, tc := range cases {
		got, err := TaskHintFrame(tc.helper, tc.configID)
		assertOfficial(t, got, err, tc.want, "m371 "+tc.helper)
	}
	if got, err := TaskHintFrame("", "x"); err == nil || got != nil {
		t.Fatalf("空帮助类型应报错, got % x, %v", got, err)
	}
}
