# fxnet2.dll 客户端帧 op 分发全表（2026-11-14 静态解码定版）

来源：fxnet2.dll 磁盘静态分析（半壳 .text 原生码）+ capstone。
分发器 = `GameReceiver` 顶层收帧函数 **@RVA 0x1CF50**（ImageBase 0x10600000 无 ASLR，IDA 直接 `G 0x1061CF50`）。

## 结构

```
0x1CFFA  movzx r12d, byte [rsi]          ; op = 帧首字节
0x1D00E  lea eax,[r12-0x27]; cmp eax,9; ja 0x1D0EE   ; 0x27..0x30 先走二级表
0x1D02D  mov ecx,[r14+rax*4+0x1D598]; jmp rcx        ; 压缩族二级表(10项)
0x1D0EE  movzx eax, byte [rbp]           ; op(rbp=原帧 或 解压后缓冲)
0x1D0F2  cmp eax, 0x37; ja default
0x1D0FB  mov ecx,[r14+rax*4+0x1D5C0]; jmp rcx        ; 主表 0x38 项(RVA 相对 ImageBase)
```

每 handler 块 = `mov r8d,edi; mov rdx,rsi; mov rcx,rbx; call Server*; jmp 收尾`。

## 主表 @RVA 0x1D5C0：op → GameReceiver::Server*

| op | handler | 块 | 函数 |
|---|---|---|---|
| 0x00 | （忽略，直达收尾） | 0x1D4D3 | — |
| 0x01 | SetVerify | 0x1D108 | 0x16480 |
| 0x02 | SetEncode | 0x1D11B | 0x167E0 |
| 0x03 | Error | 0x1D12E | 0x16910 |
| 0x04 | LoginSucceed | 0x1D141 | 0x16A80 |
| 0x05 | WorldInfo | 0x1D154 | 0x175F0 |
| 0x06 | Idle | 0x1D368 | 0x19150 |
| 0x07 | Queue | 0x1D3A1 | 0x195D0 |
| 0x08 | Terminate | 0x1D167 | 0x17780 |
| 0x09 | **PropertyTable** | 0x1D1C6 | 0x178F0 |
| 0x0A | **RecordTable** | 0x1D1D9 | 0x17C60 |
| 0x0B | **EntryScene** | 0x1D17A | 0xE320 |
| 0x0C | **ExitScene** | 0x1D18D | 0xE590 |
| 0x0D | **AddObject** | 0x1D1A0 | 0xE700 |
| 0x0E | **RemoveObject** | 0x1D1B3 | 0xEA40 |
| 0x0F | **SceneProperty** | 0x1D1EC | 0xED40 |
| 0x10 | **ObjectProperty** | 0x1D1FF | 0xEF40 |
| 0x11 | RecordAddRow | 0x1D2E3 | 0x10520 |
| 0x12 | RecordDelRow | 0x1D2F6 | 0x11230 |
| 0x13 | RecordGrid | 0x1D309 | 0x122C0 |
| 0x14 | RecordClear | 0x1D31C | 0x12F10 |
| 0x15 | CreateView | 0x1D271 | 0xF450 |
| 0x16 | DeleteView | 0x1D284 | 0xF6D0 |
| 0x17 | ViewProperty | 0x1D297 | 0xF8C0 |
| 0x18 | ViewAdd | 0x1D2AA | 0xFBC0 |
| 0x19 | ViewRemove | 0x1D2BD | 0xFF10 |
| 0x1A | Speech | 0x1D212 | 0x18110 |
| 0x1B | SystemInfo | 0x1D225 | 0x18300 |
| 0x1C | Menu | 0x1D238 | 0x184A0 |
| 0x1D | ClearMenu | 0x1D24B | 0x18970 |
| 0x1E | **Custom**（custommsg 数字 id 通道） | 0x1D25E | 0x18B40 |
| 0x1F | **Location** | 0x1D32F | 0x13960 |
| 0x20 | Moving | 0x1D342 | 0x13C80 |
| 0x21 | AllDest | 0x1D355 | 0x13FA0 |
| 0x22 | Warning | 0x1D37B | 0x19280 |
| 0x23 | FromGmcc | 0x1D38E | 0x19420 |
| 0x24 | LinkTo | 0x1D3B4 | 0x14430 |
| 0x25 | Unlink | 0x1D3C7 | 0x14B90 |
| 0x26 | LinkMove | 0x1D3DA | 0x147E0 |
| 0x27 | Custom（仅作内层帧） | 0x1D3ED | 0x18B40 |
| 0x28 | AddObject（仅作内层帧） | 0x1D400 | 0xE700 |
| 0x29 | RecordAddRow（内层） | 0x1D413 | 0x10520 |
| 0x2A | ViewAdd（内层） | 0x1D426 | 0xFBC0 |
| 0x2B | **ViewChange（裸帧）** | 0x1D2D0 | 0x101F0 |
| 0x2C | AllDest（内层） | 0x1D439 | 0x13FA0 |
| 0x2D | **AllProp（裸帧）** | 0x1D44C | 0x14E20 |
| 0x2E | AllProp（内层） | 0x1D45C | 0x14E20 |
| 0x2F | **AddMoreObject（裸帧）** | 0x1D46C | 0x15390 |
| 0x30 | AddMoreObject（内层） | 0x1D47C | 0x15390 |
| 0x31 | RemoveMoreObject | 0x1D48C | 0x15930 |
| 0x32 | ChargeValidstring | 0x1D49C | 0x15E60 |
| 0x33-0x36 | （default 0x1D4BC，无 handler=忽略） | — | — |
| 0x37 | SetQR | 0x1D4AC | 0x16230 |

## 压缩族二级表 @RVA 0x1D598（先于主表判定）

- **0x27/0x28/0x29/0x2A/0x2C/0x2E/0x30（7 个）→ 0x1D03A 解压块**：读帧[1] bit1 定 1/4 字节长度、
  读长度 → 解压到 [this+0x88] 缓冲（调 0x10601730）→ 回 0x1D0EE 用**解压后内层帧**递归分发。
  外服 0x28 LZ 压缩对象帧（0x46/0x47 模板等）即此通道；主表里同名条目是内层帧复用。
- **0x2B/0x2D/0x2F（3 个）→ 直跳 0x1D0EE**：不压缩，ViewChange/AllProp/AddMoreObject 裸帧直发。

## 与既有事实互证

- 0x0D=AddObject：NPC 物化主通道（外服模式 NPC 渲染收束 §"0x0D 头行通道"）；0x0F=SceneProperty、0x10=ObjectProperty 与服务端属性帧方言吻合；
- 0x1E=Custom：m 数字 id 全在此外层内；0x1F=Location（msg337 朝向等坐标族）；
- 0x28=AddObject 压缩包装（外服 LZ 字典/0x46 明文模板路线，op28-fxnet2-lz-decoder.md 的 ServerAddObject 结论精确成立）；
- 0x2D/0x2E=AllProp 双通道（玩家档案 0x28 帧族走 AddObject/AllProp 复合，见 player-profile-0x28-frame）。

## 工具

tools/fxnet2_serverxrefs.py（诊断串 LEA 反查定位 49 个 Server* 函数 + E8 公共调用者定位分发器）、
capstone 解表脚本（本文来源）。

## 附：0x09 属性表与 official_property_table_2433.txt 的对照（2026-11-14 反解）

客户端解析 = `ServerPropertyTable` @0x178F0（fxnet2，disasm 定案）：

```
帧格式: [op 0x09][count u16 @+1][行数据 @+3]
每行:   [Name NUL 串][TypeCode u8]
存储:   顺序压入 GameReceiver+0xA0 的 vector —— 行序=vector 下标=表 Index（位置强绑定）
entry:  0x38 字节 { string Name@+0x18, u8 TypeCode@+0x30, flags@+0x34=0 }
校验:   消费字节数 != 帧长 → 仅 trace（ConsolePrintf）不拒帧
通知:   完成后调 this+0x40 消费者 vtable+0x78 = fxgamelogic vf15(PROPERTY_TABLE, count)
```

### 逐列对照

| 文件列 | 0x09 帧行 | 客户端内存 | 下游消费 |
|---|---|---|---|
| `Index` | 行在帧中的位置（不显式上线） | vector 下标 | 0x10/0x0F/0x0D 值对里的 index 直接查它；**错位=静默错列** |
| `Name` | NUL 结尾串 | entry+0x18 | fxgamelogic by-name 读取（`record->vt("Type")`/`"NpcType"`/`"LogicState"`…）→ hybrid2433 头块 0..173 的存在意义 |
| `TypeCode` | 行尾 u8 | entry+0x30 | 决定值对解码：1=byte 2=word 3=int32 4=int64 5=float32 7=string 8=宽串 9=object；**0/6=不可编码**（占位、计位置，值帧不得携带） |

### 值帧方言（同轮反解）

- **0x10 ObjectProperty** @0xEF40：`[op][isView u8][view/对象名 int32 @+2][行名 int32 @+6][count u16 @+0xA][值对 blob @+0xC]`
  - isView=0：对象按名字串哈希（BKDR ×0x83）在场景对象表里查
  - isView=1：view+row 双名查记录（Go 侧「视图行更新必须 0x10 isView=1」的客户端实锚）
  - 值 blob 由 0x1060BE60 按 0x09 表逐对解（index u16 + 表类型定长的值）
- **0x0F SceneProperty**：同一表同一值对方言，目标=场景域记录
- **0x0A RecordTable**：视图 schema 通道（列定义走同一 0x09 表类型码）

### 工程意义

- `NINEYIN_PROPERTY_TABLE` 三档（fork/official2433/hybrid2433）本质=选哪份文件行集按上表编码成 0x09 帧；
  official2433 头块换血（家园/帮会/VIP 域 0..~173）后 NPC 头行名无行可落 → OrderedProperties 静默丢弃 → NPC 不渲染
  （hybrid2433 = fork 头块 + official 共享域，两全）。
- 表追加只许末尾：中间插入=后续所有行 index 漂移（+2/+7 插入漂移即此类），客户端不校验名字只认位置。

## 附：IDA 快捷导航表（fxnet2.dll，ImageBase 0x10600000 无 ASLR，直接 G 跳转）

### 核心入口

| G 地址 | 位置 | 内容 |
|---|---|---|
| `0x1061CF50` | .text | 顶层分发器：op=帧首字节；0x27-0x30 先走二级表 |
| `0x1061D5C0` | .text 数据 | 主跳转表（0x38 项 RVA4，相对 ImageBase） |
| `0x1061D598` | .text 数据 | 压缩族二级表（10 项：7 解压/3 直发） |
| `0x1061D03A` | .text | LZ 解压块：帧[1] bit1 定 1/4B 长度 → 解压到 this+0x88 → 内层帧递归 |
| `0x1061D0EE` | .text | 二次分发点：`movzx eax,byte[rbp]`（原帧或解压缓冲）→ 主表 |

### 表↔帧对照核心

| G 地址 | op | 函数 | 看点 |
|---|---|---|---|
| **`0x106178F0`** | **0x09** | ServerPropertyTable | `[op][count u16@+1][行@+3: Name\0 + TypeCode u8]`；行序=vector 下标=表 Index；entry 0x38B{Name@+0x18,Type@+0x30}；尾部 `cmp rbp,r15` 长度校验；回调 vf15(PROPERTY_TABLE) |
| **`0x1060EF40`** | **0x10** | ServerObjectProperty | `[op][isView u8][对象名 int32@+2][行名 int32@+6][count u16@+0xA][值对@+0xC]`；isView=0 对象名 BKDR×0x83 哈希查、=1 走 0x1060ADA0 双名查；值对解码=**`0x1060BE60`**（按 0x09 表 index→Type 定长读） |
| `0x10617C60` | 0x0A | ServerRecordTable | 视图 schema（列定义同一类型码族） |
| `0x1060ED40` | 0x0F | ServerSceneProperty | 场景域值对，同一表方言 |
| `0x1060E700` | 0x0D | ServerAddObject | 对象物化（属性块走同一 0x09 表） |

### 其余 handler（op → G 地址）

```
0x01 0x10616480 SetVerify         0x02 0x106167E0 SetEncode       0x03 0x10616910 Error
0x04 0x10616A80 LoginSucceed      0x05 0x106175F0 WorldInfo       0x06 0x10619150 Idle
0x07 0x106195D0 Queue             0x08 0x10617780 Terminate       0x0B 0x1060E320 EntryScene
0x0C 0x1060E590 ExitScene         0x0E 0x1060EA40 RemoveObject    0x11 0x10610520 RecordAddRow
0x12 0x10611230 RecordDelRow       0x13 0x106122C0 RecordGrid      0x14 0x10612F10 RecordClear
0x15 0x1060F450 CreateView         0x16 0x1060F6D0 DeleteView      0x17 0x1060F8C0 ViewProperty
0x18 0x1060FBC0 ViewAdd            0x19 0x1060FF10 ViewRemove      0x1A 0x10618110 Speech
0x1B 0x10618300 SystemInfo         0x1C 0x106184A0 Menu            0x1D 0x10618970 ClearMenu
0x1E 0x10618B40 Custom             0x1F 0x10613960 Location        0x20 0x10613C80 Moving
0x21 0x10613FA0 AllDest            0x22 0x10619280 Warning         0x23 0x10619420 FromGmcc
0x24 0x10614430 LinkTo             0x25 0x10614B90 Unlink          0x26 0x106147E0 LinkMove
0x2B 0x106101F0 ViewChange(裸)     0x2D 0x10614E20 AllProp(裸)     0x2E 0x10614E20 AllProp(内层)
0x2F 0x10615390 AddMoreObject(裸)  0x31 0x10615930 RemoveMoreObject
0x32 0x10615E60 ChargeValidstring  0x37 0x10616230 SetQR
```

### 辅助函数

| G 地址 | 作用 |
|---|---|
| `0x1060BE60` | 0x10/0x0F 值对解码循环（查 0x09 表 entry.Type 定长读值） |
| `0x1060ADA0` | isView=1 路径 view+row 双名查找 |
| `0x10601820` | 名字格式化（int32 → 串，配 ×0x83 BKDR 哈希 @0x1060F020） |
| `0x10601730` | LZ 解压调用（0x27-0x30 压缩族） |
| `0x10653DA8` | RTTI `.?AVGameReceiver@@`（xref → vtable → 类布局） |
