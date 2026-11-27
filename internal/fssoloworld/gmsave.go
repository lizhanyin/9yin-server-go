package fssoloworld

import (
	"encoding/binary"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// sendRoleSave 0x04 存档（gm save 命令 / 下线 entryClientClosed / 位置
// 2s 节流三源）：roleState 内存态（GM 热改后的权威）Marshal 成 RST1 json
// 发 store。store 端 merge 库内 meta 后落 nx_roles（role_state_json 覆盖）。
// V1 不做 PlSaving 状态流转与周期 ticker（手动+下线先行）。
// 存档前把战斗会话态当前血/蓝写回存档（syncLiveVitals——调用方均不持
// combatState.mu）。
func (s *Server) sendRoleSave(pl *Player) bool {
	s.syncLiveVitals(pl)
	rst1, err := rolestate.MarshalRoleState(pl.roleState)
	if err != nil {
		s.trace("gm save marshal failed client=%d: %v", pl.Conn1, err)
		return false
	}
	payload := buildSaveStore(pl.Conn1, pl.Conn2, pl.Account, pl.Name, pl.roleState.Position, []byte(rst1))
	return s.inner.SendToStore(0, 0, 0x04, payload)
}

// syncLiveVitals 战斗会话态当前血/蓝写回存档（重登持久化的写侧）：live.hp
// → CurHP、st.mp → CurMP。未进战斗/未用蓝＝会话态未建，保持存档原值
//（首次进战斗 ensureLiveLocked/ensureVitals 又从存档取，闭环）。零值不写
//（0 在读侧语义是「未持久化按满值」）。
func (s *Server) syncLiveVitals(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	st := &pl.combat
	st.mu.Lock()
	hp, hpReady := st.live.hp, st.live.ready
	mp, mpReady := st.mp, st.mpReady
	st.mu.Unlock()
	if hpReady && hp > 0 {
		pl.roleState.CurHP = hp
	}
	if mpReady && mp > 0 {
		pl.roleState.CurMP = mp
	}
}

// storeSaveReply 0x03 回执 {client_id, serial, result}（官方
// MessageSave → 0x03 0x58B）：簿记面，trace 即可（玩家可能已离线）。
func (s *Server) storeSaveReply(id1 uint32, p []byte) bool {
	if len(p) < 12 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	result := binary.LittleEndian.Uint32(p[8:])
	s.trace("(SoloWorld::StoreMsgSave)save reply client=%d result=%d", clientID, result)
	return true
}
