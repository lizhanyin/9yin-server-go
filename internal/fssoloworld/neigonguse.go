package fssoloworld

// neigonguse.go C2S 215 运行内功切换（2433 线 player_progress.go case 215
// 定版）：校验已学 → 换 st.CurNeiGong → 帧序 MaxSP 专帧（先刷分母）→
// 换书资源重推（钳新上限后的 HPRatio/HP/MP/SP 当前值——高换低=新低值、
// 低换高=旧值，不回满）→ 资源批（CurNeiGong/五行/FacultyName/BufferInfo4
// 动态——旧书 buff 图标随批刷新消失）→ 战斗面板单属性逐帧
//（CombatAttributeValues 同一取值面）→ m169 两值帧挂新书可见 buff（buff
// 栏图标只由 m169 驱动）→ sendRoleSave。取数面 internal/neigongdata
//（客户端设计态 share 树直读）。

import (
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/neigongdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

var (
	neiGongCatalogMu     sync.Mutex
	neiGongCatalogTried  string
	neiGongCatalogCached *neigongdata.Catalog
)

// neiGongCatalogOf 内功目录（进程级缓存，skillCatalog 同款数据根锚定；
// 全缺返回 nil——调用方按缺目录降级：五行 0、BufferInfo4 空）。
func (s *Server) neiGongCatalogOf() *neigongdata.Catalog {
	roots := skillDataRoots()
	key := strings.Join(roots, "\x00")
	neiGongCatalogMu.Lock()
	defer neiGongCatalogMu.Unlock()
	if key == neiGongCatalogTried {
		return neiGongCatalogCached
	}
	neiGongCatalogTried, neiGongCatalogCached = key, nil
	for _, root := range roots {
		catalog, err := neigongdata.Load(root)
		if err == nil {
			neiGongCatalogCached = catalog
			s.trace("neigong catalog loaded from %s（%d 行五行表）", root, catalog.Count())
			break
		}
	}
	if neiGongCatalogCached == nil {
		s.trace("neigong catalog unavailable（候选根 %v）：五行/BufferInfo4 降级", roots)
	}
	return neiGongCatalogCached
}

// resetNeiGongCatalog 清目录缓存（测试缝）。
func resetNeiGongCatalog() {
	neiGongCatalogMu.Lock()
	neiGongCatalogTried, neiGongCatalogCached = "", nil
	neiGongCatalogMu.Unlock()
}

// equipBookOf 已学书中按 ConfigID 取书（nil=未学）。
func equipBookOf(st *rolestate.RoleState, configID string) *rolestate.NeiGongBook {
	for i := range st.NeiGong {
		if st.NeiGong[i].ConfigID == configID {
			return &st.NeiGong[i]
		}
	}
	return nil
}

// runningBookOf 运行内功书（CurNeiGong 匹配；缺席/未设回退首本——
// entergen vitalBook 同口径；无书 nil）。
func runningBookOf(st *rolestate.RoleState) *rolestate.NeiGongBook {
	if book := equipBookOf(st, st.CurNeiGong); book != nil {
		return book
	}
	if len(st.NeiGong) > 0 {
		return &st.NeiGong[0]
	}
	return nil
}

// derefBookOrZero 书指针安全取值（nil=零书——系数推导的无书形态）。
func derefBookOrZero(book *rolestate.NeiGongBook) rolestate.NeiGongBook {
	if book == nil {
		return rolestate.NeiGongBook{}
	}
	return *book
}

// refreshNeiGongBooks 按目录刷新全部书的 varprop 效果（2433 线
// restoreFaculty → refreshAllModernEffects 口径）：五维 Add 与
// BuffID/BuffLevel 随（StaticData, Level）档行重取，档缺保持存档值
// （StarterSet 兜底）；BuffStaticData 空时从 buff_new.ini 补。登录装载
// （storeLoadEnter）与 215 换书共调。返回是否有变化（登录侧触发存档）。
func refreshNeiGongBooks(catalog *neigongdata.Catalog, st *rolestate.RoleState) bool {
	if catalog == nil {
		return false
	}
	changed := false
	for i := range st.NeiGong {
		book := &st.NeiGong[i]
		// 五行随目录补（内修面板按 FacultyName QueryProp("WuXing")——
		// 0 值查表失败面板整列退原始键显示；不依赖 varprop 档行）。
		if book.WuXing == 0 {
			if w := catalog.WuXingOf(book.ConfigID); w != 0 {
				book.WuXing, changed = int(w), true
			}
		}
		row, ok := catalog.VarPropRow(int32(book.StaticData), int32(book.Level))
		if !ok {
			continue
		}
		changed = applyVarPropRow(book, row, catalog) || changed
	}
	return changed
}

// applyVarPropRow 档行落到书（facultyLevelUpLoop 升级同口径）：五维/
// buff 全量重取；资源 Add（MaxHP/MaxMP/MaxParry/MinMagicDef）**仅补零**——
// 非零存档快照=官服建号对拍值（首三本 30 级 1630/4050 系），proppack
// 合成位不覆盖之。返回是否有变化。
func applyVarPropRow(book *rolestate.NeiGongBook, row neigongdata.VarProp, catalog *neigongdata.Catalog) bool {
	changed := false
	if book.StrAdd != int(row.StrAdd) || book.StaAdd != int(row.StaAdd) ||
		book.DexAdd != int(row.DexAdd) || book.IngAdd != int(row.IngAdd) ||
		book.SpiAdd != int(row.SpiAdd) {
		changed = true
	}
	book.StrAdd, book.StaAdd, book.DexAdd = int(row.StrAdd), int(row.StaAdd), int(row.DexAdd)
	book.IngAdd, book.SpiAdd = int(row.IngAdd), int(row.SpiAdd)
	if row.BufferID != "" {
		if book.BuffID != row.BufferID || book.BuffLevel != int(row.BufferLevel) {
			changed = true
		}
		book.BuffID, book.BuffLevel = row.BufferID, int(row.BufferLevel)
	}
	// 资源 Add 补零（档行 proppack 合成面；快照非零不覆盖）。
	if book.MaxHPAdd == 0 && row.MaxHPAdd != 0 {
		book.MaxHPAdd, changed = int(row.MaxHPAdd), true
	}
	if book.MaxMPAdd == 0 && row.MaxMPAdd != 0 {
		book.MaxMPAdd, changed = int(row.MaxMPAdd), true
	}
	if book.MaxParryAdd == 0 && row.MaxParryAdd != 0 {
		book.MaxParryAdd, changed = int(row.MaxParryAdd), true
	}
	if book.MinMagicDefAdd == 0 && row.MinMagicDefAdd != 0 {
		book.MinMagicDefAdd, changed = int(row.MinMagicDefAdd), true
	}
	// BuffStaticData 随当前 BuffID 重取（换 buff 档后旧静态位是脏值——
	// StarterSet 兜底静态位对应建号 buff，不是档行 buff）。
	if book.BuffID != "" && catalog != nil {
		if static, ok := catalog.BuffStaticData(book.BuffID); ok && int(static) != book.BuffStaticData {
			book.BuffStaticData, changed = int(static), true
		}
	}
	return changed
}

// handleUseNeiGong C2S 215 运行内功切换（文件头帧序）。
func (s *Server) handleUseNeiGong(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomUseNeiGong) {
		return
	}
	if len(message.Values) < 2 || message.Values[1].Type != 6 || strings.TrimSpace(message.Values[1].Text) == "" {
		s.trace("neigong use: 215 需要 ConfigID string client=%d", pl.Conn1)
		return
	}
	configID := strings.TrimSpace(message.Values[1].Text)
	st := &pl.roleState
	book := equipBookOf(st, configID)
	if book == nil {
		s.trace("neigong use: 未学内功 %s 拒收 client=%d", configID, pl.Conn1)
		return
	}
	catalog := s.neiGongCatalogOf()
	previous := st.CurNeiGong
	st.CurNeiGong = configID
	id := pl.entergenIdentity()
	// 0. 书 varprop 档随目录重取（Add/BuffID 与登录刷新同源——书升级
	//    只在 facultyLevelUpLoop 内重取，换书前可能还是 StarterSet 兜底值）。
	refreshNeiGongBooks(catalog, st)
	// 战斗权威跟新书（2433 equipNeiGong → SetDerivedMaxima 口径）：上限
	// 随五维系数推导（entergen DerivedMax*），面板缓存作废重建；当前
	// 血/蓝/怒**保留现值钳进新上限**（高值换低值=新低值为准；低值换
	// 高值=旧值不变，不回满——2433 equipNeiGong 怒气钳制同口径）。
	combat := &pl.combat
	combat.mu.Lock()
	pl.panel = nil
	pushed := false
	var curHP, curMP, curSP, curMaxHP int
	pushMP := false
	if combat.live.ready {
		pushed = true
		if maxHP := int(entergen.DerivedMaxHP(*st, *book)); maxHP > 0 {
			combat.live.maxHP = maxHP
			combat.live.hp = clampInt(combat.live.hp, 1, maxHP)
		}
		if maxMP := int(entergen.DerivedMaxMP(*st, *book)); maxMP > 0 {
			combat.mp = clampInt(combat.mp, 0, maxMP)
		}
		if maxSP := int(entergen.RageCapOf(configID)); maxSP > 0 {
			combat.live.maxSP = maxSP
			combat.live.sp = clampInt(combat.live.sp, 0, maxSP)
		}
		curHP, curMaxHP, curSP = combat.live.hp, combat.live.maxHP, combat.live.sp
		if combat.mpReady {
			curMP, pushMP = combat.mp, true
		}
	}
	combat.mu.Unlock()

	// 1. MaxSP 专帧：换书改怒气有效上限，先刷分母再发资源批。
	if frame, err := playerPropertyFrame(pl,
		entergen.NamedProperty("MaxSP", entergen.Int32Value(entergen.RageCapOf(configID)))); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("neigong use: MaxSP 专帧编码失败: %v", err)
	}

	// 1.5 换书资源重推：钳进新上限后的当前血/蓝/怒按单属性帧下发
	//（HPRatio 按新上限现算——低换高时 <100；没进过战斗的会话无生存
	// 态，跳过；会话未用过蓝不发 MP——ensureVitals 首用时自存档取）。
	if pushed {
		b := &frameBuilder{ident: pl.wireIdent()}
		b.prop("HPRatio", entergen.Int32Value(hpRatio(curHP, curMaxHP)))
		b.prop("HP", entergen.Int32Value(int32(curHP)))
		if pushMP {
			b.prop("MP", entergen.Int32Value(int32(curMP)))
		}
		b.prop("SP", entergen.Int32Value(int32(curSP)))
		for _, frame := range b.frames {
			s.sendToClient(pl, frame)
		}
	}

	// 2. 资源批：CurNeiGong/五行/FacultyName + BufferInfo4 动态合成
	//（旧书 buff 图标随批刷新消失，新书静态位就位）。
	props := []entergen.IndexedProperty{
		entergen.NamedProperty("CurNeiGong", entergen.StringValue(configID)),
		entergen.NamedProperty("FacultyName", entergen.StringValue(configID)),
		entergen.NamedProperty("CurNeiGongWuXing", entergen.Int32Value(catalog.WuXingOf(configID))),
	}
	buffInfo4 := equipBuffInfo4(catalog, id, book)
	props = append(props, entergen.NamedProperty("BufferInfo4", entergen.StringValue(buffInfo4)))
	if frame, err := playerPropertyFrame(pl, props...); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("neigong use: 资源批编码失败: %v", err)
	}

	// 3. 战斗面板单属性逐帧（五维/资源 Add 随新书面板值）。
	for _, spec := range entergen.CombatAttributeValues(*st) {
		if frame, err := playerPropertyFrame(pl,
			entergen.NamedProperty(spec.Name, entergen.Int32Value(spec.Value))); err == nil {
			s.sendToClient(pl, frame)
		}
	}

	// 4. m169 两值帧：登记新书可见 buff（图标只由此帧驱动）。
	if book.BuffID != "" {
		if frame, err := entergen.EquipBuffFrame(id, book.BuffID); err == nil {
			s.sendToClient(pl, frame)
		} else {
			s.trace("neigong use: m169 buff 帧编码失败 %s: %v", book.BuffID, err)
		}
	}
	s.sendRoleSave(pl)
	s.trace("neigong use: 运行内功 %s → %s（五行 %d）client=%d",
		previous, configID, catalog.WuXingOf(configID), pl.Conn1)
}

// equipBuffInfo4 装备书的 BufferInfo4 槽串（无 buff/缺 StaticData 空串）。
func equipBuffInfo4(catalog *neigongdata.Catalog, id entergen.Identity, book *rolestate.NeiGongBook) string {
	if book == nil || book.BuffID == "" || catalog == nil {
		return ""
	}
	static, ok := catalog.BuffStaticData(book.BuffID)
	if !ok {
		return ""
	}
	return entergen.BufferInfo4CSV(id, static, int32(book.BuffLevel))
}
