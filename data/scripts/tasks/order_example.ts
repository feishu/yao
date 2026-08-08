/**
 * 示例：Yao 延迟任务深度 TS SDK 生产实战范例
 */

import { createAsynqTask, useAsynqTaskMiddleware } from '../utils/asynq_task';

// ============================================================================
// 1. 声明式集中定义 Tasks (带强类型泛型推导)
// ============================================================================

/** 自动取消未支付订单 Task */
export const CancelOrderTask = createAsynqTask<[orderId: string, reason?: string]>(
    "scripts.order.Cancel",
    { queue: "default", maxRetry: 5, timeout: "5m" }
);

/** 自动过期 VIP 会员 Task */
export const ExpireVipTask = createAsynqTask<[userId: number]>(
    "scripts.user.ExpireVIP",
    { queue: "high" }
);

/** 发送催告通知 Task (支持加锁防重复投递) */
export const RemindUserTask = createAsynqTask<[userId: number, msg: string]>(
    "scripts.notify.Remind"
);

// ============================================================================
// 2. 注册全局拦截/审计中间件 (洋葱模型)
// ============================================================================

useAsynqTaskMiddleware(async (ctx, next) => {
    const startTime = Date.now();
    console.log(`[Task Dispatch Start] Process: ${ctx.processName}, Args:`, ctx.args);

    try {
        const taskId = await next();
        const duration = Date.now() - startTime;
        console.log(`[Task Dispatched Success] TaskID: ${taskId}, Cost: ${duration}ms`);
        return taskId;
    } catch (err: any) {
        console.error(`[Task Dispatch Failed] Process: ${ctx.processName}, Error:`, err);
        throw err;
    }
});

// ============================================================================
// 3. 业务函数中流畅调用范例
// ============================================================================

/**
 * 场景 A：创建订单后投递 30 分钟延迟取消任务
 */
export async function OnOrderCreated(orderId: string) {
    // 相对延迟 30 分钟 ("30m")
    const taskId = await CancelOrderTask.delay("30m", orderId, "超时未支付");
    return taskId;
}

/**
 * 场景 B：高级调用 — 带有动态去重 Key (防并发重复投递)
 */
export async function OnOrderPaymentPending(orderId: string) {
    // 使用 Fluent Builder，临时覆盖配置：在 2 小时内去重，投递到 critical 队列
    const taskId = await CancelOrderTask.withOptions({
        queue: "critical",
        uniqueKey: `order:cancel:${orderId}`,
        uniqueTTL: "2h"
    }).delay("1.5h", orderId, "支付确认超时");

    return taskId;
}

/**
 * 场景 C：绝对时间点调度 (例如 2026-08-10 到期)
 */
export async function GrantTemporaryVIP(userId: number, expireDate: Date) {
    // 指定绝对 Date
    const taskId = await ExpireVipTask.scheduleAt(expireDate, userId);
    return taskId;
}
