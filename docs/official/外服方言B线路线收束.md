# 外服方言 B 线路线收束（2026-11-14）

> **结论一句话**：外服家族客户端（F:\Game\Online\9yin\9yin）的实体渲染缺口
> 已经走通全链——进场结构（单次进场 + ClientReady 两相位）→ 0x0B 官方
> 4 属性形（**idx1468 MaxVisuals=230 是场景可视表容量 = AddObject 注册闸，
> 历史恒缺发是所有对象帧全丢的最终根因**）→ 实体帧（外服 wire 原字节模板
> 库实证渲染，生成器 19 字段定版已修齐待验）→ 我方数据 overlay。
> 当前生产格局：**0x0B 全代码化 + 外服帧模板库喂实体 + 我方数据接管。**

承接：`docs/official/外服模式NPC渲染收束.md`（fork 客户端 A 线）、
`docs/official/外服0x29批量预载通道解析.md`、`docs/official/外服玩家档案0x28全量帧.md`。

---

## 一、判别轮次时间线（第十~十六轮，全部真机定案）

| 轮 | 结论 |
|---|---|
| 10 | C2S key 销案（CC3B6DFA 对 C2S 同样正确；「乱码」=信封 [1:17] 16B 签名段误读）。外服时序定案：`0x0B`+化身+首爆发进场即发 → 客户端读条 3~10s → **C2S 单字节 0x09=就绪** → 65ms 第二爆发 → 1s-tick AOI 物化簇 |
| 11 | 两阶段回放失败根因=双进场毒杀（trace: `game_visual:GetPlayer not exist`）；**wholesale 进场替换（NINEYIN_REPLAY_REPLACE_ENTRY）= 全量渲染成功**（角色/怪/任务/邮件/经脉/武学/快捷栏/内修/buff 全显） |
| 12 | 换心（SWAP_SELF）：我方 41 字段档案=只剩头；k#1978 明文模板换 id=「主玩家创建失败 1706-2309」（lua：`msg_CreateMainPlayerFailed`，`CreateSceneBodyObject` 返回无效对象；k=re-entry 缝合帧不可作首次进场模板） |
| 13 | **SELF_IDS 会话身份借用外服 c 对（0x07020fa4/0x003b3cc3）=化身全渲染+我方服务端逻辑首次整体命中 self**（出招/经脉/轻功/格挡/选靶全通）；buff 被我方 155 属性 vitals 帧清空 → 固定 buff 双件套定案并落地 |
| 14 | slim 包（1670→335 帧，纯结构帧）；物化推迟到显式 0x09；物品域挪至 0x09 后（op15 视图容器就位才发行） |
| 15 | **★对象表闸定案**：单变量递进（基线✓→我方 0x0B✗→透传✓）锁定破坏点=0x0B 内容；外服 0x0B 83B 逐字节=4 属性形；**idx1468 MaxVisuals=u16 230 缺发=对象表未开张**；`sendEntryScene` 回填后真机零 property error、怪渲染 |
| 16 | **city05 世界成型**：npc.pcapng 开场 60 帧实体 wire 模板渲染成功；**罗塞塔石碑**=外服自己发的 mode-0x46 帧×8 明文直读（19 字段定版），生成器修齐 |

## 二、0x0B 官方 4 属性形（生成器已代码化）

```
[0]=0x0B [1:5]=场景对象ID=进场流水号（样本 0x1521；恒 0x1F6A3 的 fork 形废弃）
[5:9]/[9:13]=玩家 obj/own  [13:15]=attrCount=4
idx178 Name      = 8B 遮蔽常量 1b 54 50 5b 02 58 00 00（跨包恒定）
idx1468 MaxVisuals = u16 230   ← 场景可视表容量，AddObject 注册闸
idx1447 ConfigID = 'ini\scene\school03_JunZiTang'（场景串）
idx1448 Resource = 'school03'
（无 idx99 尾——fork 形的 99 尾官方样本没有）
```

## 三、实体帧方言（0x28 家族）

- **mode 0x47（LZ）wire 原字节重放不受池状态影响**（化身透传在各 slim 环境全程成功实证）——模板库路线可行；World2 解码的明文中段会因池种子不全错位（表外怪 idx=坏字节），重放判别必须用原 wire 字节。
- **mode 0x46 官方自己在野使用**（npc.pcapng ×8）——通道无罪；可建帧 19 字段定版（罗塞塔）：
  `CollideRadius(179)=0.915 / State(183) / Type(181)=4 / Resource(1448) / RotatePara(1565)="0.0, yaw, 0.0" / ConfigID(1447) / Photo(184)="icon\funcnpc\<resource>.png" / NpcType(1566)=14 / ActionSpeed(214)=1.0 / NotPositive(1596)=1 / AIEffect(84)="nil" / Level(1472) / BossLevel(1597)=1 / IDCard(1602) / BornAction(1604)=stand / WeaponMode(1606)=hand1_1 / HPRatio(471) / ParryRatio(480) / HitHPRatio(484)=100`；变换块 scale 槽=1.25。
- 0x28 与 0x0D 同簇对象集零重叠（双通道不同对象）；外服登录固定 buff 三件套=槽1 装备整合 `buff_group_flag_101`(26439)、槽2 饱食度 `buf_baosd_02`(7735)、槽3 内功；槽 4+ 动态。

## 四、当前部署格局（start-9yin.bat env 族）

```
NINEYIN_REPLAY_FRAMES_FILE  = replay_p1_slim.bin        # 13 帧：头帧+0x0B（我方生成）+化身透传位
NINEYIN_REPLAY_PHASE2_FILE  = replay_p2_slim.bin        # 322 帧：m 族/op15/0x2A/0x10（结构帧）
NINEYIN_REPLAY_REPLACE_ENTRY= 1                          # wholesale 进场替换
NINEYIN_REPLAY_SWAP_SELF    = 1                          # 0x0B/化身换心
NINEYIN_REPLAY_SELF_IDS     = 0x07020fa4,0x003b3cc3     # 会话身份借用（撤除前置=档案我方生成）
NINEYIN_REPLAY_NATIVE_OVERLAY=1                          # 我方数据接管+物化（0x09 后）
NINEYIN_REPLAY_NPC_FILE     = replay_city05_npcs_wire60.bin  # 外服实体 wire 模板库
```

## 五、余项清单

1. 我方 catalog NPC 用修齐后生成器真机验（19 字段版首次测试）
2. 化身 posA → 存档坐标（现=外服出生点；需 c#124 级首次进场档案明文改坐标或档案生成器）
3. Level/State/CollideRadius 接 NPC 目录真值（现静态 25/stand/0.915）
4. wire 模板库 → 生成器全面替换（含 0x47 压缩器评估）
5. SELF_IDS 撤除（前置=档案 292~393 字段段我方生成，装备区逆向）
6. 背包扩展页锚点生效确认、ServerRecordAddRow add failed 单帧排查

## 六、工具资产

- `tools/c2s_key_probe.py` / `c2s_ready_probe.py`：C2S key 销案 + 0x09 就绪律
- `tools/build_replay_phases.py`：两阶段打包；slim 变体（p1_slim/p2_slim）
- `tools/c_aoi_cluster_decode.py`：c.pcapng AOI 簇 World2 解码（注意明文中段可坏）
- `tmp/op28_analysis/replay_city05_npcs_wire60.bin`：city05 实体模板（npc.pcapng 开场 60 帧）
- lua64.package 解包链：`jiuyin-unpack.exe`（PCK0 v15+Lua5.1 去 XOR）+ `unluac.jar` → `tmp/lua64_unpacked/`；`tmp/stage_main.dec.lua`（建主玩家链）
