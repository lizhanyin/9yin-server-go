package neigongdata

// neigongdata_test.go 目录装载单测：夹具 share 树（五行/曲线/varprop
// 档/静态区间/buff StaticData）解析与查询面。

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCatalogQueries(t *testing.T) {
	root := writeTree(t, map[string]string{
		"share/faculty/wuxuebaseinfo.ini": "[ng_wd_001]\nnWuXing=5\n[ng_jh_201]\nnWuXing=7\n",
		"share/faculty/wuxuelevelinfo.ini": "[ng_wd_001]\nr=1,200,100\nr=2,600,100\n",
		"share/skill/neigong/neigong_static.ini":   "[5459]\nMinVarPropNo=10\nMaxVarPropNo=12\n",
		"share/skill/neigong/neigong_varprop.ini":  "[10]\nLevel=1\nNeiGongLevel=3\n[11]\nLevel=2\nNeiGongLevel=6\nBufferID=buf_x\nBufferLevel=2\nStrAdd=4\n@PropModifyPackRec=20,21\n",
		"share/skill/buff_new.ini":                 "[buf_x]\nStaticData=1742\n",
		"share/modifypack/proppack.ini": "[20]\nr=,MaxHPAdd,1630,0\nr=,MinMagicDefAdd,45,0\nComment=x\n[21]\nr=,MaxMPAdd,10,0\nr=,StrAdd,3,0\n[30]\nr=someSkill,SomeName,5,0\nr=,MaxHPAdd,999,1\n",
	})
	catalog, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.WuXingOf("ng_wd_001"); got != 5 {
		t.Fatalf("WuXingOf(ng_wd_001)=%d, want 5", got)
	}
	if got := catalog.WuXingOf("ng_jh_201"); got != 7 {
		t.Fatalf("WuXingOf(ng_jh_201)=%d, want 7", got)
	}
	if got := catalog.WuXingOf("absent"); got != 0 {
		t.Fatalf("未收录应 0, got %d", got)
	}
	if total, ok := catalog.LevelTotal("ng_wd_001", 2); !ok || total != 600 {
		t.Fatalf("LevelTotal(l2)=%d,%v, want 600,true", total, ok)
	}
	if _, ok := catalog.LevelTotal("ng_wd_001", 9); ok {
		t.Fatal("曲线缺级应 ok=false")
	}
	row, ok := catalog.VarPropRow(5459, 2)
	if !ok || row.BufferID != "buf_x" || row.NeiGongLevel != 6 {
		t.Fatalf("VarPropRow(5459,2)=%+v,%v", row, ok)
	}
	// proppack 合并：直列 StrAdd=4 + pack 21 的 StrAdd=3；资源 Add 全由
	// pack 合成；技能形态行（30 节首位非数值）与 Mode≠0 行不入合并。
	if row.StrAdd != 7 || row.MaxHPAdd != 1630 || row.MaxMPAdd != 10 || row.MinMagicDefAdd != 45 {
		t.Fatalf("proppack 合并异常: %+v", row)
	}
	if row.MaxParryAdd != 0 {
		t.Fatalf("无 MaxParryAdd 包行应 0: %+v", row)
	}
	if _, ok := catalog.VarPropRow(5459, 3); ok {
		t.Fatal("档区间无该级应 ok=false")
	}
	if static, ok := catalog.BuffStaticData("buf_x"); !ok || static != 1742 {
		t.Fatalf("BuffStaticData(buf_x)=%d,%v", static, ok)
	}
}

func TestLoadRequiresWuXing(t *testing.T) {
	root := writeTree(t, map[string]string{
		"share/faculty/wuxuebaseinfo.ini": "[x]\nOtherKey=1\n",
	})
	if _, err := Load(root); err == nil {
		t.Fatal("无 nWuXing 行应整体报错")
	}
}

func TestNilCatalogDegraded(t *testing.T) {
	var catalog *Catalog
	if got := catalog.WuXingOf("ng_wd_001"); got != 0 {
		t.Fatalf("nil 目录 WuXingOf=%d, want 0", got)
	}
	if _, ok := catalog.VarPropRow(1, 1); ok {
		t.Fatal("nil 目录 VarPropRow 应 false")
	}
}
