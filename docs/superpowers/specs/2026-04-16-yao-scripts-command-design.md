# yao scripts 命令设计

**目标**

在 `yao` 根命令下新增 `scripts` 子命令，用于输出当前应用在加载完成后注册到 `github.com/yaoapp/gou/runtime/v8.Scripts` 的全部脚本名称。

**范围**

- 只新增一个只读 CLI 命令
- 只输出注册项名称，不输出脚本内容、不做过滤、不改现有命令行为
- 输出范围包含普通 `scripts/*` 和 `services/*` 形成的 `__yao_service.*`

**设计**

1. 命令挂载在根命令下，名称为 `scripts`
2. 执行顺序固定为 `Boot()` -> `engine.Load(config.Conf, engine.LoadOption{Action: "scripts"})`
3. 从 `v8.Scripts` 读取全部 key，排序后逐行写入标准输出
4. 加载失败时打印错误并返回非零退出语义

**验证**

- 单元测试验证名称收集逻辑会返回排序后的全部注册项
- 单元测试验证命令输出包含普通脚本和 `__yao_service.*`
- 本地执行 `go test ./cmd`
