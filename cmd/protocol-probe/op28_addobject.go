package main

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
)

// op28_addobject.go：0x28 明文物化帧（外服方言，B 路线）。
//
// 外服真流定版（l.pcapng 224×op28 会话解码，2026-11-14）：
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
// 外服把大属性集物化帧走 0x28（值对 26+ 起），小对象走裸 0x0D。
// fork 客户端对 op 0x28 走 LZ 解压（fxnet2 分发表二级表），明文 0x46
// 帧会被当压缩流解压失败丢弃——本通道仅供外服家族客户端
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

// npcOfficial0x28Enabled：NINEYIN_NPC_0X28_OFFICIAL=1 时 NPC 物化帧按
// 外服 AOI 簇 16-prop 模板全新生成（不再 wrap fork 0x0D 大档）。
// 依据（c.pcapng 簇逐字节解码 + 历轮 16-prop 定版合并）：外服怪帧
// #1776 = 17 字段精选（CollideRadius/State/Type/Resource/ConfigID/
// Photo/ActionSpeed/Level/BossLevel/IDCard 族），坐标走变换块
// [8:60]，无 fork 表尾 PosiX/Orient 字段、无 Name——fork 大档字段集
// 外服客户端按自记录方言读不懂故不建对象（15:28 会话实证）。
func npcOfficial0x28Enabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_NPC_0X28_OFFICIAL")) == "1"
}

// officialNPC0x28Frame 按外服方言生成 mode=0x46 明文 0x28 物化帧。
// 字段值大部分为外服静态缺省（16-prop 定版），ConfigID/Resource 用
// 目录真值；串按 0x28 记录方言 [u32 len 含NUL][GBK+NUL]。
func officialNPC0x28Frame(objectID, ownerID uint32, x, y, z, yaw float32, configID, resource string) []byte {
	body := make([]byte, 0, 320)
	body = binary.LittleEndian.AppendUint32(body, objectID)
	body = binary.LittleEndian.AppendUint32(body, ownerID)
	for _, f := range []float32{x, y, z, yaw, 0, 0, 0, 0, npcDetail0x28BaseScale, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1)
	countAt := len(body)
	body = binary.LittleEndian.AppendUint16(body, 0) // count 占位

	count := 0
	put := func(idx uint16, typ int, val any) {
		body = binary.LittleEndian.AppendUint16(body, idx)
		switch v := val.(type) {
		case float32:
			body = binary.LittleEndian.AppendUint32(body, math.Float32bits(v))
		case uint32:
			body = binary.LittleEndian.AppendUint32(body, v)
		case int32:
			body = binary.LittleEndian.AppendUint32(body, uint32(v))
		case byte:
			body = append(body, v)
		case string:
			raw := append([]byte(nil), []byte(v)...)
			raw = append(raw, 0)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
			body = append(body, raw...)
		}
		_ = typ
		count++
	}
	// 官方 0x46 可建帧 19 字段定版（npc.pcapng #1841/#1898 明文走查，
	// 罗塞塔石碑——官方自己用 0x46 建对象的确切内容）：
	put(179, 5, float32(0.915))            // CollideRadius（样本 0.915）
	put(183, 7, "stand")                   // State（样本 walk=巡逻中，物化时刻 stand）
	put(181, 1, byte(4))                   // Type
	if resource != "" {
		put(1448, 7, resource) // Resource
	}
	put(1565, 7, "0.0,0.0,0.0")            // RotatePara（样本按朝向，静态零）
	put(1447, 7, configID)                 // ConfigID
	photoRoot := resource
	if photoRoot == "" {
		photoRoot = configID
	}
	put(184, 7, "icon\funcnpc\\" + photoRoot + ".png") // Photo（官方公式 iconuncnpc\）
	put(214, 5, float32(1.0))              // ActionSpeed
	put(1596, 3, uint32(1))                // NotPositive（样本白名=1）
	put(84, 7, "nil")                      // AIEffect
	put(1472, 1, byte(25))                 // Level（静态，后续接目录真值）
	put(1597, 1, byte(1))                  // BossLevel
	put(1602, 7, "null,null,null,null,null") // IDCard
	put(1604, 7, "stand")                  // BornAction
	put(1606, 7, "hand1_1")                // WeaponMode
	put(471, 3, uint32(100))               // HPRatio
	put(480, 3, uint32(100))               // ParryRatio
	put(484, 3, uint32(100))               // HitHPRatio
	binary.LittleEndian.PutUint16(body[countAt:], uint16(count))

	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))
	frame = append(frame, body...)
	return frame
}
