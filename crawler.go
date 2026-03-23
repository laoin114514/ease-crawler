package easecrawler

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/laoin114514/ease-crawler/logger"
)

// ContextLoggerKey 是框架约定的日志器注入键。
// 插件在 Run 中可通过 GetAs[*log.Logger](ctx, ContextLoggerKey) 获取专属 logger。
const ContextLoggerKey = "logger"
const LoggerPrefix = "[ease]"

// Crawler 是插件最小实现接口。
//
// 约束：
//  1. Name() 返回插件唯一名称（建议在同一分组内唯一）
//  2. Meta() 返回运行元信息（如执行间隔、是否启动即跑）
//  3. Run(*Context) 执行一次采集任务
//
// 说明：当前框架采用“单次执行函数 + 外部调度循环”的设计，
// 即 Run 只负责“做一次事”，循环调度由 Engine 负责。
type Crawler interface {
	Run(c *Context) error
	Name() string
	Meta() Meta
}

// Meta 描述插件的调度行为与可选能力。
type Meta struct {
	// Interval 为执行间隔。<=0 时引擎会回退为 1s。
	Interval time.Duration
	// StartImmediately 表示引擎启动后是否先立即执行一次。
	StartImmediately bool
	// Logger 允许插件自带日志器（可覆盖框架自动分配的 logger）。
	// 一般情况下不必设置，框架会按分组自动注入。
	Logger *logger.EaseLogger
}

// Context 是插件运行时上下文容器。
// 使用并发安全的 sync.Map，适合在多 goroutine 读写依赖。
//
// 典型用途：
//   - 注入 logger
//   - 注入 DB / 配置 / 客户端等共享依赖
//
// 注意：建议统一 key 常量，避免字符串拼写错误。
type Context struct {
	values sync.Map
}

// Set 向上下文写入一个键值。
func (c *Context) Set(key string, value any) {
	c.values.Store(key, value)
}

// Get 从上下文读取原始值。
func (c *Context) Get(key string) (any, bool) {
	return c.values.Load(key)
}

// GetAs 以泛型方式读取并做类型断言。
// 断言失败、key 不存在、ctx 为空时返回 (零值, false)。
func GetAs[T any](c *Context, key string) (T, bool) {
	if c == nil {
		var zero T
		return zero, false
	}
	v, ok := c.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	val, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return val, true
}

func GetCrawlerLogger(c *Context) *logger.EaseLogger {
	loggerInstance, ok := GetAs[*logger.EaseLogger](c, ContextLoggerKey)
	if !ok {
		return logger.NewLogger(os.Stdout, LoggerPrefix, log.LstdFlags, "INFO")
	}
	return loggerInstance
}
