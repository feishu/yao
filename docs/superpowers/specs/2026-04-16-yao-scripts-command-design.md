# yao scripts 命令设计

**目标**

在 `yao` 根命令下新增 `scripts` 子命令，用于输出当前应用在加载完成后注册到 `github.com/yaoapp/gou/runtime/v8.Scripts` 的全部脚本名称。

**范围**

- 只新增一个只读 CLI 命令
- 只输出注册项名称，不输出脚本内容
- 输出范围包含普通 `scripts/*` 和 `services/*` 形成的 `__yao_service.*`
- 支持使用可重复的 `-m` 参数按名称过滤输出结果
- 支持使用 `-e, --error` 只输出 TS 加载错误
- `script.Load` 改为全局聚合错误而不是首错即停

**设计**

1. 命令挂载在根命令下，名称为 `scripts`
2. 执行顺序固定为 `Boot()` -> `engine.Load(config.Conf, engine.LoadOption{Action: "scripts"})`
3. 新增可重复 flag：`-m, --match`
4. 从 `v8.Scripts` 读取全部 key，排序后逐行写入标准输出
5. `-m` 匹配规则如下：
   - 不传 `-m` 时输出全部注册项
   - `-m a.b` 按完整名称精确匹配
   - `-m "*.xxx"`、`-m "xxx.*"` 按 glob 匹配
   - 多个 `-m` 采用 OR 语义，命中任一模式即输出
   - 非法模式直接返回错误
6. 新增 `-e, --error`：
   - 只输出 TS 错误，不输出正常脚本名
   - 输出格式优先为 `文件:行:列 错误信息`
   - 有错误时返回非零退出码
7. `script.Load` 改为聚合错误模式：
   - 遍历 `scripts` 和 `services` 时，单文件加载失败不会中断后续扫描
   - 所有错误扫描完成后统一返回聚合错误
   - 无错误的脚本仍然继续注册到 `v8.Scripts`
8. `v8` 层保留 esbuild 的 `File/Line/Column/Text`，不能再只返回纯文本错误

**验证**

- 单元测试验证名称收集逻辑会返回排序后的全部注册项
- 单元测试验证命令输出包含普通脚本和 `__yao_service.*`
- 单元测试验证 `-m` 的精确匹配、通配匹配、多模式 OR 和非法模式
- 单元测试验证 `script.Load` 遇到 TS 错误时会继续加载后续脚本并返回聚合错误
- 单元测试验证 `yao scripts --error` 只输出错误项
- 本地执行 `go test ./cmd`
