package main

import (
	"encoding/binary"
	"testing"
)

// TestWrapObjectAddAsOp28 对照官服 l.pcapng 明文 0x28 帧定版结构
//（帧 #2537：len-1=328/outlen=319、体=[名@0][行名@4][xyz/orient@8][值对数@0x3C][值对@0x3E]，
// 首值对 CollideRadius@179 官方位）。
func TestWrapObjectAddAsOp28(t *testing.T) {
	plain, err := npcAddObject(2, testNPCSpawn(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] != 0x0D {
		t.Fatalf("source frame op=%#x", plain[0])
	}
	frame := wrapObjectAddAsOp28(plain)
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("header=%x", frame[:2])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(len(frame)-1) {
		t.Fatalf("len-1=%d, want %d (帧长-1)", got, len(frame)-1)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != uint32(len(plain)-1) {
		t.Fatalf("outlen=%d, want %d (体长)", got, len(plain)-1)
	}
	// 体 = 0x0D 帧去掉首字节逐字节平移。
	body := frame[10:]
	if len(body) != len(plain)-1 {
		t.Fatalf("body len=%d, want %d", len(body), len(plain)-1)
	}
	for i := range body {
		if body[i] != plain[i+1] {
			t.Fatalf("body[%d]=%#x, want %#x（0x0D 平移-1）", i, body[i], plain[i+1])
		}
	}
	// 体头布局锚（官服帧实证）：名@0/行名@4/坐标簇@8。
	name, rowname := int32(binary.LittleEndian.Uint32(body)), int32(binary.LittleEndian.Uint32(body[4:]))
	if name == 0 || rowname == 0 {
		t.Fatalf("name/rowname zero: %#x/%#x", name, rowname)
	}
}

// TestWrapObjectAddAsOp28RejectsNonAdd 非 0x0D 帧原样返回（防御）。
func TestWrapObjectAddAsOp28RejectsNonAdd(t *testing.T) {
	keep := []byte{0x10, 1, 2, 3}
	got := wrapObjectAddAsOp28(keep)
	if len(got) != len(keep) || got[0] != 0x10 {
		t.Fatalf("non-0x0D frame must pass through, got %x", got)
	}
	for i := range keep {
		if got[i] != keep[i] {
			t.Fatalf("pass-through mutated byte %d: %x", i, got)
		}
	}
}
