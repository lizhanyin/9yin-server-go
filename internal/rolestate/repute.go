package rolestate

// repute.go 声望与历练的累加面（任务完成奖励；官服引导抓包 #1933/#2409：
// Repute_Record 行＝名称、数值、标志位，新声望加行、已有声望逐格写数值；
// #1938：历练写玩家属性 LiveGroove_<下标>）。历练域口径＝2433 线
// docs/systems/livepoint-conversion.md：毫历练存储、1 点历练=1000 修为、
// 槽 5 个每槽上限 999 点、转化速度 10/20/50/100/200 修为/100 秒、普通档
// 70% 进修为 30% 进潜能。

// LiveGrooveCap 历练槽上限（毫历练：999 点 ×1000；2433 线 grooveCap 同值）。
const LiveGrooveCap int32 = 999_000

// AddRepute 声望累加：已有同名记录累加数值（added=false，返回行下标与新
// 数值）；没有则追加一条（added=true）。
func (s *RoleState) AddRepute(name string, count int32) (row int, total int32, added bool) {
	for i := range s.Reputes {
		if s.Reputes[i].Name == name {
			s.Reputes[i].Count += count
			return i, s.Reputes[i].Count, false
		}
	}
	s.Reputes = append(s.Reputes, Repute{Name: name, Count: count})
	return len(s.Reputes) - 1, count, true
}

// AddLiveGroove 历练累加（下标 1..5，毫历练；钳槽上限 999 点），返回新
// 余额；下标越界不累加。扣减穿底钳 0（现行调用方只加不减，防御口径）。
func (s *RoleState) AddLiveGroove(index int, count int32) (int32, bool) {
	if index < 1 || index >= len(s.LiveGrooves) || count == 0 {
		return 0, false
	}
	s.LiveGrooves[index] += count
	if s.LiveGrooves[index] > LiveGrooveCap {
		s.LiveGrooves[index] = LiveGrooveCap
	}
	if s.LiveGrooves[index] < 0 {
		s.LiveGrooves[index] = 0
	}
	return s.LiveGrooves[index], true
}

// AddLivePoints 发 N 点历练（×1000 毫按序填槽，溢出丢弃——2433 线
// addLivepoint 同形；GM add_livepoint / 历练药共用）。返回实际入槽毫数。
func (s *RoleState) AddLivePoints(points int32) int32 {
	remaining := points
	if remaining < 0 {
		remaining = 0
	}
	if remaining > 999 { // 单次上限 999 点（2433 GM 入口同限）
		remaining = 999
	}
	remaining *= 1000
	added := int32(0)
	for i := 1; i < len(s.LiveGrooves) && remaining > 0; i++ {
		space := LiveGrooveCap - s.LiveGrooves[i]
		if space <= 0 {
			continue
		}
		put := min(space, remaining)
		s.LiveGrooves[i] += put
		added += put
		remaining -= put
	}
	return added
}
