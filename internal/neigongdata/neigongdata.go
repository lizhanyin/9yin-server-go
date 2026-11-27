// Package neigongdata 内功目录装载（服务端只读，客户端设计态 share 树
// 直读；移植 2433 线 cmd/protocol-probe/neigong_effects.go 的目录面；
// 效果面收 @PropModifyPackRec 属性包（proppack.ini），@SkillModifyPackRec
// 技能包不在本包）。
//
//	share/faculty/wuxuebaseinfo.ini    configId → nWuXing（五行/属性：
//	    内功 5=太极/6=阴柔/7=阳刚；缺行 0，客户端按 ≤0 退回缺省文案）
//	share/faculty/wuxuelevelinfo.ini   ng_* 每级修炼曲线（r=level,total,
//	    speed；只取 total 列 = 升到该级所需累计修炼度）
//	share/skill/neigong/neigong_static.ini  StaticData → varprop 档区间
//	    （MinVarPropNo/MaxVarPropNo）
//	share/skill/neigong/neigong_varprop.ini 档行：Level/NeiGongLevel/
//	    BufferID/BufferLevel/五维 Add（档缺 0）
//	share/modifypack/proppack.ini       @PropModifyPackRec 属性修改器包
//	    （r=Name,Value,Mode；Mode 0 直加——合并进档行的五维/资源 Add）
//	share/skill/buff_new.ini           buff 名 → StaticData（BufferInfo4
//	    CSV 首列 staticHex 的来源）
//
// 全部 best-effort：单档缺失/解析失败保持空表不废整目录（夹具仓/精简
// 部署可降级运行）。GBK ini 方言与 itemdata 同款（合法 UTF-8 直通，
// 否则 GB18030 转码）。
package neigongdata

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// VarProp 一档 varprop 行（neigong_varprop.ini）。五维/资源 Add 已含
// @PropModifyPackRec 引用的 proppack.ini 修改器合并（Target 空/Mode 0
// 直加——2433 neigong_effects.go effect() 口径）。
type VarProp struct {
	Level       int32  // 档对应的书等级
	NeiGongLevel int32 // 实力轴（引擎 PowerLevel 数据源）
	BufferID    string
	BufferLevel int32
	StrAdd      int32
	StaAdd      int32
	DexAdd      int32
	IngAdd      int32
	SpiAdd      int32
	// 资源 Add（proppack 合成面；varprop 直列无这些键）。
	MaxHPAdd       int32
	MaxMPAdd       int32
	MaxParryAdd    int32
	MinMagicDefAdd int32
}

// propPackRow proppack.ini 一行修改器（r=Name,Value,Mode；技能包多一
// 位 Target 前缀，本面只收属性包）。
type propPackRow struct {
	name  string
	value int32
	mode  int32
}

// Catalog 内功目录（进程级只读缓存）。
type Catalog struct {
	wuXing      map[string]int32       // configId → nWuXing
	levelTotals map[string]map[int32]int32 // configId → level → total
	ranges      map[int32]modernRange // StaticData → varprop 档区间
	varProps    map[int32]VarProp     // VarPropNo → 档行（pack 已合并）
	varPacks    map[int32][]int32     // VarPropNo → @PropModifyPackRec 列表
	propPacks   map[int32][]propPackRow // proppack.ini 节号 → 修改器行
	buffStatic  map[string]uint32     // buff 名 → StaticData
	source      string
}

type modernRange struct {
	min, max int32
}

// WuXingOf configId 的五行/属性（未收录 0）。
func (c *Catalog) WuXingOf(configID string) int32 {
	if c == nil {
		return 0
	}
	return c.wuXing[configID]
}

// LevelTotal 升到 level 级所需累计修炼度（configId 曲线缺该级时 ok=false，
// 调用方回退自然曲线）。
func (c *Catalog) LevelTotal(configID string, level int32) (int32, bool) {
	if c == nil {
		return 0, false
	}
	total, ok := c.levelTotals[configID][level]
	return total, ok
}

// VarPropRow StaticData 的书在 level 级的 varprop 档行（区间内按 Level
// 匹配；2433 线 varPropRow 同序）。
func (c *Catalog) VarPropRow(staticData, level int32) (VarProp, bool) {
	if c == nil {
		return VarProp{}, false
	}
	entry, exists := c.ranges[staticData]
	if !exists || entry.min <= 0 || entry.max < entry.min {
		return VarProp{}, false
	}
	for id := entry.min; id <= entry.max; id++ {
		if candidate, ok := c.varProps[id]; ok && candidate.Level == level {
			return candidate, true
		}
	}
	return VarProp{}, false
}

// BuffStaticData buff 名的 StaticData（BufferInfo4 CSV 首列）。
func (c *Catalog) BuffStaticData(buffID string) (uint32, bool) {
	if c == nil {
		return 0, false
	}
	static, ok := c.buffStatic[buffID]
	return static, ok
}

// Count 诊断面：收录的五行行数。
func (c *Catalog) Count() int {
	if c == nil {
		return 0
	}
	return len(c.wuXing)
}

// Source 来源描述。
func (c *Catalog) Source() string {
	if c == nil {
		return ""
	}
	return c.source
}

// Load 从数据根装载目录（root/share/…）。全部 best-effort：核心档
// （wuxuebaseinfo）缺席才整体报错，其余单档失败降级空表。
func Load(root string) (*Catalog, error) {
	catalog := &Catalog{
		wuXing:      map[string]int32{},
		levelTotals: map[string]map[int32]int32{},
		ranges:      map[int32]modernRange{},
		varProps:    map[int32]VarProp{},
		varPacks:    map[int32][]int32{},
		propPacks:   map[int32][]propPackRow{},
		buffStatic:  map[string]uint32{},
		source:      root,
	}
	base := func(parts ...string) string {
		return filepath.Join(append([]string{root, "share"}, parts...)...)
	}
	if err := parseINI(base("faculty", "wuxuebaseinfo.ini"), func(section, key, value string) error {
		if key != "nWuXing" {
			return nil
		}
		if parsed, err := parseInt32(value); err == nil {
			catalog.wuXing[section] = parsed
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("neigongdata: wuxuebaseinfo: %w", err)
	}
	if len(catalog.wuXing) == 0 {
		return nil, fmt.Errorf("neigongdata: %s 无 nWuXing 行", root)
	}
	_ = parseINI(base("faculty", "wuxuelevelinfo.ini"), func(section, key, value string) error {
		if key != "r" || !strings.HasPrefix(section, "ng_") {
			return nil
		}
		parts := strings.Split(value, ",")
		if len(parts) < 2 {
			return nil
		}
		level, err := parseInt32(parts[0])
		if err != nil {
			return nil
		}
		total, err := parseInt32(parts[1])
		if err != nil {
			return nil
		}
		if catalog.levelTotals[section] == nil {
			catalog.levelTotals[section] = map[int32]int32{}
		}
		catalog.levelTotals[section][level] = total
		return nil
	})
	_ = parseINI(base("skill", "neigong", "neigong_static.ini"), func(section, key, value string) error {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			return nil
		}
		entry := catalog.ranges[int32(id)]
		switch key {
		case "MinVarPropNo":
			entry.min, _ = parseInt32(value)
		case "MaxVarPropNo":
			entry.max, _ = parseInt32(value)
		}
		catalog.ranges[int32(id)] = entry
		return nil
	})
	_ = parseINI(base("skill", "neigong", "neigong_varprop.ini"), func(section, key, value string) error {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			return nil
		}
		record := catalog.varProps[int32(id)]
		var parseErr error
		switch key {
		case "Level":
			record.Level, parseErr = parseInt32(value)
		case "NeiGongLevel":
			record.NeiGongLevel, parseErr = parseInt32(value)
		case "BufferID":
			record.BufferID = strings.TrimSpace(value)
		case "BufferLevel":
			record.BufferLevel, parseErr = parseInt32(value)
		case "@PropModifyPackRec":
			// 修改器包引用（单值或逗号列表；技能包 @SkillModifyPackRec
			// 不在本面）。
			for _, part := range strings.Split(value, ",") {
				if pack, err := parseInt32(part); err == nil {
					catalog.varPacks[int32(id)] = append(catalog.varPacks[int32(id)], pack)
				}
			}
		case "StrAdd", "StaAdd", "DexAdd", "IngAdd", "SpiAdd":
			var parsed int32
			parsed, parseErr = parseInt32(value)
			if parseErr == nil {
				switch key {
				case "StrAdd":
					record.StrAdd = parsed
				case "StaAdd":
					record.StaAdd = parsed
				case "DexAdd":
					record.DexAdd = parsed
				case "IngAdd":
					record.IngAdd = parsed
				case "SpiAdd":
					record.SpiAdd = parsed
				}
			}
		}
		if parseErr != nil {
			return nil
		}
		catalog.varProps[int32(id)] = record
		return nil
	})
	_ = parseINI(base("skill", "buff_new.ini"), func(section, key, value string) error {
		if key != "StaticData" {
			return nil
		}
		if parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32); err == nil {
			catalog.buffStatic[section] = uint32(parsed)
		}
		return nil
	})
	// proppack.ini 修改器包（r=Name,Value,Mode；技能包形态 Name 位是
	// Target 前缀——首段可数值判别，本面只收三段属性行）。缺席降级空表
	//（资源 Add 全零）。
	_ = parseINI(filepath.Join(root, "share", "modifypack", "proppack.ini"), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			return nil
		}
		parts := strings.Split(value, ",")
		for len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
			parts = parts[1:]
		}
		if len(parts) < 3 {
			return nil
		}
		amount, err1 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		mode, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
		if err1 != nil || err2 != nil {
			return nil
		}
		catalog.propPacks[int32(id)] = append(catalog.propPacks[int32(id)], propPackRow{
			name: strings.TrimSpace(parts[0]), value: int32(amount), mode: int32(mode)})
		return nil
	})
	// 档行的 pack 合并（Target 空/Mode 0 直加——2433 effect() 口径；
	// 五维/资源 Add 累加进档行返回值）。
	for id, packs := range catalog.varPacks {
		record := catalog.varProps[id]
		for _, pack := range packs {
			for _, modifier := range catalog.propPacks[pack] {
				if modifier.mode != 0 {
					continue
				}
				switch modifier.name {
				case "StrAdd":
					record.StrAdd += modifier.value
				case "StaAdd":
					record.StaAdd += modifier.value
				case "DexAdd":
					record.DexAdd += modifier.value
				case "IngAdd":
					record.IngAdd += modifier.value
				case "SpiAdd":
					record.SpiAdd += modifier.value
				case "MaxHPAdd":
					record.MaxHPAdd += modifier.value
				case "MaxMPAdd":
					record.MaxMPAdd += modifier.value
				case "MaxParryAdd":
					record.MaxParryAdd += modifier.value
				case "MinMagicDefAdd":
					record.MinMagicDefAdd += modifier.value
				}
			}
		}
		catalog.varProps[id] = record
	}
	return catalog, nil
}

// parseINI GBK ini 方言迭代（; / // 注释；[节] 键=值）。文件缺席返回错误，
// 行级畸形静默跳过。
func parseINI(path string, consume func(section, key, value string) error) error {
	raw, err := res.ReadFile(path)
	if err != nil {
		return err
	}
	if !utf8.Valid(raw) {
		if decoded, derr := simplifiedchinese.GB18030.NewDecoder().Bytes(raw); derr == nil {
			raw = decoded
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || section == "" {
			continue
		}
		if err := consume(section, strings.TrimSpace(key), value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func parseInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(parsed), nil
}
