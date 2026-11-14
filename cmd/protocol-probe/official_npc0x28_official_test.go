package main

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestOfficialNPC0x28FrameInvariants 程序反推不变量（2026-11-14 fxnet2
// ServerAddObject 0xdbc0 反汇编定案，docs/official/注入帧数据-AddObject全链程序反推.md）：
//
//	包装头：[0x28][0x46][帧长-1 u32@+2][体长 u32@+6][体@+10]——分发器读
//	       mode@+1 定 4B 长度头、outlen=u32@+6，解压长度 != outlen 即丢帧；
//	体部（= 0x0D 帧去 op 字节平移）：obj u32@0/own u32@4/posA f32×4@8/
//	       posB@24/scale@40/rot2@44/零@48,52/u32 标志@56（==1）/
//	       count u16@60/值对@62；
//	尺寸闸：ServerAddObject 要求 len(=outlen+1) >= 0x3F → 全帧 >= 0x40；
//	值对：idx u16 + 0x09 表线型宽（串=[u32 len 含NUL][GBK+NUL]），
//	       idx 越表 = 整帧 "property error"。
func TestOfficialNPC0x28FrameInvariants(t *testing.T) {
	const obj, own uint32 = 0x7A5C0B, 0x1105C0E3
	frame := officialNPC0x28Frame(obj, own, 928.33, 48.75, 998.877, 0.906, "kuangshi_1001npc01", "npc\\cj_102005", 14)

	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("header=% x, want 28 46", frame[:2])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(len(frame)-1) {
		t.Fatalf("len-1=%d, want %d", got, len(frame)-1)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != uint32(len(frame)-10) {
		t.Fatalf("outlen=%d, want %d（解压长度校验）", got, len(frame)-10)
	}
	if len(frame) < 0x40 {
		t.Fatalf("frame len=%d < 0x40（ServerAddObject size error 闸）", len(frame))
	}

	body := frame[10:]
	if got := binary.LittleEndian.Uint32(body); got != obj {
		t.Fatalf("obj=%#x, want %#x", got, obj)
	}
	if got := binary.LittleEndian.Uint32(body[4:]); got != own {
		t.Fatalf("own=%#x, want %#x", got, own)
	}
	posA := [4]float32{928.33, 48.75, 998.877, 0.906}
	for i, want := range posA {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(body[8+4*i:])); got != want {
			t.Fatalf("posA[%d]=%v, want %v", i, got, want)
		}
		// 静立形 posB==posA（官方帧 A≠B 为走动物化，模板生成器 A=B）。
		if got := math.Float32frombits(binary.LittleEndian.Uint32(body[24+4*i:])); got != want {
			t.Fatalf("posB[%d]=%v, want %v（静立同值）", i, got, want)
		}
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(body[40:])); got != npcDetail0x28BaseScale {
		t.Fatalf("scale=%v, want %v", got, npcDetail0x28BaseScale)
	}
	if got := binary.LittleEndian.Uint32(body[56:]); got != 1 {
		t.Fatalf("标志 u32@56=%d, want 1（官方帧恒 1）", got)
	}

	// 值对区走查：count@60、逐对按表线型宽推进、终点必须精确贴齐体长、
	// 串值 len 含尾 NUL 且末字节为 NUL。
	count := int(binary.LittleEndian.Uint16(body[60:]))
	off := 62
	seen := map[string]bool{}
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("field#%d idx=%d 越当前 0x09 表（客户端整帧拒）", i, idx)
		}
		off += 2
		switch entry.Type {
		case clientdata.WireByte:
			off++
		case clientdata.WireWord:
			off += 2
		case clientdata.WireInt32, clientdata.WireFloat32:
			off += 4
		case clientdata.WireInt64, clientdata.WireObject:
			off += 8
		case clientdata.WireString, clientdata.WireWideString:
			n := int(binary.LittleEndian.Uint32(body[off:]))
			off += 4
			if n < 1 || off+n > len(body) {
				t.Fatalf("field#%d %s 串长 %d 越界", i, entry.Name, n)
			}
			if body[off+n-1] != 0 {
				t.Fatalf("field#%d %s 串 len=%d 但末字节非 NUL", i, entry.Name, n)
			}
			off += n
		default:
			t.Fatalf("field#%d %s 线型 %d 不可编码（0/6 占位行）", i, entry.Name, entry.Type)
		}
		seen[entry.Name] = true
	}
	if off != len(body) {
		t.Fatalf("值对区终点 %d != 体长 %d（语法失步=半坏帧）", off, len(body))
	}
	for _, must := range []string{"CollideRadius", "State", "Type", "Resource", "ConfigID", "Photo", "NpcType"} {
		if !seen[must] {
			t.Fatalf("模板缺 %s 行", must)
		}
	}
	// NpcType 官方位次：Photo(184) 与 ActionSpeed(214) 之间（npc.pcapng
	// 原生 19 字段怪帧 #1841/#1898/#1956 实证，i32=14）。
	order := []string{}
	texts := map[string]string{}
	i32s := map[string]int32{}
	off = 62
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, _ := clientdata.PropertyByIndex(int(idx))
		off += 2
		order = append(order, entry.Name)
		switch entry.Type {
		case clientdata.WireString, clientdata.WireWideString:
			n := int(binary.LittleEndian.Uint32(body[off:]))
			texts[entry.Name] = string(body[off+4 : off+4+n-1])
			off += 4 + n
		case clientdata.WireInt32:
			i32s[entry.Name] = int32(binary.LittleEndian.Uint32(body[off:]))
			off += 4
		case clientdata.WireByte:
			off++
		case clientdata.WireWord:
			off += 2
		case clientdata.WireFloat32:
			off += 4
		case clientdata.WireInt64, clientdata.WireObject:
			off += 8
		}
	}
	pos := map[string]int{}
	for i, n := range order {
		if _, ok := pos[n]; !ok {
			pos[n] = i
		}
	}
	if !(pos["Photo"] < pos["NpcType"] && pos["NpcType"] < pos["ActionSpeed"]) {
		t.Fatalf("NpcType 位次错：Photo@%d NpcType@%d ActionSpeed@%d", pos["Photo"], pos["NpcType"], pos["ActionSpeed"])
	}
	if i32s["NpcType"] != 14 {
		t.Fatalf("NpcType=%d, want 14（官方怪形常量）", i32s["NpcType"])
	}
	// Photo 官方样本形 icon\funcnpc\<model>.png——\f 笔误回归锚
	//（2026-11-14 曾以换页符 0x0C 下发：Go 串 "icon\funcnpc" 的 \f）。
	if photo := texts["Photo"]; len(photo) < 13 || photo[:13] != "icon\\funcnpc\\" {
		t.Fatalf("Photo=%q 前缀非 icon\\funcnpc\\（\\f 换页符回归?）", photo)
	}
	for name, text := range texts {
		for i := 0; i < len(text); i++ {
			if text[i] == 0x0C {
				t.Fatalf("%s=%q 含换页符 0x0C", name, text)
			}
		}
	}
}
