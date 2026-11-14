package main

import (
	"encoding/binary"
	"os"
	"strings"
)

// op28_addobject.go：0x28 明文物化帧（官服方言，B 路线）。
//
// 官服真流定版（l.pcapng 224×op28 会话解码，2026-11-14）：
//
//	帧 = [op 0x28][mode 0x46|0x47][帧长-1 u32][体长 u32][体]
//	mode 0x46 = 明文体（6 帧先例：outlen==体长，免 LZ）
//	mode 0x47 = LZ 压缩体（218 帧，解压=同布局明文）
//	体  = 0x0D AddObject 帧去掉首字节 op 的整体平移：
//	      [名 int32@0][行名 int32@4][X/Y/Z/Orient f32@8..24]
//	      [固定头余量][u32@0x38][值对数 u16@0x3C][值对@0x3E]
//	值对 = 标准 0x09 表方言（idx u16 + 表线型定宽值）——首值对
//	      CollideRadius@179 形态与 0x0D #1777 全等。
//
// 官服把大属性集物化帧走 0x28（值对 26+ 起），小对象走裸 0x0D。
// fork 客户端对 op 0x28 走 LZ 解压（fxnet2 分发表二级表），明文 0x46
// 帧会被当压缩流解压失败丢弃——本通道仅供官服家族客户端
//（NINEYIN_NPC_ADD_AS_0X28=1 启用，默认关不扰 fork 会话）。

// npcAddAsOp28Enabled 控制 NPC 物化帧改走 0x28 明文方言。
func npcAddAsOp28Enabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_ADD_AS_0X28")) == "1"
}

// wrapObjectAddAsOp28 把 0x0D AddObject 帧包装成 mode=0x46 明文 0x28 帧。
// 入参 plain 必须以 0x0D 开头（含 op 字节）；出参可直接下发。
func wrapObjectAddAsOp28(plain []byte) []byte {
	if len(plain) < 2 || plain[0] != 0x0D {
		return plain
	}
	body := plain[1:]
	frame := make([]byte, 10, 10+len(body))
	frame[0] = 0x28
	frame[1] = 0x46
	binary.LittleEndian.PutUint32(frame[2:], uint32(9+len(body))) // 帧长-1
	binary.LittleEndian.PutUint32(frame[6:], uint32(len(body)))   // 体长
	frame = append(frame, body...)
	return frame
}

// npcResourceOverride 判别实验开关：NINEYIN_NPC_RESOURCE_OVERRIDE=npc\monster483
// 强制全 NPC 用同一外服库在册模型串。
func npcResourceOverride() string {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_RESOURCE_OVERRIDE"))
}
