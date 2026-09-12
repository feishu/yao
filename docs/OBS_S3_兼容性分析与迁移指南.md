# 华为云 OBS 与 Amazon S3 兼容性分析及迁移指南

> 文档版本：v1.0
> 适用场景：当前使用 rustfs（S3 兼容自托管对象存储），评估/迁移至华为云 OBS
> 客户端栈：aws-sdk-go-v2（`github.com/aws/aws-sdk-go-v2/service/s3`）

---

## 目录

1. [背景与目标](#1-背景与目标)
2. [结论速览](#2-结论速览)
3. [OBS 的两套 API 面](#3-obs-的两套-api-面)
4. [分面兼容性分析](#4-分面兼容性分析)
   - 4.1 整体兼容度
   - 4.2 逐项兼容性矩阵
   - 4.3 访问风格（Path Style vs Virtual Hosted Style）
   - 4.4 认证与签名
   - 4.5 头域与存储类型语义差异
   - 4.6 OBS 独有扩展（S3 无对应）
5. [aws-sdk-go-v2 集成指南](#5-aws-sdk-go-v2-集成指南)
   - 5.1 基础配置
   - 5.2 支持的方法清单
   - 5.3 不支持 / 会失败的方法
   - 5.4 Go 侧特有陷阱
6. [rustfs → OBS 迁移清单](#6-rustfs--obs-迁移清单)
7. [附录：术语与参考](#7-附录术语与参考)

---

## 1. 背景与目标

团队当前使用 **rustfs** 作为对象存储底座（Rust 实现、Apache 2.0、宣称 100% S3 兼容、MinIO 平替）。
现评估将存储后端迁移 / 双写到 **华为云 OBS（对象存储服务）**，核心诉求是：

- 确认 OBS 是否"全面支持所有 S3 API"；
- 确认 OBS 与标准 S3 行为是否"完全相同"；
- 在 **aws-sdk-go-v2** 下，现有调用能否平滑迁移。

---

## 2. 结论速览

| 问题 | 答案 |
|---|---|
| OBS 是否全面支持所有 S3 API？ | **否。** 覆盖约 95% 常用 S3 REST 操作，存在少数不支持项与多项语义差异。 |
| OBS 与 S3 是否完全相同？ | **不是完全相同。** 协议层可平滑迁移，但访问风格、加密模型、存储类型枚举、扩展头域存在差异。 |
| aws-sdk-go-v2 方法是否都支持？ | **绝大多数支持。** SDK 方法 1:1 映射到 S3 REST 操作，OBS 支持了对应操作的方法即可用；不支持的 REST 操作在 SDK 层同样会失败。 |
| 应用代码改动量？ | 业务 CRUD / 分片上传 / 生命周期 / 预签名基本零改动，仅换 endpoint + 凭证；但需排查 Path Style、SSE-S3、requestPayment、torrent、MfaDelete、存储类型枚举、邮箱 ACL 等。 |

---

## 3. OBS 的两套 API 面

OBS 并不是"把 S3 原样实现了一遍"，而是提供了**两套并存且语义不同**的接口：

| API 面 | 头域前缀 | 签名方式 | 说明 |
|---|---|---|---|
| **原生 OBS API** | `x-obs-` | OBS 自有签名（OBS HMAC） | 华为私有扩展（追加写、WORM、修改写、重命名、截断等）都在这一面。 |
| **S3 兼容面** | `x-amz-` | AWS Signature V4 | 面向 S3 生态（boto3 / aws-cli / aws-sdk-go-v2 / MinIO / rustfs 系工具）。 |

> ⚠️ 你最初给的链接 `obs_04_0005.html` 属于 **原生 OBS API 概览页**，不是 S3 对比矩阵。逐项兼容性要看 OBS 的 *"Compatibility with Amazon S3 APIs"* 文档。

---

## 4. 分面兼容性分析

### 4.1 整体兼容度

- OBS S3 兼容面支持 **约 95% 的 S3 核心 REST API**，覆盖对象生命周期管理的基本盘。
- 剩余的差异分为三类：**(a) 明确不支持的操作**、**(b) 支持但语义不同**、**(c) OBS 独有扩展**。

### 4.2 逐项兼容性矩阵

#### ❌ 明确不支持的 S3 操作

| S3 操作 | OBS 状态 | 说明 |
|---|---|---|
| `GET/PUT Bucket requestPayment` | 不支持 | OBS 无"请求方付费"模型 |
| `GET Object torrent` | 不支持 | `.torrent` 下载 |
| 服务端加密 `SSE-S3`（S3 托管密钥） | 不支持 | OBS 仅支持 `SSE-KMS` / `SSE-C` / OBS 自有加密 |
| `MfaDelete` | 名义接受但**无效** | 返回 200，但设置不生效 |
| 按邮箱授权 ACL（`x-amz-grant-*`） | 不支持 | 只能用 OBS canonical user ID 授权 |

#### ⚠️ 支持但存在语义差异

| 操作 | 差异点 |
|---|---|
| `GET Bucket Location` | 响应体 XML 节点结构不同（OBS 用 `Location`，S3 容器结构不同） |
| `HEAD Object` | 自定义元数据编码不同（OBS 走 Base64，S3 走 ASCII/UTF） |
| `PUT Bucket` | 桶命名规则与区域相关规则与 S3 不一致；OBS 上限 100 个桶 |
| `PUT Bucket policy` | OBS 只支持**部分** condition 条件 |
| `PUT Bucket notification` | 仅当配置项 ≤100 且配置文本 ≤100KB 时支持到 100 项 |
| 错误码 | 大部分兼容 AWS 错误码格式，但部分自定义错误含华为云特有的 Code / Message |

### 4.3 访问风格（Path Style vs Virtual Hosted Style）

**这是从 rustfs 迁移到 OBS 最关键、最容易踩的坑。**

- OBS **默认不支持 Path Style 请求**，只支持 **Virtual Hosted Style**。
  - Virtual Hosted：`https://<bucket>.obs.<region>.myhuaweicloud.com/<key>`
  - Path Style：`https://obs.<region>.myhuaweicloud.com/<bucket>/<key>` ❌
- 老版本 S3 客户端（如 Elasticsearch `repository-s3` < 7.4 仅支持 Path Style）对接 OBS 会直接失败。
- 华为官方文档明确写有"禁止使用 path 请求方式访问 OBS 桶"的通知。
- **rustfs / MinIO 自托管常默认 Path Style**，迁移时必须强制切换为 Virtual Hosted。

### 4.4 认证与签名

- S3 兼容面使用 **AWS Signature V4**，预签名 URL 同样兼容 SigV4。
- 因此 **boto3 / aws-cli / aws-sdk-go-v2 等只需换 `endpoint` + AK/SK 即可对接**，业务代码几乎不用改。
- 注意：OBS 的 AK/SK 来自华为 IAM 体系，需重新签发，不能复用 AWS / rustfs 的凭证。

### 4.5 头域与存储类型语义差异

- **头域前缀**：`x-obs-`（原生面）与 `x-amz-`（S3 面）跨面不互通。
- **存储类型枚举不同**：
  - OBS 原生：`STANDARD / WARM / COLD / DEEP_ARCHIVE`
  - S3：`STANDARD / STANDARD_IA / GLACIER / DEEP_ARCHIVE` 等
  - 在 OBS 的 **S3 兼容面**一般接受 S3 形态的值（`STANDARD_IA`）并内部映射到 `WARM`，但冷归档（`DEEP_ARCHIVE`）与具体映射**建议实测确认**。稳妥做法：桶级设默认存储类，或用生命周期规则做层级转换。
- **OBS 扩展能力 S3 没有**：追加写（Append Object）、WORM、修改写对象、重命名对象、截断对象——仅在 OBS 并行文件系统（PFS）面存在，标准 S3 语义和 rustfs 均不提供。
- **一致性**：OBS 的 List / 追加写最终一致性已压到亚秒级，S3 客户端可近似无缝替代。

### 4.6 OBS 独有扩展（S3 无对应）

| 特性 | 说明 | S3 / rustfs 是否有 |
|---|---|---|
| Append Object（追加写） | 在对象尾部追加数据 | 无 |
| WORM（一次写多次读） | 写后不可改删 | 无（rustfs 有 WORM 合规特性，但语义不同） |
| 修改写 / 截断 / 重命名对象 | PFS 面文件语义操作 | 无 |
| 并行文件系统（PFS） | POSIX 语义桶，头域 `x-obs-fs-file-interface:Enabled` | 无 |

> 反向提醒：若现有系统依赖 OBS 上述独有特性，迁回 rustfs 会丢能力。

---

## 5. aws-sdk-go-v2 集成指南

### 5.1 基础配置

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func newOBSClient(ak, sk, region string) *s3.Client {
    cfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion(region), // 必须填 OBS 桶所在区域，如 cn-north-4
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(ak, sk, ""),
        ),
    )
    if err != nil {
        panic(err)
    }

    return s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String("https://obs." + region + ".myhuaweicloud.com")
        // 关键：OBS 只支持 virtual-hosted style，切勿设 UsePathStyle: true
        o.UsePathStyle = false
    })
}
```

> 若代码中原本为 rustfs/MinIO 设了 `UsePathStyle: true`，**必须改为 `false` 或删掉该行**，否则所有请求失败。

### 5.2 支持的方法清单（直接可用）

| 能力 | aws-sdk-go-v2 方法（节选） |
|---|---|
| 桶 CRUD | `CreateBucket` / `DeleteBucket` / `ListBuckets` / `HeadBucket` / `GetBucketLocation` |
| 对象 CRUD | `PutObject` / `GetObject` / `HeadObject` / `DeleteObject` / `DeleteObjects` / `CopyObject` |
| 列表 / 版本 | `ListObjectsV2` / `ListObjectVersions` |
| ACL | `PutBucketAcl` / `GetBucketAcl` / `PutObjectAcl` / `GetObjectAcl` |
| 分片上传 | `CreateMultipartUpload` / `UploadPart` / `CompleteMultipartUpload` / `AbortMultipartUpload` / `ListParts` / `ListMultipartUploads` |
| 生命周期 / 策略 | `PutBucketLifecycleConfiguration` / `GetBucketLifecycleConfiguration` / `PutBucketPolicy` / `GetBucketPolicy` / `DeleteBucketPolicy` |
| CORS / 版本 / 通知 | `PutBucketCors` / `GetBucketCors` / `DeleteBucketCors` / `PutBucketVersioning` / `GetBucketVersioning` / `PutBucketNotificationConfiguration` / `GetBucketNotificationConfiguration` |
| 标签 / 日志 / 网站 | `PutObjectTagging` / `GetObjectTagging` / `DeleteObjectTagging` / `PutBucketLogging` / `GetBucketLogging` / `PutBucketWebsite` / `GetBucketWebsite` / `DeleteBucketWebsite` |
| 恢复归档 | `RestoreObject` |

### 5.3 不支持 / 会失败的方法

| SDK 方法 | 结果 | 原因 |
|---|---|---|
| `GetBucketRequestPayment` / `PutBucketRequestPayment` | 报错 | OBS 无 requestPayment 模型 |
| `GetObjectTorrent` | 报错 | OBS 不支持 .torrent |
| `PutObject` 带 `ServerSideEncryption: types.ServerSideEncryptionAes256`（SSE-S3） | 报错 | OBS 不支持 SSE-S3，改用 `aws:kms` 或 SSE-C（`SSECustomerAlgorithm` / `SSECustomerKey`） |
| `PutBucketVersioning` 带 `MfaDelete: types.MfaDeleteEnabled` | 静默无效 | 返回 200 但不生效 |
| ACL 用邮箱授权（`GrantFullControl` 传 email） | 报错 | 必须用 OBS canonical user ID，不支持 `x-amz-grant-*` |

### 5.4 Go 侧特有陷阱

1. **Path Style 必须关闭**：见 4.3 / 5.1。从 rustfs/MinIO 迁过来代码里大概率有 `UsePathStyle: true`。
2. **存储类型枚举对不齐**：`types.StorageClass` 只有 S3 取值，OBS S3 面一般接受 S3 形态值并内部映射，但冷归档与具体映射**建议实测**；稳妥做法用桶默认存储类或生命周期规则。
3. **Region 与 endpoint 必须匹配**：OBS endpoint 区域化，`WithRegion` 与 `BaseEndpoint` 要一致，否则签名/路由错误。
4. **同时对接 rustfs 与 OBS**：建两个 client 实例——rustfs client `UsePathStyle: true`，OBS client `UsePathStyle: false`。该开关是全局的，不能共用。

---

## 6. rustfs → OBS 迁移清单

- [ ] **切换访问风格**：移除 / 置否 `UsePathStyle`，改用 Virtual Hosted Style。
- [ ] **更换凭证**：在华为 IAM 签发 AK/SK，替换 rustfs 的凭证。
- [ ] **更换 endpoint**：指向 `https://obs.<region>.myhuaweicloud.com`。
- [ ] **排查 SSE-S3 依赖**：将 `ServerSideEncryptionAes256` 改为 `aws:kms` 或 SSE-C，否则上传报错。
- [ ] **排查 requestPayment / torrent / MfaDelete**：确认业务未使用，否则需改方案。
- [ ] **排查邮箱 ACL 授权**：改为 canonical user ID 授权。
- [ ] **校验存储类型映射**：`STANDARD_IA` ↔ `WARM`、`GLACIER` ↔ `COLD` 等映射实测；归档用生命周期。
- [ ] **校验自定义元数据 / Location 响应格式**：依赖 `HEAD Object` 元数据或 `GET Bucket Location` 结构的代码需适配。
- [ ] **列举 / 一致性回归测试**：OBS List 最终一致性亚秒级，多写后立即列举的场景加少量重试。
- [ ] **双写或灰度验证**：建议先双写一个小桶跑通全链路，再切流量。

---

## 7. 附录：术语与参考

- **S3 兼容面 / 原生 OBS 面**：见第 3 节。
- **Virtual Hosted Style**：`https://<bucket>.endpoint/<key>`，OBS 唯一支持的寻址方式。
- **Path Style**：`https://endpoint/<bucket>/<key>`，OBS 不支持。
- **SSE-S3 / SSE-KMS / SSE-C**：服务端加密三种模型，OBS 不支持 SSE-S3。
- **PFS（并行文件系统）**：OBS 的 POSIX 语义桶，提供追加写、重命名等扩展。

### 参考来源
- 华为云 OBS API 参考（原生 API 概览 `obs_04_0005.html`）
- 华为云 OBS 与 Amazon S3 API 兼容性矩阵（官方 *Compatibility with Amazon S3 APIs* 文档及运营商镜像）
- 华为云最佳实践：通过 S3 插件迁移 Elasticsearch 至 OBS（明确 Path Style 限制）
- RustFS 官方站点（定位与兼容性声明）
