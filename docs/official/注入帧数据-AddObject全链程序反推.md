# 注入帧数据·AddObject 全链程序反推（2026-11-14 定版）

> 方法论切换：B 线注入帧此前全靠抓包样本逐字节照抄（16-prop/19 字段罗塞塔），
> 结论：**我方两代生成器编码与程序逐位一致，
> 「解析零报错但不建对象」的卡点不在 wire 层**——fxnet2 侧建对象无条件成功（场景就绪时），
> 缺口在 fxgamelogic→Lua 视觉创建链。
>
> 工具：`tools/re_inject/`（fxnet2_addobject_disasm.py 驾驶舱 / fxnet2_xref.py E8E9 交叉引用 /
> fxnet2_stringmap.py 诊断串→函数真身表 / fxnet2_rtti.py / fxgamelogic_dis.py）。
> 分析副本 `artifacts/fxgame/fxnet2_x64.dll`（=F: 外服家族客户端 bin64，MD5 一致）。

## 一、函数真身表（字符串 xref 定标，修正旧文档地址系）

旧 `fxnet2-op-dispatch-table.md` 的 RVA 与本二进制对不上（壳内布局差异）——**以诊断串
`(GameReceiver::Server*)xxx error` LEA 反查为准**重建：

| op | 函数 | 真身 RVA | 消费者槽（fxgamelogic vtable@0x12e02178） |
|---|---|---|---|
| 0x0B | ServerEntryScene | 0xd810 | +0xD8 = vf27 ENTRY_SCENE |
| 0x0C | ServerExitScene | 0xda60 | +0xE0 = vf28 EXIT_SCENE |
| **0x0D** | **ServerAddObject** | **0xdbc0** | **+0xE8 = vf29 真·OnAddObject** |
| 0x0E | ServerRemoveObject | 0xdee0 | +0xF0 = vf30（Remove） |
| 0x0F | ServerSceneProperty | 0xe1d0 | — |
| 0x10 | ServerObjectProperty | 0xe3b0 | — |
| 0x1F | ServerLocation | 0x12ce0 | +0x218 = vf67 LOCATION |
| 0x2D/2E | ServerAllProp | 0x140e0 | — |
| 0x2F/30 | ServerAddMoreObject | 0x14610 | — |
| （值对） | **RecvProperty** | **0xa2c0** | — |
| （帧分发器） | GameReceiver::OnDispatch | ~0x1bca0 | 主表 @0x1c26c（op-1 索引） |
| （解压包装） | LZ unwrap | 0x11f0 / 0x126d | 二级表 @0x1c244（op 0x27..0x30） |

**vtable 槽勘误**：GAMEMSGHANDLER-MAP.md 把 vf29 标成 OnRemoveObject 是错置——
fxnet2 ServerAddObject 调 +0xE8（slot29=0x12a348e0）、ServerRemoveObject 调 +0xF0（slot30）。
slot29 真身 = OnAddObject（本轮 disasm 定案）。

## 二、0x0D AddObject 帧——程序权威布局

```
[+0]    op 0x0D
[+1]    obj  u32     ─┐ 两段拼对象键名：sprintf("%u-%u", obj, own) → "286331153-134347291"
[+5]    own  u32     ─┘ （格式串 @0x41358 "%u-%u"，0x12e0 格式化到 0x20 缓冲）
[+9]    posA x,y,z,yaw  4×f32   （当前位）
[+0x19] 8×f32                  （posB x,y,z,yaw + scale + 第二朝向 + 2×保留零）
[+0x39] u32 标志               （官方帧恒 1）
[+0x3D] count u16              （值对数）
[+0x3F] 值对×count
硬闸：帧长 < 0x3F → "(GameReceiver::ServerAddObject)size error" 整帧丢弃
```

值对（RecvProperty 0xa2c0 循环，与 0x10/0x0F 同一方言）：

```
[idx u16][值]
  idx ≥ 0x09 表长（[this+0xB0]）→ 整帧失败 = "property error"（唯一来源）
  值宽 = 0x09 表该行 TypeCode：1=byte 2=word 3=i32 4=i64 5=f32 7=串 8=宽串 9=object
  串 = [u32 len][len 字节]（GBK，len 含尾 NUL；0x9c00 读取器）
```

## 三、建对象闸——与旧结论相反：查不到=新建

旧结论「AddObject 查对象表名查不到不建对象」**对本客户端不成立**。真实链条：

```
scene = [this+0x38]mgr → [+0x68]当前场景     （0x0B EntryScene 切换设置；null → "no scene" 拒）
gate 0x1d6e0(scene, name):
    factory = [scene+8] → vt+0x130("GameSceneObj", name)   ★类名串 @0x434C0
    = CreateOrGet（按名建/取 GameSceneObj 实体，0x100 字节，构造器 0x1dd90）
    scene+0x88 map 注册（0x91d0）
→ RecvProperty 写属性 → 变换块写 record+0x88..0xB8 → consumer vt+0xE8(name, count) 通知
```

- **GameSceneObj 构造器零属性注册**——record+0x78 schema 表（0x48B 哈希对象）初始为空；
  RecvProperty 名字 miss 时 **0x94c0 动态插入新绑定** + consumer vt+0x110(record, name)
  「新属性」通知（=vf34）。**不存在字段集白名单/静默丢弃**（视图 0x0A 族才有固定 schema）。
- 字节值有 `xor [record+0x70]` 掩码（构造器置 0，GameSceneObj 恒 0，无影响）。
- 对象键名只由 obj/own 两 u32 决定——**身份=名字**，重复 AddObject=按名合并。

## 四、0x28 包装——就是 AddObject 的压缩外衣（非「详情」通道）

分发器（~0x1bca0）：`op=frame[0]`；op∈0x27..0x30 → 二级表 @0x1c244：

```
scratch = [this+0x88]; scratch[0] = op（0x28 保留）
mode = frame[1]; hdrlen = (mode&2) ? 4 : 1
outlen = u32[frame+2+hdrlen] & mask((4-hdrlen)*8)   ; 0x46/0x47 → u32@+6 全 32 位
outlen > 0xFFFF → 丢弃
0x11f0(blob=frame+1, dst=scratch+1)：mode&1 → LZ(0x1000)；else → 0x126d 纯 memcpy
  解压体起点 = blob+1+hdrlen+4 = frame+10
decoded_len != outlen → 丢弃
→ 内层分发：主表[scratch[0]-1] @0x1c26c → op 0x28 块(0x1c0d9) → call 0xdbc0 ServerAddObject
  （rsi=scratch、rdi=outlen+1——ServerAddObject 读 scratch+1 起 = 体部）
```

**所以 0x46 明文帧体布局 = 0x0D 帧去掉 op 字节**（npc_detail_v3.go 的注释定义完全正确）：

```
[0x28][0x46][帧长-1 u32 @+2][体长 u32 @+6][体 @+10]
体 = [obj u32@0][own u32@4][posA@8:24][posB@24:40][scale@40][rot2@44][0@48,52]
     [u32 1@56][count u16@60][值对@62]
```

- `+2` 的「帧长-1」在 fxnet2 层不校验（传输层帧界用）；**`+6` 体长必须与实际体长精确相等**。
- mode 0x46：bit0=0（memcpy）bit1=1（4B 长度头）✓ 我方两生成器均用 0x46。
- **架构修正**：v3「0x28 详情帧」实质是第二个 AddObject（同对象=属性合并+重写 posA/posB
  +再通知）。外服方言 0x0D 与 0x28 物化的是两群互不重叠的对象（npc.pcapng 实证），
  并非「先物化后详情」的配对关系。我方当前对同一 NPC 0x0D(或 0x46 模板)+v3 0x28 双发
  = 无害冗余（记录合并），但 posA 以后发帧为准。

## 五、EntryScene / Location 补充

- **0x0B EntryScene**：`len ≥ 0xF`；场景名 = "%u-%u" % (u32@+5, u32@+9)；
  0x1a40 = leave 旧场景（vt+0x30）→ 建/取新场景 → [mgr+0x68]=当前场景；
  属性块 count u16@+0xD、值对@+0xF。**对象帧必须晚于 0x0B**（否则 "no scene"）。
- **0x1F Location**（0x12ce0）：消费者 +0x218=vf67；body 形 obj@1/own@5/posA@9
  （工作区 replayFramesPackAlignSelfPos 的改写位与程序读位一致）。

## 六、生成器逐位校验结论（抓到两处真数据坏，已修）

编码方言逐位一致；但程序推导不变量测试（official_npc0x28_official_test.go）
在 officialNPC0x28Frame 抓到两处真数据坏，2026-11-14 已修：

1. **Photo 串 `\f` 换页符笔误**：`"icon\funcnpc\\"` 的 `\f` 被 Go 解析成
   0x0C——客户端收到 `icon\x0Cuncnpc\…` 而非官方样本 `icon\funcnpc\…`
   （崩溃报告样本 #2 实证官方形）。同类笔误全仓扫描仅此一处真雷
   （main.go:4333 为注释假阳性；v3/旧生成器档均正确双反斜杠）。
2. **posB 全零**：变换块 posB 槽（体+24）写了 0,0,0,0——ServerAddObject 把
   posA/posB 都写进 record（+0x88/+0x98），外服静立样本 A=B；posB=0=
   客户端记录「目标位=原点」。已改 A=B（v3 生成器本来就是对形）。

| 检查项 | 程序事实 | officialNPC0x28Frame | npcDetail0x28V3Frame |
|---|---|---|---|
| 包装头 10B（0x46/len-1/outlen/体@10） | §四 | ✓ | ✓ |
| 体头 obj/own/12×f32/u32@56 | §二平移 | ✓（修后） | ✓ |
| count@60/值对@62 | ✓ | ✓ | ✓ |
| 值对 idx u16 + 表线型宽 | ✓ | ✓ | ✓ |
| 串 [u32 len 含NUL][GBK+NUL] | ✓ | ✓（修后） | ✓ |
| outlen == len(体)（解码校验） | ✓ | ✓ | ✓ |
| 帧长 ≥ 0x40 | ✓ | ✓（19 字段体 ~300B） | ✓ |

回归锚：TestOfficialNPC0x28FrameInvariants——Photo 前缀 `icon\funcnpc\`、
全串禁 0x0C、静立 posB==posA、值对区精确贴齐体长、idx 全在表内。

被撤的 `NpcType(1566)=14` 补丁——**二续反转**：npc.pcapng 原生 19 字段怪帧实证
官方本就带 NpcType（i32=14，位次=Photo 后/ActionSpeed 前，见 §七），已按原生位次
恢复进模板；fxgamelogic vf68 的 NpcType==45=静默忽略（本轮亲读 0x12a38d00 证实：
FindProp("NpcType")+GetInt==0x2D → return 0）。

## 七、原生帧验证与罗塞塔定版（二续：npc.pcapng 原生 0x46 帧程序级验证）

`tools/re_inject/extract_native_npc_pack.py` 用本文解析器逐帧验证 npc.pcapng 的
8 个原生 0x46 帧：**7 过 1 坏**（坏帧=#17027 流尾截断，outlen 失配+idx 越表——
上轮判别通道翻车的坏数据类，本次被验证器拦截）。已产回放包
`tmp/npc_native46_replay.bin`（2952B/7 帧，[u32 len][frame] 格式）。

**19 字段怪形定版字段序**（#1841/#1898/#1956 三帧同构，值全解出）：

```
179 CollideRadius f32=0.915 | 183 State str=walk | 181 Type u8=4
1448 Resource str=npc\worldnpc493 | 1565 RotatePara str="0.0,<var>,0.0"
1447 ConfigID str=monc010504 | 184 Photo str=icon\funcnpc\worldnpc493.png
1566 NpcType i32=14 | 214 ActionSpeed f32=1.0 | 1596 NotPositive i32=1
84 AIEffect str=nil | 1472 Level u8=5 | 1597 BossLevel u8=1
1602 IDCard 5×null | 1604 BornAction=stand | 1606 WeaponMode=hand1_1
471/480/484 三 Ratio=100
```

- **23 字段 Boss 形**（#3542/#10472）：多 1562 Fixed=1/1605 WeaponName=s_r_255/
  1610 FixedBeHitAct/1618 SelfForceID=999 + 尾部 631/632 BufferInfo CSV；
  **NpcType=4**（非 14）、Level=160/BossLevel=10/scale=5.0/CollideRadius=1.0。
- **40/49 字段离线玩家形**（#13176/#13271）：Type=2/LogicState=106(offact)/
  1415 OffLineState=1 + Name(wstr 遮蔽)/Face/Hair/Cloth/Weapon/Book/School/
  ActionSet 族——**化身 0x28 档案参考方言**（41 字段档案的在场面）。
- posA≠posB（walk 走动物化 A=当前/B=段末目的地）✓ v3 语义一致；官方 RotatePara
  中段=非零变值（我方模板静态零——待真机判别项）。

## 七b、NpcType 语义定版（2026-11-14 深夜类型扫描轮，23:28 会话）

真机扫描包（npc_typesweep46.bin，#1841 基底只改 Type/NpcType）：Type=4 组
NpcType∈{1,4,14,99} 四只外观行为全同；**NpcType=45 负对照缺席**——vf68 反汇编
的静默忽略闸真机验证通过（反推链端到端可信）。

程序+Lua 语义定案（share/npc_type_define.lua 110 值+使用点）：

- **NpcType 不分派视觉模型**（模型=Resource 的 ini）；它分派**交互/功能**：
  - HeadEffectType 表（head_game.lua）：**61→func_01 … 69→func_08 头顶功能图标**、
    122~138→func_10、50/51→func_11、300→func_05——可观测差异要用这些值
  - menu_functions：401=治疗英雄右键菜单、704=宠物举报
  - mouse_control：180=骑乘光标（配 HostName）、ROAD_SIGN=路牌
  - fxgamelogic vf68：**45=整对象忽略**
- **官方怪用 14=未定义值**（不在 110 值 define 表）——无分支命中=通用怪默认行为，
  这就是四只全同的原因。
- **Type(181)=视觉大类分派位（真机 Type 轴观察补记）**：Type=2 走玩家合成链
  （role_composite 读 Cloth/Hair/Face/Sex）——缺这组属性时只渲染出局部（头）；
  Type=1/3/5 出怪模型但**无动作**（动作集选择按 Type 分派，非 4 无匹配=站桩）；
  Type=4=完整怪链（ini 模型+动作）。**生产模板保持 Type=4**；人形 NPC 要动
  =Type=2+补全 40/49 字段离线玩家形（Cloth/Hair/Face/Sex/ActionSet 族，§七）。

**头标轮真机确认（达到预期）**：npc_headicon46.bin 13 帧——61~69 各挂不同
功能图标、45 位缺席。「不同功能 NPC 的头标」=纯数据配置（NpcType 选值）。

生产接线（同轮落地+数据源勘误）：officialNPC0x28Frame 增 npcType 参透传
（schema 真值>0 优先、缺省 14；scene_lifecycle 物化点按 entity.properties 查
NpcType）。**勘误：模板表自带 NpcType 列**（npcconfig/*/**.txt 三行头第三行
英文名绑定，如 worldnpc/newworldnpc.txt 沐蝶音 NpcType=533）——此前误查
share/npc/world 不存在的目录得出「无列」结论；解析器按绑定行落
Properties["NpcType"]，sceneFields 含 1566（回归锚
TestSceneFieldsContainNpcType）→ **管线全通：模板真值→OrderedProperties→
entity.properties→物化帧 idx1566→客户端 HeadEffectType 挂头标**。
npcconfig/npc_type.txt 是编辑器分类典（非必需数据源）。

**生产模板真机首测（23:28 会话，OFFICIAL=1）**：961 条 opcode=0x28 模板帧
fork id（0x11001xxx）全量下发、用户确认 NPC 渲染——id 形状问题关闭
（与反推一致：各层无 id 校验），pack B 隔离实验不再需要。

## 八、对象 id 形状证据（官方 vs 我方）

`tools/re_inject/official_npc_id_shapes.py`（npc.pcapng 84 物化帧普查）：

- **官方全部 obj=0x0e00xxxx/0x0e0xxxxx 形**；own 两族：0x0a5xxxxx（有主）/
  0x0000xxxx（自持），自持族 own 与 obj 低 16 位强相关（+0xDD 步进族）。
- motion_id=(obj>>8)&0xFFFFF 全落 917530..921498 窄带（会话流水语义）。
- **我方今晚会话**（21:08，live.log 152 条 materialize）obj=0x11000005..（fork
  序列形）——与官方完全不同形；化身渲染成功恰是「借官方 id」
  （NINEYIN_REPLAY_SELF_IDS）先例。
- 程序侧无 id 形状校验（"%u-%u" 无闸）；id 影响下游 motion/visual 查表
  （vf68 motion_id）——**两步判别第二变量**。

## 九、真机判别通道（已重装，数据经预验证）

**★ 判别结果（2026-11-14 21:38/21:40 两会话）：原生 7 帧 NPC 出现**——
服务端日志 `replayed 7 validated official NPC frame(s)`、trace.log 零 AddObject/
scene 报错、客户端可见。**渲染缺口定案=生成器内容**：相位/时序/传输层/官方 id
全部打通，0x46 明文方言端到端可用。剩余唯一未隔离变量=fork 形 id
（0x11000xxx）能否渲染——pack B（`tmp/op28_analysis/npc_forkid46_replay.bin`，
19 字段怪帧仅改写体头 obj/own 为 0x11001001..03，其余零改写）已备好隔离验证。

- `NINEYIN_REPLAY_NPC_FILE=<pack 路径>`：显式 0x09 后零改写直发 7 帧验证过的
  官方原生 0x46 帧（两步判别法第一步；上轮同位通道因半坏数据误判被撤 1a9168e，
  本轮数据经 fxnet2 解析器逐帧预验证）。
- 判别树：出现=渲染缺口定案「生成器内容差异」→ 按 §七字段序收敛模板；
  不出现=相位/id 形状 → §八（NPC id 借官方 0x0e00xxxx 形）。
- 今晚会话（21:08-21:13 fork 0x0D 大档）trace.log 全程零 AddObject/scene 报错
  ——fxnet2 对 638B fork 大档也全收（动态 schema 落错名槽），「不渲染」确在下游。

## 十、「解析零报错但不建（可见）对象」的定位收敛

fxnet2 侧：场景就绪时 AddObject **必然**建记录并通知（无白名单闸）→ 不建可见实体只能
发生在下游：

```
consumer vt+0xE8 = 0x12a348e0 OnAddObject:
    [this+0x60] 组件空 → 返回
    game_client 组件按名查 record（0x12a9f450，"%u-%u"）→ miss → 返回（静默）
    0x11d38290 SceneObjRecord::Add(name, 0)（count 被丢弃）
→ 记录行进场景对象视图 → Lua 回调（scene_obj:RegisterCallBack）
→ stage_main.show_scene_obj(ident) → scene_obj:AddObject(ident)（C++ 组件）
→ CreateSceneBodyObject（模型/动作加载，失败即不可见）
```

嫌疑排序（结合既有真机证据）：
1. **视觉表 id 形状**：vf68 `wid=record->GetID(); motion_id=(wid>>8)&0xFFFFF`——
   我方合成 NPC 的 obj u32 若非官方形（如 0x7A5C0B 族：高 20 位=视觉/动作表项），
   查表 miss → 对象在但不渲染。**下一轮真机判别首选**。
2. Resource 模型串有效性（CreateSceneBodyObject 模型加载失败）。
3. NpcType==45 误值（不可能——模板无此行/值 14）。
4. 时序：对象帧早于 0x0B（"no scene" 静默丢弃，trace.log 可证）。

## 十一、后续工具沉淀

- `tools/re_inject/fxnet2_addobject_disasm.py`：任意 RVA 反汇编驾驶舱（含函数名标注）。
- `tools/re_inject/fxnet2_stringmap.py`：Server* 函数真身表重建（诊断串 xref）。
- `tools/re_inject/fxnet2_xref.py`：全 .text E8/E9 交叉引用。
- `tools/re_inject/fxgamelogic_dis.py`：fxgamelogic dump 节映射反汇编 + vtable 槽读取。
- Lua 侧全链可读：`tmp/lua64_unpacked/lua64/`（本轮重解包 2297 文件）+
  unluac 反编译样例（scene_obj_prop/stage_main/role_composite .dec.lua 在 tmp/）。
