/**
 * Asynq 延迟任务高级 TypeScript 强类型 SDK
 *
 * 设计特性:
 * 1. 深度类型推导 (Type-Driven Architecture)
 * 2. 丰富的时间表达式 (支持 "30s", "10m", "1.5h", "2d", "1h 30m", Date 对象)
 * 3. 生产级 Asynq 配置项 (指定队列、重试策略、去重 Key、超时控制)
 * 4. 流畅构造模式 (Fluent Builder Pattern) 与中间件/拦截器支持
 */

import { Process } from "@yao/runtime";

// ============================================================================
// 1. 类型定义与时间表达式解析引擎
// ============================================================================

export type TimeUnit = 's' | 'sec' | 'm' | 'min' | 'h' | 'hr' | 'd' | 'day';

export type TimeString =
    | `${number}${TimeUnit}`
    | `${number} ${TimeUnit}`
    | `${number}${TimeUnit} ${number}${TimeUnit}`
    | `${number} ${TimeUnit} ${number} ${TimeUnit}`;

export type DurationInput = TimeString | number | Date;

export interface AsynqTaskConfig {
    /** 指定队列名称，默认为 "default" (例如 "critical" | "high" | "low") */
    queue?: string;
    /** 任务最大重试次数 (默认 3) */
    maxRetry?: number;
    /** 任务超时时间 (例如 "30s" | "5m") */
    timeout?: DurationInput;
    /** 任务去重 Key，在特定时间段内防重复投递 */
    uniqueKey?: string;
    /** 去重 Key 的生效生存周期 (例如 "1h") */
    uniqueTTL?: DurationInput;
    /** 任务完成后保留时长 (例如 "24h") */
    retention?: DurationInput;
    /** 附加扩展元数据 */
    metadata?: Record<string, any>;
}

/**
 * 高级时间解析器：解析各种时间表示形式并统一转换为秒数或 ISO 8601 时间戳
 */
export class TimeUtils {
    private static readonly UNIT_MAP: Record<string, number> = {
        s: 1, sec: 1,
        m: 60, min: 60,
        h: 3600, hr: 3600,
        d: 86400, day: 86400,
    };

    /**
     * 将输入统一转为绝对秒数 (Seconds)
     */
    static toSeconds(input: DurationInput): number {
        if (typeof input === 'number') {
            if (!Number.isFinite(input) || input < 0) {
                throw new Error(`[AsynqTask] Invalid duration: ${input}`);
            }
            return input;
        }

        if (input instanceof Date) {
            if (Number.isNaN(input.getTime())) {
                throw new Error('[AsynqTask] Invalid date');
            }
            const diff = Math.floor((input.getTime() - Date.now()) / 1000);
            return diff > 0 ? diff : 0;
        }

        if (typeof input === 'string') {
            const trimmed = input.trim();
            const matches = [...trimmed.matchAll(/(\d+(?:\.\d+)?)\s*([a-zA-Z]+)/g)];
            if (matches.length === 0 || matches.length > 2 || matches.map(match => match[0]).join(' ') !== trimmed) {
                throw new Error(`[AsynqTask] Invalid time format: "${input}". Expected format like "30s", "10m", "1.5h", "2d"`);
            }

            const seconds = matches.reduce((total, match) => {
                const value = parseFloat(match[1]);
                const unit = match[2].toLowerCase();
                const multiplier = this.UNIT_MAP[unit];
                if (!multiplier) {
                    throw new Error(`[AsynqTask] Unsupported time unit: "${unit}". Valid units: s, m, h, d`);
                }
                return total + value * multiplier;
            }, 0);
            if (!Number.isFinite(seconds)) throw new Error(`[AsynqTask] Invalid duration: "${input}"`);
            return Math.round(seconds);
        }

        throw new Error(`[AsynqTask] Unsupported duration input type: ${typeof input}`);
    }

    /** 将配置中的时间字段转换为底层 Process 使用的秒数。 */
    static normalizeConfig(config: AsynqTaskConfig): AsynqTaskConfig {
        const normalized: AsynqTaskConfig = { ...config };
        if (config.timeout !== undefined) normalized.timeout = this.toSeconds(config.timeout);
        if (config.uniqueTTL !== undefined) normalized.uniqueTTL = this.toSeconds(config.uniqueTTL);
        if (config.retention !== undefined) normalized.retention = this.toSeconds(config.retention);
        if (config.metadata !== undefined) normalized.metadata = { ...config.metadata };
        return normalized;
    }

    /** 复制配置，避免中间件修改默认配置或并发调用之间相互影响。 */
    static cloneConfig(config: AsynqTaskConfig): AsynqTaskConfig {
        return this.normalizeConfig(config);
    }

    private static assertFiniteSeconds(seconds: number): number {
        if (!Number.isFinite(seconds) || seconds < 0) {
            throw new Error(`[AsynqTask] Invalid duration: ${seconds}`);
        }
        return Math.round(seconds);
    }

    /**
     * 将输入统一转换为 ISO 8601 时间戳字符串
     */
    static toISOString(input: Date | string | number): string {
        if (input instanceof Date) {
            if (Number.isNaN(input.getTime())) throw new Error('[AsynqTask] Invalid date');
            return input.toISOString();
        }
        if (typeof input === 'number') {
            return new Date(Date.now() + this.assertFiniteSeconds(input) * 1000).toISOString();
        }
        if (typeof input === 'string') {
            // 如果已经是 ISO 字符串
            if (!isNaN(Date.parse(input))) {
                return new Date(input).toISOString();
            }
            // 尝试按相对时间解析
            const seconds = this.assertFiniteSeconds(this.toSeconds(input as TimeString));
            return new Date(Date.now() + seconds * 1000).toISOString();
        }
        throw new Error(`[AsynqTask] Cannot convert input to ISOString: ${input}`);
    }
}

// ============================================================================
// 2. 中间件与上下文
// ============================================================================

export interface AsynqTaskContext<TArgs extends any[] = any[]> {
    processName: string;
    args: TArgs;
    config: AsynqTaskConfig;
    delaySeconds?: number;
    executeAt?: string;
}

export type AsynqTaskMiddleware<TArgs extends any[] = any[]> = (
    ctx: AsynqTaskContext<TArgs>,
    next: () => Promise<string>
) => Promise<string>;

// 全局中间件列表
const globalMiddlewares: AsynqTaskMiddleware[] = [];

/**
 * 注册全局 AsynqTask 中间件 (例如注入 Audit Log、Trace ID、监控等)
 */
export function useAsynqTaskMiddleware(middleware: AsynqTaskMiddleware): void {
    globalMiddlewares.push(middleware);
}

// ============================================================================
// 3. 流畅配置 Builder (AsynqTaskOptionsBuilder)
// ============================================================================

export class AsynqTaskOptionsBuilder<TArgs extends any[], TRet = any> {
    private config: AsynqTaskConfig = {};

    constructor(private readonly parentAsynqTask: AsynqTask<TArgs, TRet>, baseConfig: AsynqTaskConfig = {}) {
        this.config = TimeUtils.cloneConfig(baseConfig);
    }

    /** 指定队列 */
    inQueue(queue: string): this {
        if (!queue.trim()) throw new Error('[AsynqTask] Queue name must not be empty.');
        this.config.queue = queue;
        return this;
    }

    /** 指定最大重试次数 */
    withRetry(maxRetry: number): this {
        if (!Number.isInteger(maxRetry) || maxRetry < 0) {
            throw new Error(`[AsynqTask] Invalid maxRetry: ${maxRetry}`);
        }
        this.config.maxRetry = maxRetry;
        return this;
    }

    /** 指定超时时间 */
    withTimeout(timeout: DurationInput): this {
        this.config.timeout = timeout;
        return this;
    }

    /** 指定去重 Key */
    withUniqueKey(key: string, ttl?: DurationInput): this {
        if (!key.trim()) throw new Error('[AsynqTask] Unique key must not be empty.');
        this.config.uniqueKey = key;
        if (ttl !== undefined) this.config.uniqueTTL = ttl;
        return this;
    }

    /** 相对延迟调度 */
    async delay(delayTime: DurationInput, ...args: TArgs): Promise<string> {
        return this.parentAsynqTask.internalDispatchDelayed(delayTime, args, this.config);
    }

    /** 绝对时刻调度 */
    async scheduleAt(executeAt: Date | string, ...args: TArgs): Promise<string> {
        return this.parentAsynqTask.internalDispatchAt(executeAt, args, this.config);
    }
}

// ============================================================================
// 4. AsynqTask 核心类 (Deepened Module Surface)
// ============================================================================

export class AsynqTask<TArgs extends any[] = any[], TRet = any> {
    private middlewares: AsynqTaskMiddleware<TArgs>[] = [];
    public readonly defaultConfig: AsynqTaskConfig;

    constructor(
        public readonly processName: string,
        defaultConfig: AsynqTaskConfig = {}
    ) {
        if (!processName || !processName.trim()) {
            throw new Error("[AsynqTask] Process name must not be empty.");
        }
        this.defaultConfig = TimeUtils.cloneConfig({ maxRetry: 3, ...defaultConfig });
    }

    /**
     * 为该任务绑定专属中间件
     */
    use(middleware: AsynqTaskMiddleware<TArgs>): this {
        this.middlewares.push(middleware);
        return this;
    }

    /**
     * 创建一个带特定配置的临时 Fluent Builder
     */
    withOptions(options: AsynqTaskConfig): AsynqTaskOptionsBuilder<TArgs, TRet> {
        const mergedConfig = { ...this.defaultConfig, ...options };
        return new AsynqTaskOptionsBuilder<TArgs, TRet>(this, mergedConfig);
    }

    /**
     * 相对延迟执行
     * @param delayTime 延迟时间 ("30s" | "10m" | "2h" | "1.5d" 或秒数)
     * @param args 传递给目标 Process 的强类型参数
     */
    async delay(delayTime: DurationInput, ...args: TArgs): Promise<string> {
        return this.dispatchDelayed(delayTime, args, this.defaultConfig);
    }

    /**
     * 在指定时间点绝对执行
     * @param executeAt Date 对象或 ISO 时间字符串
     * @param args 传递给目标 Process 的强类型参数
     */
    async scheduleAt(executeAt: Date | string, ...args: TArgs): Promise<string> {
        return this.dispatchAt(executeAt, args, this.defaultConfig);
    }

    /**
     * 立即异步投递执行 (延迟 0 秒)
     */
    async dispatch(...args: TArgs): Promise<string> {
        return this.delay(0, ...args);
    }

    /**
     * 批量并发投递多个同类延迟任务
     */
    async dispatchBatch(items: Array<{ delay: DurationInput; args: TArgs }>): Promise<string[]> {
        return Promise.all(items.map(item => this.delay(item.delay, ...item.args)));
    }

    /**
     * 撤销尚未执行的延迟 AsynqTask
     * @param taskId AsynqTask ID
     * @param queue 队列名称 (可选，默认 "default")
     */
    static async cancel(taskId: string, queue = "default"): Promise<boolean> {
        if (!taskId) return false;
        return Process("utils.asynq.Cancel", taskId, queue);
    }

    // ------------------------------------------------------------------------
    // 内部执行引擎与中间件洋葱管道
    // ------------------------------------------------------------------------

    internalDispatchDelayed(delayTime: DurationInput, args: TArgs, config: AsynqTaskConfig): Promise<string> {
        return this.dispatchDelayed(delayTime, args, config);
    }

    internalDispatchAt(executeAt: Date | string, args: TArgs, config: AsynqTaskConfig): Promise<string> {
        return this.dispatchAt(executeAt, args, config);
    }

    private async dispatchDelayed(delayTime: DurationInput, args: TArgs, config: AsynqTaskConfig): Promise<string> {
        const seconds = TimeUtils.toSeconds(delayTime);
        const ctx: AsynqTaskContext<TArgs> = {
            processName: this.processName,
            args,
            config: TimeUtils.cloneConfig(config),
            delaySeconds: seconds,
        };

        const runner = async () => {
            // 调用 Yao 底层 Process
            return Process(
                "utils.asynq.EnqueueIn",
                ctx.processName,
                ctx.args,
                ctx.delaySeconds,
                ctx.config
            );
        };

        return this.executePipeline(ctx, runner);
    }

    private async dispatchAt(executeAt: Date | string, args: TArgs, config: AsynqTaskConfig): Promise<string> {
        const isoString = TimeUtils.toISOString(executeAt);
        const ctx: AsynqTaskContext<TArgs> = {
            processName: this.processName,
            args,
            config: TimeUtils.cloneConfig(config),
            executeAt: isoString,
        };

        const runner = async () => {
            return Process(
                "utils.asynq.EnqueueAt",
                ctx.processName,
                ctx.args,
                ctx.executeAt,
                ctx.config
            );
        };

        return this.executePipeline(ctx, runner);
    }

    private async executePipeline(ctx: AsynqTaskContext<TArgs>, finalRunner: () => Promise<string>): Promise<string> {
        const allMiddlewares = [...globalMiddlewares, ...this.middlewares];
        let index = -1;

        const dispatchNext = async (i: number): Promise<string> => {
            if (i <= index) {
                throw new Error("[AsynqTask] next() called multiple times in middleware");
            }
            index = i;

            if (i === allMiddlewares.length) {
                return finalRunner();
            }

            const middleware = allMiddlewares[i];
            return middleware(ctx, () => dispatchNext(i + 1));
        };

        return dispatchNext(0);
    }
}

// ============================================================================
// 5. 声明式 AsynqTask 工厂函数
// ============================================================================

/**
 * 声明式创建强类型 AsynqTask 定义
 *
 * @param processName 调用的 Yao Process 名称 (例如 "scripts.order.Cancel")
 * @param defaultConfig 任务默认配置 (队列、重试策略等)
 */
export function createAsynqTask<TArgs extends any[] = any[], TRet = any>(
    processName: string,
    defaultConfig?: AsynqTaskConfig
): AsynqTask<TArgs, TRet> {
    return new AsynqTask<TArgs, TRet>(processName, defaultConfig);
}
