package entergen

// 0x10 人物数据大批（登录全量面板——b-line loginVitalUpdate 定版移植）。
//
// 源：2433 线 52626 会话 s2c #27（0x10 len=1299/n=160 属性，0x0D→0x1F
// 之后、物理流单帧族之前）；生成器蓝本 = 9yin-server-go-2433
// cmd/protocol-probe player_actor.go stateProperties + loginVitalUpdate
//（Java PlayerActor.stateProperties 同位：头部 vitals → 门派 → BufferInfo
// 槽 → motion → 轻功参数 → progress 内功域/五维/攻防面板 → 经脉 8 件套
// → 尾追 MaxSP 一次声明）。
//
// 关键口径（b-line 注释定版）：
//   - MaxSP(580) 只由此登录全量批携带一次，官方全程零下发 MaxSPAdd(583)
//     ——加成折进 580（值 = RageCap：基础 80 + 运行内功档位加成，ragecap.go）；
//     把上限塞进 0x0D 归档或周期批会让客户端进世界崩溃；
//   - *UpSpeedAdd 位是五维分数系数分量（round(0.07·Sta)/round(0.2·Spi)，
//     12173/55866 抓包），*UpSpeed 基础位 = ceil(总上限/10)；
//   - 格挡族登录补位：CantParry=0/ParryRatio=100/ParryType=0/
//     ParryUpSpeed=4（官方首帧 0x10 同拍必发，之后不再重推）；
//   - ParryValue@475 = MaxParry + MaxParryAdd 推导和（抓包 116/157 吻合）；
//   - ModifySkillLockTime 官方口径 200（Java PlayerActor #2014 反汇编）。
//
// 建号数据源：rolestate 白板五维/资源 + StarterSet 首本书 ng_wd_001
// varprop 加成（与 AppearanceFrame/CombatAttributeFrames 同一推法）。

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// vitalBook 解析当前装备内功（缺席回退零书——与 AppearanceFrame 同法）。
func vitalBook(ctx Context) rolestate.NeiGongBook {
	state := ctx.State
	for i := range state.NeiGong {
		if state.NeiGong[i].ConfigID == state.CurNeiGong {
			return state.NeiGong[i]
		}
	}
	if len(state.NeiGong) > 0 {
		return state.NeiGong[0]
	}
	return rolestate.NeiGongBook{}
}

// wuXingWithBookFallback CurNeiGongWuXing 取值：目录合成值优先，缺行
//（0）回退运行书行 WuXing（官服 ng_jh_001=6 阴柔——面板本地化
// ui_wuxue_prop_N，0 值查表失败显示原码）。
func wuXingWithBookFallback(catalogValue int32, bookWuXing int) int32 {
	if catalogValue != 0 {
		return catalogValue
	}
	return int32(bookWuXing)
}

// PlayerVitalBatchFrame 登录全量人物数据批（单帧 0x10，n≈150+）。
// 官方发送位：0x0D 物化 → 0x1F 位置 → 本批 → 物理流单帧族。
func PlayerVitalBatchFrame(ctx Context) ([]byte, error) {
	state := ctx.State
	book := vitalBook(ctx)
	bookCurNeiGong := state.CurNeiGong
	if bookCurNeiGong == "" {
		bookCurNeiGong = book.ConfigID
	}

	// 面板五维（白板 + 运行书 varprop——官服白板口径 RoleBaseProp）；
	// 资源 Add 走五维系数推导（resourceadds.go——2·Str+7·Sta 系官方
	// 公式层，书资源分量为加数之一）。
	str := int32(state.Str) + int32(book.StrAdd)
	sta := int32(state.Sta) + int32(book.StaAdd)
	dex := int32(state.Dex) + int32(book.DexAdd)
	ing := int32(state.Ing) + int32(book.IngAdd)
	spi := int32(state.Spi) + int32(book.SpiAdd)
	adds := DerivedResourceAdds(state, book)
	maxHP := DerivedMaxHP(state, book)
	maxMP := DerivedMaxMP(state, book)
	maxParry := int32(state.Parry) + adds.MaxParryAdd

	// 官方恢复公式：**基础列 0**——站立回复只走 Add 列五维公式（官方逐点
	// 表：Sta 每 5 秒 +0.07 生命、Spi 每 5 秒 +0.2 蓝；docs/official/
	// 九阴建角初始数据归纳.md §3.1）。旧基础位 ceil(上限/10) 是打坐速率
	// 口径，塞基础列会让客户端站立也按打坐速回（「回血回蓝值高了」病灶）。
	// 打坐回复由服务端权威帧驱动（sitcross.go），不依赖本列。
	hpUpSpeedAdd := staHundredthsAdd(sta)
	mpUpSpeedAdd := spiTenthsAdd(spi)

	objHex := fmt.Sprintf("%X", ctx.Identity.ObjectID)
	ownHex := fmt.Sprintf("%X", ctx.Identity.OwnerID)
	curHP, curMP, hpRatio, mpRatio := currentVitals(ctx, maxHP, maxMP)

	props := []IndexedProperty{}

	// ---- 头部 vitals（stateProperties 前段）----
	props = append(props,
		ap("LogicState", ByteValue(0)), // 52626 #27 首条 idx465 byte 0
		ap("HPRatio", Int32Value(hpRatio)),
		ap("MPRatio", Int32Value(mpRatio)),
		// 客户端用基础值 + MaxHPAdd/MaxMPAdd 推导显示总量。
		ap("MaxHP", Int32Value(int32(state.HP))),
		ap("MaxMP", Int32Value(int32(state.MP))),
		ap("HP", Int32Value(int32(curHP))),
		ap("MP", Int32Value(int32(curMP))),
		ap("SpeedRatio", Int32Value(100)),
		ap("HitHP", Int32Value(int32(state.HP))),
		ap("HitHPRatio", Int32Value(100)),
		ap("QingGongPoint", Int32Value(150)),
		ap("MaxQingGongPoint", Int32Value(150)),
		ap("MaxQingGongPointAdd", Int32Value(0)),
		ap("NeigongPKStatus", WordValue(0)),
		ap("Dead", ByteValue(0)),
		ap("InParry", Int32Value(0)),
		ap("InManualParry", Int32Value(0)),
		// 格挡族登录补位（官方首帧同拍必发，之后不再重推）。
		ap("CantParry", Int32Value(0)),
		ap("ParryRatio", Int32Value(100)),
		ap("ParryType", Int32Value(0)),
		ap("ParryUpSpeed", Int32Value(4)),
		ap("ParryValue", Int32Value(maxParry)), // MaxParry+MaxParryAdd 推导和
		ap("CurSkillID", StringValue("")),
		ap("CurSkillTarget", ObjectValue(0, 0)),
		ap("ModifySkillLockTime", Int32Value(200)), // 官方口径 200（Java #2014）
		ap("HPHeartSpeed", Int32Value(1000)),
		ap("HPHeartSpeedAdd", Int32Value(0)),
		ap("MPHeartSpeed", Int32Value(1000)),
		ap("MPHeartSpeedAdd", Int32Value(0)),
		ap("HPUpSpeed", Int32Value(0)),
		ap("HPUpSpeedAdd", Int32Value(hpUpSpeedAdd)),
		ap("MPUpSpeed", Int32Value(0)),
		ap("MPUpSpeedAdd", Int32Value(mpUpSpeedAdd)),
		ap("Name", WideValue(ctx.RoleName)),
		// 货币三币（2433 线 currenciesUpdate 口径：碎银 1 / 官银 2 / 黄金 0）。
		ap("CapitalType1", Int64Value(state.Money)),
		ap("CapitalType2", Int64Value(state.OfficialSilver)),
		ap("CapitalType0", Int64Value(state.Gold)),
	)

	// BufferListStr：2433 表在名（fork 线外服表无此名停发——方言门控按
	// Must* 编码期报错兜底），值与 0x0D 归档同源（按运行书动态）。
	props = append(props, ap("BufferListStr", StringValue(bufferListString(ctx))))

	// ---- 门派域（建号无门派：三空串）----
	props = append(props,
		ap("School", StringValue("")),
		ap("Force", StringValue("")),
		ap("NewSchool", StringValue("")),
	)

	// ---- BufferInfo 24 槽（CSV 条；登录 buff 与 0x0D 归档同源）。
	// 槽 4=装备内功 buff——按 CurNeiGong 动态（ctx.BuffInfo4，旧版写死
	// ng_wd_001 的 1742 槽条）。----
	bufferInfos := map[int]string{
		1: fmt.Sprintf("6747,%s,%s,1,0,0", objHex, ownHex),
		2: fmt.Sprintf("1E37,%s,%s,0,0,0", objHex, ownHex),
	}
	if ctx.BuffInfo4 != "" {
		bufferInfos[4] = ctx.BuffInfo4
	}
	for slot := 1; slot <= 24; slot++ {
		props = append(props, ap(fmt.Sprintf("BufferInfo%d", slot), StringValue(bufferInfos[slot])))
	}

	// ---- motion 三件（物理流随后覆盖同名——官方同构）----
	props = append(props,
		ap("Gravity", Float32Value(9.8)),
		ap("GravityAdd", Float32Value(0)),
		ap("DropHeightPub", Float32Value(100000)),
	)

	// ---- 轻功/冲刺参数 28 条（qinggongMotionProperties 同值）----
	props = append(props,
		ap("RunSpeedAdd", Float32Value(0)), ap("JumpSpeedAdd", Float32Value(0)),
		ap("DriftSpeed", Float32Value(7.5)), ap("DriftSpeedAdd", Float32Value(0)),
		ap("DriftJumpSpeed", Float32Value(6.0)), ap("DriftJumpSpeedAdd", Float32Value(0)),
		ap("SndJumpSpeed", Float32Value(6.0)), ap("SndJumpSpeedAdd", Float32Value(0)),
		ap("ThdJumpSpeed", Float32Value(6.0)), ap("ThdJumpSpeedAdd", Float32Value(0)),
		ap("AirRushSpeed", Float32Value(15)), ap("AirRushSpeedAdd", Float32Value(0)),
		ap("AirRushDist", Float32Value(12)), ap("AirRushDistAdd", Float32Value(0)),
		ap("LandRushSpeed", Float32Value(25)), ap("LandRushSpeedAdd", Float32Value(0)),
		ap("LandRushDist", Float32Value(6.5)), ap("LandRushDistAdd", Float32Value(0)),
		ap("ClimbSpeed", Float32Value(7.5)), ap("ClimbSpeedAdd", Float32Value(0)),
		ap("GRushRange", Float32Value(5.0)), ap("GRushRangeAdd", Float32Value(1.5)),
		ap("ARushRange", Float32Value(8.0)), ap("ARushRangeAdd", Float32Value(6.5)),
		ap("WRushRange", Float32Value(8.0)), ap("WRushRangeAdd", Float32Value(6.5)),
	)

	// ---- progress：内功书域 + 历练/团练 + 五维 + 攻防面板 ----
	// 修炼族绑定存档（官服引导女 t=546..557：FacultyName 选书＋State=2
	// 转换中＋FillSpeed=200；FacultyName 空档回退运行书，进度三值跟修炼书
	//——与 appearance.go 同口径）。
	facultyName := state.FacultyName
	if facultyName == "" {
		facultyName = bookCurNeiGong
	}
	cultivated := book
	for i := range state.NeiGong {
		if state.NeiGong[i].ConfigID == facultyName {
			cultivated = state.NeiGong[i]
			break
		}
	}
	props = append(props,
		ap("CurNeiGong", StringValue(bookCurNeiGong)),
		ap("Faculty", Int32Value(int32(state.Faculty))),
		ap("FacultyState", Int32Value(state.FacultyState)),
		ap("FacultyStyle", Int32Value(state.FacultyStyle)),
		ap("FacultyName", StringValue(facultyName)),
		ap("CurFillValue", Int32Value(int32(cultivated.Fill))),
		ap("TotalFillValue", Int32Value(int32(cultivated.Total))),
		ap("CurLevel", Int32Value(int32(cultivated.Level))),
		ap("FillSpeed", Int32Value(state.FacultyFillSpeed)),
		// 历练层 5 条（角色态累计，任务完成奖励累加；建号全零）+ 转化加成
		// + 潜能存量。
		ap("LiveGroove_1", Int32Value(state.LiveGrooves[1])),
		ap("LiveGroove_2", Int32Value(state.LiveGrooves[2])),
		ap("LiveGroove_3", Int32Value(state.LiveGrooves[3])),
		ap("LiveGroove_4", Int32Value(state.LiveGrooves[4])),
		ap("LiveGroove_5", Int32Value(state.LiveGrooves[5])),
		ap("LiveSpeedPower", Int32Value(0)),
		ap("HoldFaculty", Int32Value(0)),
		// 团练疲劳层。
		ap("TeamFacultyState", Int32Value(0)),
		ap("TeamFacultyValue", Int32Value(0)),
		ap("TeamFacultyLast", Int32Value(0)),
		ap("PowerValue", Int32Value(int32(book.Power))),
		ap("MaxPowerValue", Int32Value(int32(book.MaxPower))),
		ap("SP", Int32Value(int32(ctx.SP))),
		// 五维面板（白板 + 书）。
		ap("Str", Int32Value(str)),
		ap("Sta", Int32Value(sta)),
		ap("Dex", Int32Value(dex)),
		ap("Ing", Int32Value(ing)),
		ap("Spi", Int32Value(spi)),
		// 攻击面板 + Add 列（Add = 对应维官方公式分量）。
		ap("MeleePower", Int32Value(80)),
		ap("ShotPower", Int32Value(80)),
		ap("MagicPower", Int32Value(80)),
		ap("MeleePowerAdd", Int32Value(str)),
		ap("ShotPowerAdd", Int32Value(dex)),
		ap("MagicPowerAdd", Int32Value(ing)),
		ap("PhyHit", Int32Value(50)),
		ap("PhyHitAdd", Int32Value(dex)),
		ap("MagicHit", Int32Value(50)),
		ap("MagicHitAdd", Int32Value(spi)),
		ap("Dodge", Int32Value(35)),
		ap("DodgeAdd", Int32Value(dex)),
		ap("PhyVa", Int32Value(16)),
		ap("PhyVaAdd", Int32Value(str)),
		ap("MagicVa", Int32Value(16)),
		ap("MagicVaAdd", Int32Value(spi)),
		ap("MaxParry", Int32Value(int32(state.Parry))),
		ap("PhyDef", Int32Value(60)),
		ap("StiDef", Int32Value(50)),
		ap("NegDef", Int32Value(50)),
		ap("MasDef", Int32Value(50)),
		ap("JujDef", Int32Value(50)),
		ap("MinMeleeDamage", Int32Value(20)),
		ap("MaxMeleeDamage", Int32Value(35)),
		ap("MaxHPAdd", Int32Value(adds.MaxHPAdd)),
		ap("MaxMPAdd", Int32Value(adds.MaxMPAdd)),
		ap("MaxParryAdd", Int32Value(adds.MaxParryAdd)),
		ap("MinMagicDefAdd", Int32Value(adds.MinMagicDefAdd)),
		ap("MaxLevel", Int32Value(int32(book.MaxLevel))),
		// 按 CurNeiGong 目录取；目录缺行回退运行书行 WuXing
		//（refreshNeiGongBooks 已随目录补进书档——两处都空才是真 0）。
		ap("CurNeiGongWuXing", Int32Value(wuXingWithBookFallback(ctx.NeiGongWuXing, book.WuXing))),
		ap("WuJiPoint", Int32Value(0)),
		ap("MaxWuJiPoint", Int32Value(0)),
	)

	// ---- 经脉 8 件套（建号零经脉）----
	props = append(props,
		ap("CurJingMai", StringValue("")),
		ap("LastJingMai", StringValue("")),
		ap("JingMaiTotalLevel", Int32Value(0)),
		ap("jmActCount", Int32Value(3)),
		ap("ZhenQiDayValue", Int32Value(500)),
		ap("ZQPLValue", Int32Value(0)),
		ap("ZQActlValue", Int32Value(0)),
		ap("ZQUnUsedValue", Int32Value(0)),
	)

	// ---- 尾追：MaxSP 全量批一次声明（官方全程零下发 MaxSPAdd）----
	props = append(props, ap("MaxSP", Int32Value(RageCap(ctx))))

	return ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0, props)
}

// sitcrossRecovery 恢复基础位：ceil(总上限/10)（10s 回满）。
func sitcrossRecovery(maximum int32) int32 {
	if maximum <= 0 {
		return 0
	}
	return (maximum + 9) / 10
}

// currentVitals 本次进场当前血/蓝（ctx.CurHP/CurMP ≤0＝满值；超上限
// 钳制——运行书换小上限时防 >100%）。返回值与 ratio 供 0x0D/0x10 两帧
// 同源取用。
func currentVitals(ctx Context, maxHP, maxMP int32) (hp, mp int, hpRatio, mpRatio int32) {
	hp, mp = int(maxHP), int(maxMP)
	if ctx.CurHP > 0 && ctx.CurHP < hp {
		hp = ctx.CurHP
	}
	if ctx.CurMP > 0 && ctx.CurMP < mp {
		mp = ctx.CurMP
	}
	return hp, mp, vitalRatio(hp, int(maxHP)), vitalRatio(mp, int(maxMP))
}

// vitalRatio 百分比（上限非正防零除——0x0D 缺省口径）。
func vitalRatio(cur, max int) int32 {
	if max <= 0 {
		return 100
	}
	ratio := cur * 100 / max
	if ratio > 100 {
		ratio = 100
	}
	return int32(ratio)
}

// staHundredthsAdd / spiTenthsAdd 官方分数系数 Add 位：
// round(0.07·Sta) / round(0.2·Spi)（12173/55866 抓包公式）。
func staHundredthsAdd(sta int32) int32 { return roundShifted(7*sta, 100) }
func spiTenthsAdd(spi int32) int32     { return roundShifted(2*spi, 10) }

func roundShifted(numerator, denominator int32) int32 {
	if numerator <= 0 {
		return 0
	}
	return (numerator + denominator/2) / denominator
}
