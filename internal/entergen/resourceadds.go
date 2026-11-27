package entergen

import (
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// resourceadds.go 资源 Add 五维系数推导（官方 wire 公式——docs/official/
// 九阴建角初始数据归纳.md §3.2「官方 55866/12173 抓包逐点解出」，2433 线
// player_progress.go attrHPPerStr 系数族同源移植）：
//
//	MaxHPAdd@583       = 2·Str + 7·Sta + 装备书资源分量
//	MaxMPAdd@584       = 4·Ing + 1·Spi + 装备书资源分量
//	MinMagicDefAdd@599 = round(0.2·Spi) + 装备书资源分量
//	MaxParryAdd@479    = 装备书资源分量（无系数分量）
//
// 五维 = 白板 + 装备书 varprop Add；书资源分量 = 存档快照/proppack
//（neigonguse.go applyVarPropRow 补零口径——快照 1630/490/840/45 系
// 官服 52626 对拍值，系数层 577/347 与其相加恰等于官方帧 2207/837）；
// 经脉/镇派分量本线未做（0）。
//
// 登录 0x0D/0x10、战斗生存态上限（ensureLiveLocked/playerMaxMP）与 215
// 换书面板必须共用同一取值面——协议与服务器权威同源，客户端显示总量
// 才能与服务器消耗/恢复用的总上限一致（2433 SetDerivedMaxima 同诫）。

// 五维→资源系数（官方抓包解出；2433 attrHPPerStr 族同名同值）。
const (
	attrHPPerStr int32 = 2
	attrHPPerSta int32 = 7
	attrMPPerIng int32 = 4
	attrMPPerSpi int32 = 1
)

// ResourceAdds 推导后的资源 Add 四件套（0x10 面板同名属性直发）。
type ResourceAdds struct {
	MaxHPAdd       int32
	MaxMPAdd       int32
	MaxParryAdd    int32
	MinMagicDefAdd int32
}

// EffectiveFive 线上五维总值（白板 + 装备书 varprop Add——0x10 面板
// 485-489 同值，官方 wire 即总值；经脉分量未做）。站立回复速率的取值面。
func EffectiveFive(state rolestate.RoleState, book rolestate.NeiGongBook) (str, sta, dex, ing, spi int32) {
	return int32(state.Str) + int32(book.StrAdd),
		int32(state.Sta) + int32(book.StaAdd),
		int32(state.Dex) + int32(book.DexAdd),
		int32(state.Ing) + int32(book.IngAdd),
		int32(state.Spi) + int32(book.SpiAdd)
}

// DerivedResourceAdds 按装备书推导资源 Add（vitalBook 同口径选书：
// CurNeiGong 匹配，缺席回退首本；零书全系数 0——书资源分量无）。
func DerivedResourceAdds(state rolestate.RoleState, book rolestate.NeiGongBook) ResourceAdds {
	str, sta, _, ing, spi := EffectiveFive(state, book)
	return ResourceAdds{
		MaxHPAdd:       str*attrHPPerStr + sta*attrHPPerSta + int32(book.MaxHPAdd),
		MaxMPAdd:       ing*attrMPPerIng + spi*attrMPPerSpi + int32(book.MaxMPAdd),
		MaxParryAdd:    int32(book.MaxParryAdd),
		MinMagicDefAdd: spiTenthsAdd(spi) + int32(book.MinMagicDefAdd),
	}
}

// HPRegenPerSecond / MPRegenPerSecond 站立回复速率（官方逐点表：体魄
// 每 5 秒 +0.07 生命、罡气每 5 秒 +0.2 蓝 → 每秒小数速率，服务端小数
// 累积进整数；面板 Add 列 round 同源公式，两端同速）。
func HPRegenPerSecond(sta int32) float64 { return float64(sta) * 7 / 500 }
func MPRegenPerSecond(spi int32) float64 { return float64(spi) / 25 }

// DerivedMaxHP/MP 总上限（白板基数 + 推导 Add）——战斗生存态与帧内
// HP/MP 满值口径共用。
func DerivedMaxHP(state rolestate.RoleState, book rolestate.NeiGongBook) int32 {
	return int32(state.HP) + DerivedResourceAdds(state, book).MaxHPAdd
}

func DerivedMaxMP(state rolestate.RoleState, book rolestate.NeiGongBook) int32 {
	return int32(state.MP) + DerivedResourceAdds(state, book).MaxMPAdd
}
