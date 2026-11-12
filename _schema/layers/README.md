# 多层 schema 快照

`l228.tl` … `l233.tl` 是 telegramdesktop 官方 `api.tl` 的**原样**快照，用于生成
`layerx` 包（多层投射表）。canonical `tg` 仍然只从 `_schema/telegram.tl` 生成，
两条输入互不影响。

## 文件

| 文件 | 用途 |
|---|---|
| `meta.json` | 层清单：层号、文件名、来源仓库、提交、日期、sha256、字节数 |
| `lNNN.tl` | 该层最后一份 `api.tl`，未经任何改写 |
| `decide.json` | 差异无法自动推断时的人工决策 |
| `audit.json` | 生成器输出的审计报告（生成物，提交入库） |

## 采集

1. 在 tdesktop 仓库按路径查提交历史：
   `https://api.github.com/repos/telegramdesktop/tdesktop/commits?path=Telegram/SourceFiles/mtproto/scheme/api.tl`
   提交标题里的 “to layer N” 是升层，“on layer N” 是该层内的修订。
2. 每层取**升到下一层之前的最后一个提交**，即该层最终状态。
3. 下载：
   `https://raw.githubusercontent.com/telegramdesktop/tdesktop/<ref>/Telegram/SourceFiles/mtproto/scheme/api.tl`
4. 记录 `sha256` 与 `bytes` 到 `meta.json`。
5. 交叉校验：`l233` 的 sha 必须等于 `_schema/telegram.tl` 头部声明的 `SHA256`。

## 校验

```powershell
go run ./cmd/gotdgen -schema _schema/telegram.tl `
  -layers-meta _schema/layers/meta.json `
  -layers-decide _schema/layers/decide.json `
  -layers-target layerx -layers-package layerx `
  -layers-audit _schema/layers/audit.json -layers-only
```

校验内容：文件哈希、层号单调、`l233` 与 canonical 源一致、所有引用可解析、
ctor id 无重复、`Sym` 无碰撞；生成结果写入 `audit.json`。

## 决策

`decide.json` 每条以 `l<层>:<TL 名>` 为 key：

| 动作 | 含义 |
|---|---|
| `skip` | 该层不可用，编码/解码都拒绝 |
| `hook` | 字段类型变更，按 `conv` 调手写转换 |
| `mask` | 编码到旧层时静默丢弃该字段 |
| `keep` | 保留自动判断 |

`conv` 目前支持 `TextStr`（`TextWithEntities` ↔ `string`，见 `layerx/hooks.go`）。

未被决策覆盖的 `Hook` 会让生成直接失败，不会静默降级。

## 生成物

`layerx/` 下 `lx_*_gen.go` 由生成器覆盖，其余文件手写：

| 文件 | 用途 |
|---|---|
| `lx_sym_gen.go` | `Sym` 常量与名字表 |
| `lx_ctor_gen.go` | `(层, wire id)` 双向表 |
| `lx_plan_gen.go` | 编解码分派 |
| `lx_put_gen.go` / `lx_get_gen.go` | 差异类型的字段级编解码 |
| `lx_scan_gen.go` | 预检字段表 |
| `lx_skip_gen.go` | 层独有类型跳过 |
| `lx_wrap_gen.go` | 信封解析 |
| `lx_res_gen.go` | 方法结果类校验 |

## 私有 RPC

`priv.json` 登记 DrKLO 客户端 `not_linked_classes.txt` 里不在官方 schema 的 ctor：

| 字段 | 含义 |
|---|---|
| `id` | 客户端 ctor id |
| `name` | TL 名 |
| `go` | 可解码的请求类型（`layerx/priv.go` 手写） |
| `skip` | 只登记不解析的原因 |

有 `go` 的条目会生成 `SymPriv*`，可正常 `Add` handler 并用 `Mark.Obj()`
取到强类型请求；其余条目在 `OddView.Priv/Name` 里可见。
安卓 12.9.1（228）与官方 228 的 schema 逐条相同，差异只在私有类。

## 私有 TL

`priv.tl` 由 `cmd/layandroid` 从安卓源码抽取（Java → TL）：

```powershell
go run ./cmd/layandroid `
  -java D:\Telegram-12.9.1-layer228\TMessagesProj\src\main\java\org\telegram\tgnet `
  -src  D:\Telegram-12.9.1-layer228\TMessagesProj_AppTests\tlscheme `
  -tl _schema/layers/priv.tl -go layerx/priv_gen.go
```

内容：32 个定义。

- 11 个客户端独有（`layerx/priv.go` 手写）
- 21 个旧 id / 私有 RPC（客户端保留的旧构造器，形状与官方不同，
  由 `-go` 生成 `layerx/priv_gen.go`）

`TestPrivTL` 校验 id 集合与形状不漂移；`TestPrivGenerated` 跑往返。
私有 id 是客户端常量，不随层变化，所以 228–233 全部可用。

## 运行时不变量

- 未知层拒绝，不 clamp、不回退 canonical 字节。
- `TakeOpt` 先预检再解码，限制字节、深度、向量、聚合元素。
- 结果只能走 `Mark.PutRes`；对象只能走 `Put`；目标层无法表示即失败。
- `Freeze` 一次编码，多层 `Write` 复用。
- `gzip_packed` 必须显式提供 `Opts.Unzip`。


## 扩层

1. 追加一条 `meta.json` 记录并放入对应 `.tl`；
2. 运行校验，确认新层与上一层差异符合预期；
3. 差异无法自动分类时，在 `decide.json` 写入决策；
4. 重新生成 `layerx`，跑测试。

## 约束

- 不修改快照内容，任何补丁都写进 `decide.json` 或 `layerx/hooks.go`。
- `legacy.tl` 的合并规则与 `cmd/dltl` 一致：按 ctor id 跳过已存在的定义。
