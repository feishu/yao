- 项目模块说明
```
mermaid
graph TB
    subgraph YaoEcosystem [Yao 整体工程生态分工]
        direction TB
        YaoRepo["yao (应用网关与编排宿主)<br/>• 负责: CLI / HTTP API 网关 / 静态资源 / 插件 / SUI / Neo AI<br/>• 目的: 统筹全局生命周期，将各组件拼装为独立可运行二进制"]
        
        GouRepo["gou (核心执行框架)<br/>• 负责: Process 调度 / Model 关系映射 / Flow / Task / V8 绑定<br/>• 目的: 业务执行引擎，提供声明式与脚本化运行时"]
        
        XunRepo["xun (数据库抽象层 DBAL)<br/>• 负责: SQL 语法编译器 / Schema 迁移 / 多数据库方言 (MySQL, PG, 达梦, Oracle)<br/>• 目的: 消除底层数据库差异，提供统一 Fluent 查询与结构构建器"]
        
        V8GoRepo["v8go (底层 CGO V8 引擎绑定)<br/>• 负责: V8 Isolate / Context / Inspector / C++ 与 Go 桥接<br/>• 目的: 提供极致的原生 JS/TS 脚本执行能力"]
        
        KunRepo["kun (基础工具基建)<br/>• 负责: MapStrAny / 日志 / 异常封装 / 类型转换<br/>• 目的: 提供全框架共享的基础数据结构与工具函数"]
    end

    YaoRepo --> GouRepo
    GouRepo --> XunRepo
    GouRepo --> V8GoRepo
    GouRepo --> KunRepo
    XunRepo --> KunRepo

```

- 核心运行机制

```
flowchart TD
    subgraph Layer1 [1. 网关与宿主编排层: yao]
        HTTP[Gin Router API 网关]
        Widgets[内置组件 Table/Form/Chart]
        SUI[SUI 模板服务端渲染]
        Neo[Neo AI 助手集成]
    end

    subgraph Layer2 [2. 核心调度与业务模型层: gou]
        ProcBus[Process 统一调度总线]
        ModelLayer[Model 动态模型定义]
        QStack[QueryStack 关系查询栈]
        V8Disp[V8 Runner Dispatcher]
    end

    subgraph Layer3 [3. 数据持久与数据库抽象层: xun]
        QB[Query Builder 链式构造器]
        Grammar[多数据库方言: MySQL / PG / 达梦 / SQLite]
        Capsule[Capsule 全局连接池管理]
    end

    subgraph Layer4 [4. 脚本执行底层绑定: v8go]
        Isolate[V8 Isolate 隔离实例]
        Context[V8 Context 执行环境]
        Bridge[CGO Go <-> JS 双向桥接]
    end

    subgraph Layer5 [5. 基建工具支撑层: kun]
        Maps[MapStrAny 点分树扁平化]
        Ex[Exception Panic/Recover 控制流]
        Log[基于 Logrus 的日志封装]
    end

    Layer1 --> Layer2
    Layer2 --> Layer3
    Layer2 --> Layer4
    Layer3 --> Layer5
    Layer4 --> Layer5

```