package entergen

// operateguide_test.go 操作模式引导 m704 对拍官服原帧（完整引导-女 #1419、
// 跳过引导-男 #1560、建号-01 #1332 同一字节；tmp/npctalk 程序化导出）。

import "testing"

func TestOperateModeGuideFrameOfficial(t *testing.T) {
	got, err := OperateModeGuideFrame()
	assertOfficial(t, got, err, "1e 01 00 02 c0 02 00 00", "m704 操作模式引导")
}
