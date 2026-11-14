package main

import (
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

// TestNPCCatalogLoads NPC 目录加载闸（换表工程 §14.3-P3 回归）：
// 2026-11-10 真机教训——外服私服方言下 VisibleNPCSpawnDefaultsCompatV1
// 的 ActionSoundMode(int32) 撞外服私服表唯一现 string、placement 层
// PosiX/Y/Z/Orient 为 fork-only 名，两层硬错令 5413/5413 creator 实例
// 尽没→选角链断连；修复后按当前表场景域过滤注入。spawns>0 必须。
func TestNPCCatalogLoads(t *testing.T) {
	reg, err := newSceneNPCRegistry(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	spawns, stats, err := reg.catalogFor(role.Scene{Config: "city05", Resource: "city05_chengdu"})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if len(spawns) == 0 {
		t.Fatal("catalog resolved zero spawns")
	}
	t.Logf("spawns=%d resolved=%d unresolved=%d", len(spawns), stats.Resolved, stats.Unresolved)
	if stats.Resolved == 0 {
		t.Fatal("resolved=0")
	}
}
