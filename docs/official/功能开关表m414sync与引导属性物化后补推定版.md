# 功能开关表 m414 sync 与引导属性物化后补推（定版）

> 来源：2026-11-28 排查「引导地图的任务没有触发引导步骤和语音」＋「点 NPC 怎么触发语音」。
> 状态：**真机已验证（2026-11-28 09:07：语音出、m950 链通）；未提交。** 首次部署 go2
> `bin\game_solounite.exe` 2026-11-28 03:16（干净导出编译）。
> 相关：选择窗触发帧 `docs/verified/新手引导选择窗触发帧m427.md`；任务协议 `docs/official/NPC任务绑定与引导流任务协议定版.md`。

## 一、结论

| 现象 | 根因 | 修法 |
|---|---|---|
| 点 NPC 没有语音（同客户端连官服／AOW 服有） | 本服从未下发功能开关表 **m414 `["sync", 开关号…]`**。客户端引擎 `HeadGame:PlaySoundByFile`（NPC 对话语音）入口闸门是 `SwitchManager::CheckSwitchEnable(0x5e5)`，开关树为空则恒假 | 进场 0x0B 之后下发官服开关全表（`internal/entergen/switchsync.go`，641 项） |
| 接取影像播完客户端不发 m950、引导步骤链（亮按钮／提示）不动 | `HelpStepID` 只在 **0x0B 之前**单发过一次；进场会重建玩家对象，场内对象只认 **0x0D 物化帧的属性区**——那份被冲掉，客户端 `HelpStepID` 为空，`freshman_help.movie_end_callback` 首段即返回 | 0x0D／0x1F 之后补推 `HelpStepID`／`FinishFreshmanRecStr`／`BornServerGameStep`（`entergen.GuideMaterializeFrames`，进场与切图两条序列都接） |

两项都是**服务端少发**，客户端与资源没有问题。

## 二、证据

### 2.1 语音链（引擎反汇编，`fxgamelogic.dll` CN 2026）

- `form_talk_movie.show_window` → `HeadGame:PlaySoundByFile(title_id, npcid)`（`tmp/lua_dec` 反编译 lua；`title_id` 即 m5 的 `task_<ConfigID>` 键）。`on_endmenu` 选窗规则：NPC 的 `NpcTalkType == 1` 开 `form_talk`（无语音窗），否则开 `form_talk_movie`。
- 引擎侧（`.text+0x3424e0`，lua 名 `PlaySoundByFile`）：① `SwitchManager->CheckSwitchEnable(0x5e5)` 为假即返回 → ② `form_freshman_voice` 已开即返回 → ③ `title_id` 非空 → ④ 场景对象 `Type == 4` 且 `ConfigID` 存在 → ⑤ 查 `snd\env\dub\sound_npctalk.ini` 的 `[NpcTalk]`（键＝`task_<ConfigID>`，引导 NPC `funnpcborn003031` → `man6_05.ogg`）→ 开语音窗播放。
- `CheckSwitchEnable` = `SwitchManager` 内一张 `std::map<int, _>` 的 `find`。这张表**只**由 S2C m414 的同一个原生 handler 维护（`.text+0x828a50`）：
  - `["gamestep", 主步, 子步]` → 存主步／子步（引擎全局态，与开关表无关）；
  - `["sync", 开关号…]` → **清空开关表**后按下发开关号逐个置开；
  - `["reset", 号, 开/关, …]` → 成对增删并回调 lua `util_switch_function.on_query_switch`。
- 本服五个真机会话（50350／54905／59258／63857／53558）下行零 `sync`、零 `reset`。

### 2.2 官服开关表（CN 2026）

- 官服位（初始账号登陆-完整引导-女）：`… → m414 gamestep(#448) → 0x0B(#476) → 0x14 → op 0x27 压缩批(#479，内层即本消息，643 值) → 0x28(玩家 0x0D, #487) → 0x29 记录批…`。每会话一次。
- 表 641 项、严格升序、全部类型字节 2（int32），**11 份官服抓包逐值一致**；含 `0x5e5`（NPC 对话语音）与 `2919`（新版人物界面按钮）。全表见 `internal/entergen/switchsync.go`，golden 见 `internal/entergen/testdata/official_switch_sync.bin`。
- **解码方法**：官服把本消息经 fxnet2 LZ 压进 op 0x27 批。模拟解码器（`tmp/shortcutpos/lzdec2.py` 一族）在本帧**不可信**——本帧有池位碰撞：`02 30 00 00`（id 48 的类型字节窗口）、`02 10 00 00`、`02 70 00 00` 分别哈希到 1／3／5 号池位，顶掉段首匹配源，解出 4 段共 274 个垃圾值。标准答案用**客户端 `fxnet2.dll` 的真解码器**取（`tmp/npctalk/reallz.py`：ctypes 载入，按 RVA 0x11f0 包装函数签名喂 blob，池跨帧共享须从连接头按序喂，遇池缺失即停）。

### 2.3 引导属性（m950 链）

- 客户端链：`form_movie_new.end_movie` 发 m806 后无条件调 `freshman_help.movie_end_callback`；它首段 `player_client:QueryProp("HelpStepID")` 为空或 `Freshman_End` 即返回；步骤 `type=7`（播动画＋黑幕）才发 m950 `[0, 下一步]`。
- 官服 `HelpStepID` 写两次：0x0B 前 0x10（#469 `chengdu_00`）与 **0x0D 属性区内**（#487 的 0x28 批内层 126 项属性，含 `HelpStepID=chengdu_00`、`FinishFreshmanRecStr`、`BornServerGameStep=45`），m950 之后再一次 0x10（#1927）。
- 官服在 0x0B 前用 0x10 单发过的属性（AvatarCount、KarmaUpperLimit、LevelTitle、MaxFaculty、各 Faculty 值…）在 0x0D 属性区**全部重发**——说明进场重建玩家对象，0x0D 属性区才是场内属性的权威载体（与键位域旧结论同源：`appearance.go`「引擎建对象时即读整个键位域，0x10 补推晚于建对象」）。
- 本服 0x0D 沿 fork 样本属性集（187 项），不含这三项；0x0B 前的单发是唯一一份。五个真机会话客户端零 m950。

## 三、实现

- `entergen.SwitchSyncFrame(ids)` → m414 `["sync", id×N]` 明文 0x1E（去外壳后与官服批内层逐字节相同，`TestSwitchSyncFrameMatchesOfficialBatch`）；`entergen.OfficialSwitchIDs` 官服全表；`Context.SwitchSyncIDs`（nil＝不发）。
- `entergen/sequence.go`：0x0B 之后、0x0D 之前追加 sync；0x1F 之后追加 `GuideMaterializeFrames`。`fssoloworld/sceneswitch.go`：切图重入 0x0D 之后同样补推引导属性（不重发 sync——连接级一次）。
- 开关：`NINEYIN_SWITCH_SYNC` 未设＝官服全表；`off`＝不发（A/B 对照）；其余＝逗号分隔开关号表。
- 守卫：`TestOfficialSwitchIDsShape`（641 项严格升序、含 0x5e5／2919）、`TestPreReadyFramesCarryGuidePropertiesAfterMaterialize`、`fssoloworld` 三个位次用例（进场 sync 紧随 0x0B、HelpStepID 在 0x0D 后、切图补推且不重发 sync）。

## 四、真机验证记录

2026-11-28 09:07 会话 65444（线路抓取）：**两项都生效**。

- 语音：用户实测「npc语言有了」；抓包里进场簇 `0x0B → m414 sync(3223 字节) → 0x0D`。
- m950 链：影像 473 结束客户端上行 `950 [0, "yanyuzhuang_02"]`，服务端回
  `0x10 HelpStepID="yanyuzhuang_02"`（#249）——与官服 #1926/#1927 同形。
- 「引导提示」仍无：卡点后移到 1347 发书之后，是另一条链（S2C m371 帮助窗，
  见 `docs/official/新手引导提示m371帮助窗定版.md`）。

## 五、遗留

- 本服 0x0D 属性集仍是 fork 样本的 187 项；官服 CN 2026 是 126 项且集合不同（多 `VisualPlayers`、`FestivalVersionID`、`VipCreateTime`、`Book`、`SelectSchool`、`BodyType`、`PKMode`、`FightData`、体力／轻功明细…，少战斗扩展列）。本次只做补推，没有把 0x0D 整体对齐官服——`appearance.go` 另有会话在改。
- 官服进场另有两个压缩批未对齐：m485（结婚坐骑表 100 值）、m368（营业时间表 231 值）。
- fxnet2 LZ 模拟解码器与真解码器的池语义差异未定案（真解码器能正确处理上述碰撞、模拟器不能）；需要解 LZ 批帧时一律走 `tmp/npctalk/reallz.py`。
- 切图后若引导步骤链仍有丢失（0x0C 重入路径之外的重建点），按同法补推。
