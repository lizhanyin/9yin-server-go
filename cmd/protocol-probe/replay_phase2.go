package main

import (
	"encoding/binary"
	"log"
	"os"
	"sync/atomic"
)

// B 线两阶段回放（2026-11-14 第十轮）。官服抓包时序定案（c/登陆/登陆2/
// 初始账号 四份交叉验证）：0x0B+化身 0x28+首爆发在进场即发；客户端读条
// 3~10s 完成后回发单字节 0x09，服务端 ~65ms 后发第二爆发（m561/m681 族、
// op15 建视图 + 0x18 视图行 + 0x2A 外观），再 ~1s 起按 1s tick 推 AOI
// 物化簇（0x28+0x0D）。把全序列在单点灌入时，Phase A 里的官服 0x0B 会
// 让客户端进入新一轮读条，其后帧全部落在读条窗口——场景对象表未建，
// AddObject 查表失败全丢。因此拆两段对齐官方相位：
//   Phase A = NINEYIN_REPLAY_FRAMES_FILE：现有 clientActivity 钩子发送
//             （0x0B+化身 0x28+首爆发）；
//   Phase B = NINEYIN_REPLAY_PHASE2_FILE：客户端对 Phase A 的 0x0B 回发
//             的下一个显式 0x09 ClientReady 时发送（第二爆发+物化簇）。
var replayPhase2Armed atomic.Bool

// replayFramesPack 原样回放 [u32 len][frame]×N 打包文件（tools 打包器
// 产出）；裸单帧文件回退。返回发送帧数。
func replayFramesPack(conn sceneMessageConnection, path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("replay file %s unreadable: %v", path, err)
		return 0
	}
	sent := 0
	rest := raw
	for len(rest) >= 4 {
		fl := int(binary.LittleEndian.Uint32(rest))
		if fl <= 0 || fl > len(rest)-4 || fl > 0x10000 {
			break
		}
		if err := conn.WriteFrame(rest[4 : 4+fl]); err != nil {
			log.Printf("replay %s: %v", path, err)
			break
		}
		sent++
		rest = rest[4+fl:]
	}
	// 兼容裸单帧文件
	if sent == 0 && len(raw) > 0 {
		if err := conn.WriteFrame(raw); err != nil {
			log.Printf("replay %s: %v", path, err)
			return 0
		}
		sent = 1
	}
	return sent
}
