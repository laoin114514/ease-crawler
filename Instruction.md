# 用于spider插件化爬虫项目的简易爬虫框架

## 框架结构
- `conCurrenter.go` 一个并发器, 用于控制并发, 启动多个爬虫任务
- `engine.go` 是框架核心调度器, 负责管理分组结构和已经注册的插件, 维护日志的输出和运行时的资源
- `crawler.go` 定义了爬虫插件的最小实现接口, 外部循环调度由`engine.go`负责
- `logger.go` 用于该框架的日志输出

---

## 框架使用方法

- **定义插件:** 创建实现 Crawler接口的结构体。
- **创建引擎:** 通过 easecrawler.New()初始化。
- **注册插件:** 在适当的分组下调用 Register方法。
- **启动引擎:** 调用 RunWithContext并传入一个可取消的上下文，以实现阻塞运行和优雅停止。
- **释放资源:** 程序退出前调用 engine.Close()。

### 一些方法的说明

- `conCurrenter.go`
	- `NewConCurrenter(concurrency int)` 创建一个并发器, 参数为并发数, 返回一个并发器指针
	- `NewConCurrenterWithTimeout(concurrency int, timeout time.Duration)` 创建一个带超时的并发器, 	返回一个并发器指针
	- `SetLogger(log *EaseLogger)` 设置并发器的日志记录器
	- `Run(params []T, handler func(T) error) error` 用于启动并发器, 通过主方法和内部协程工作, 返回错误

- `crawler.go`
	- `Set(key string, value interface{})` 向上下文设置一个键值对, 可以在插件运行时通过 `Get(key string)` 方法获取
	- `Get(key string)` 从上下文中读取一个键对应的值, 如果键不存在, 返回 nil 和 false
	- `GetAs(c *Context, key string)` 以泛型方式读取并做类型断言, 返回对应类型的零值和一个布尔值, 表示是否成功
	- `GetCrawlerLogger(ctx *Context) *EaseLogger` 从上下文中获取爬虫插件的日志, 如果不存在, 则使用`NewLogger`创建一个新的日志记录器

- `engine.go`
	- `New()` 创建一个新的引擎实例并初始化CrawlerGroup根分组, 返回一个引擎指针
	- `SetLoggerRootDir(dir string)` 设置引擎的日志记录器根目录, 参数为目录路径, 空字符串会回退为默认值 "logs"。
	- `GetCrawer(key string)` 获取单个插件, 参数为插件的键名, 返回已注册插件的指针
	- `GetCrawers()` 获取所有已注册的插件, 返回一个包含所有插件指针的切片
	- `DevRun(key string)` 用于开发环境, 直接运行指定的插件, 用于调试
	- `Run()` 使用 background context 启动所有注册插件。该函数会阻塞(直到所有调度 goroutine 退出，通常需配合` RunWithContext`)。
	- `RunWithContext(ctx context.Context)` 使用传入的上下文启动所有注册插件并且受到ctx的生命周期控制, 当 ctx.Done() 触发后，各插件循环会退出，RunWithContext 返回。
	- `Register(crawler Crawler, groupKey ...string)` 注册一个插件, 参数为插件实例和分组键名(可选, 默认为根分组)。
		- 关键规则：
			1. 名称不能为空
			2. 同一 groupPath/name 不能重复注册
			3. 日志路径统一为 logs/(group-path)/(crawler).log
			4. 注册失败不 panic，记录日志并返回（降级处理）
	- `Group (name string) *CrawlerGroup` 获取/创建一个分组, 参数为分组键名, 返回一个分组指针
		- 行为:
			1. 空名称直接返回当前分组
			2. 已存在子分组则复用
			3. 新建子分组并初始化对应日志目录
	- `Close()` 关闭引擎, 释放所有资源, 包括并发器和已注册的插件

- `logger.go`
	- `NewLogger(out io.Writer, prefix string, flags int)` 创建一个新的日志记录器
		- 参数:
			- out: 日志输出目标（如 os.Stdout、文件句柄等）
			- prefix: 日志全局前缀（如 [easecrawler]）
			- flags: 日志标志（如 log.LstdFlags 包含时间戳）
	- `InitLogger(out io.Writer, prefix string, flags int)` 用于全局日志实例的初始化
	- `SetPrefix(prefix string)` 设置日志记录器的前缀, 参数为新的前缀字符串
	- `GetPrefix() string` 获取当前日志记录器的前缀, 返回当前前缀字符串
	- `Printf(format string, v ...any)/Println(v ...any)/Print(v ...any)` 输出INFO级别日志, 参数format为格式化字符串, v为格式化参数
		- 框架同时提供了各个级别的日志打印方法:
			- `Errorf/Errorln/Error`
			- `Warnf/Warnln/Warn`
			- `Fatalf/Fatalln/Fatal`

### 代码示例

```go
package main

import (
	easecrawler "gitlab.unde.site/QingLuan/spider/pkg/easecrawler"
	"time"
)

// MyCrawler 示例：一个简单的采集插件
type MyCrawler struct {
	Name string
}

func (c *MyCrawler) Run(ctx *easecrawler.Context) error {
	// 1. 从上下文中获取日志器
	logger := easecrawler.GetCrawlerLogger(ctx)
	
	// 2. 执行你的采集任务
	logger.Printf("插件 [%s] 开始执行...", c.Name)
	// TODO: 你的业务逻辑，如 HTTP 请求、数据解析、存储等
	// 例如：resp, err := http.Get("https://example.com")
	
	logger.Printf("插件 [%s] 执行完成", c.Name)
	return nil // 返回 nil 表示成功，返回 error 表示失败
}

func (c *MyCrawler) Name() string {
	return c.Name
}

func (c *MyCrawler) Meta() easecrawler.Meta {
	return easecrawler.Meta{
		Interval:          10 * time.Second, // 每10秒执行一次
		StartImmediately:  true,             // 引擎启动后立即执行一次
		Logger:            nil,               // 使用框架分配的日志器
	}
}
```
