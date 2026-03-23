package easecrawler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Engine 是框架核心调度器。
//
// 主要职责：
//  1. 管理分组结构（通过内嵌 CrawlerGroup 作为根组）
//  2. 管理已注册插件（crawlers）
//  3. 按 Meta 配置调度插件执行
//  4. 维护日志输出（全局日志 + 分组目录下插件日志）
//  5. 维护运行时资源（文件句柄等）
type Engine struct {
	// 根分组，路径为 "/"。
	CrawlerGroup
	// 框架级上下文容器（当前保留，后续可注入全局依赖）。
	ctx *Context
	// 全局读写锁：保护分组树、注册表、日志文件映射等共享状态。
	mu sync.RWMutex
	// 全局兜底日志器。
	engineLogger *EaseLogger
	// 日志根目录，默认 "logs"。
	logRootDir string
	// 全局插件注册表：key = groupPath/name。
	crawlers map[string]*registeredCrawler
	// 打开的日志文件句柄，便于 Close() 时统一释放。
	logFiles     map[string]*os.File
	loggerPrefix string
}

// registeredCrawler 是注册后插件的内部表示。
// 保存运行所需元数据，避免每次执行重复计算。
type registeredCrawler struct {
	path    string
	name    string
	crawler Crawler
	log     *EaseLogger
	ctx     *context.Context
	cancel  context.CancelFunc
}

// CrawlerGroup 表示分组节点。
//
// 设计说明：
//   - path: 当前分组路径
//   - children: 子分组索引（用于链式 Group）
//   - parent: 父分组引用（当前主要用于结构表达）
//   - engine: 回指引擎，便于访问共享注册表和锁
type CrawlerGroup struct {
	path     string
	engine   *Engine
	children map[string]*CrawlerGroup
	parent   *CrawlerGroup
}

// New 创建引擎实例，并初始化根分组。
func New() *Engine {
	e := &Engine{
		CrawlerGroup: CrawlerGroup{
			path:     "/",
			children: make(map[string]*CrawlerGroup),
			parent:   nil,
		},
		mu:           sync.RWMutex{},
		ctx:          new(Context),
		engineLogger: NewLogger(os.Stdout, "", log.LstdFlags),
		logRootDir:   "logs",
		crawlers:     make(map[string]*registeredCrawler),
		logFiles:     make(map[string]*os.File),
		loggerPrefix: "[ease-engine] ",
	}

	// 让根分组可以访问引擎。
	e.engine = e
	e.engineLogger.SetPrefix(e.loggerPrefix)
	return e
}

// SetLogRootDir 设置日志根目录。
// 空字符串会回退为默认值 "logs"。
func (e *Engine) SetLogRootDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "logs"
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logRootDir = dir
}

// 获取单个插件
func (e *Engine) GetCrawler(key string) *registeredCrawler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.crawlers[key]
}

// 获取所有插件
func (e *Engine) GetCrawlers() map[string]*registeredCrawler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]*registeredCrawler, len(e.crawlers))
	for k, v := range e.crawlers {
		out[k] = v
	}
	return out
}

// 开发模式运行单个插件
func (e *Engine) DevRun(key string) {
	crawler := e.GetCrawler(key)
	if crawler == nil {
		e.engineLogger.Errorf("插件不存在: %s", key)
		return
	}
	logger := Logger
	cctx := &Context{}
	cctx.Set(ContextLoggerKey, logger)
	crawler.crawler.Run(cctx)
}

// Run 使用 background context 启动所有注册插件。
// 该函数会阻塞（直到所有调度 goroutine 退出，通常需配合 RunWithContext）。
func (g *CrawlerGroup) Run() {
	g.RunWithContext(context.Background())
}

// RunWithContext 启动所有注册插件，并受 ctx 生命周期控制。
// 当 ctx.Done() 触发后，各插件循环会退出，RunWithContext 返回。
func (g *CrawlerGroup) RunWithContext(ctx context.Context) {
	g.engine.mu.RLock()
	items := make([]*registeredCrawler, 0, len(g.engine.crawlers))
	for _, c := range g.engine.crawlers {
		items = append(items, c)
	}
	g.engine.mu.RUnlock()

	g.engine.engineLogger.Printf("crawler引擎启动, 插件总数=%d", len(items))
	for _, item := range items {
		meta := item.crawler.Meta()
		g.engine.engineLogger.Printf("插件发现: 名称=%s, 元信息={间隔=%s,启动即跑=%t,自定义日志=%t}", item.name, meta.Interval, meta.StartImmediately, meta.Logger != nil)
	}

	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(it *registeredCrawler) {
			defer wg.Done()
			g.engine.runCrawlerLoop(ctx, it)
		}(item)
	}
	wg.Wait()
}

// runCrawlerLoop 执行单个插件的调度循环。
//
// 行为：
//  1. 读取 Meta（Interval / StartImmediately / Logger）
//  2. 可选先执行一次
//  3. 按 Interval 周期执行
//  4. 收到 ctx.Done 后停止
func (e *Engine) runCrawlerLoop(ctx context.Context, item *registeredCrawler) {
	meta := item.crawler.Meta()
	interval := meta.Interval
	if interval <= 0 {
		interval = time.Second
	}

	runOnce := func() {
		defer func() {
			if r := recover(); r != nil {
				item.log.Errorf("插件panic: %v\n%s", r, string(debug.Stack()))
				e.logf(item.path, item.name, "插件panic已恢复: %v", r)
			}
		}()

		logger := item.log
		if meta.Logger != nil {
			logger = meta.Logger
		}
		cctx := &Context{}
		cctx.Set(ContextLoggerKey, logger)

		if err := item.crawler.Run(cctx); err != nil {
			e.logf(item.path, item.name, "运行失败: %v", err)
			return
		}
		e.logf(item.path, item.name, "运行成功")
	}

	if meta.StartImmediately {
		runOnce()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			e.logf(item.path, item.name, "停止运行")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

// Register 在当前分组下注册一个插件。
//
// 关键规则：
//  1. 名称不能为空
//  2. 同一 groupPath/name 不能重复注册
//  3. 日志路径统一为 logs/<group-path>/<crawler>.log
//  4. 注册失败不 panic，记录日志并返回（降级处理）
func (g *CrawlerGroup) Register(crawler Crawler) {
	name := strings.TrimSpace(crawler.Name())
	if name == "" {
		g.engine.logf(g.path, "unknown", "注册crawler失败: 空名称")
		return
	}

	groupPath := cleanGroupPath(g.path)
	key := filepath.ToSlash(filepath.Join(groupPath, name))
	filePath := g.engine.crawlerLogPath(groupPath, name)

	g.engine.mu.Lock()
	defer g.engine.mu.Unlock()

	if _, exists := g.engine.crawlers[key]; exists {
		g.engine.logf(groupPath, name, "注册失败: 重复的crawler")
		return
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		g.engine.logf(groupPath, name, "注册失败: 创建目录失败: %v", err)
		return
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		g.engine.logf(groupPath, name, "注册失败: 打开日志文件失败: %v", err)
		return
	}

	logger := NewLogger(f, "", log.LstdFlags)
	g.engine.crawlers[key] = &registeredCrawler{
		path:    groupPath,
		name:    name,
		crawler: crawler,
		log:     logger,
	}
	g.engine.logFiles[key] = f
}

// Group 获取/创建子分组（支持链式调用）。
//
// 行为：
//  1. 空名称直接返回当前分组
//  2. 已存在子分组则复用
//  3. 新建子分组并初始化对应日志目录
func (g *CrawlerGroup) Group(name string) *CrawlerGroup {
	name = strings.TrimSpace(name)
	if name == "" {
		return g
	}

	g.engine.mu.Lock()
	defer g.engine.mu.Unlock()

	if child, ok := g.children[name]; ok {
		return child
	}

	newPath := filepath.ToSlash(filepath.Join(cleanGroupPath(g.path), name))
	newGroup := &CrawlerGroup{
		path:     newPath,
		engine:   g.engine,
		parent:   g,
		children: make(map[string]*CrawlerGroup),
	}
	g.children[name] = newGroup

	groupDir := filepath.Join(g.engine.logRootDir, filepath.FromSlash(newPath))
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		g.engine.logf(newPath, "group", "创建目录失败: %v", err)
	}
	return newGroup
}

// Close 关闭引擎持有的所有日志文件句柄。
// 建议在程序退出前调用，避免文件句柄泄漏。
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var firstErr error
	for key, f := range e.logFiles {
		if f == nil {
			continue
		}
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(e.logFiles, key)
	}
	return firstErr
}

// crawlerLogPath 计算单个插件日志文件路径：
// logs/<group-path>/<crawler>.log
func (e *Engine) crawlerLogPath(groupPath, crawlerName string) string {
	groupPath = cleanGroupPath(groupPath)
	fileName := sanitizeFileName(crawlerName) + ".log"
	return filepath.Join(e.logRootDir, filepath.FromSlash(groupPath), fileName)
}

// cleanGroupPath 规范化分组路径并做安全处理。
//
// 处理逻辑：
//   - 统一分隔符为 '/'
//   - 去掉首尾 '/'
//   - 替换 ".." 防止目录穿越
//   - 空路径回退到日志根目录
func cleanGroupPath(path string) string {
	p := strings.TrimSpace(path)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.Trim(p, "/")
	p = strings.ReplaceAll(p, "..", "_")
	if p == "" {
		return "/"
	}
	return p
}

// sanitizeFileName 清洗插件名称，避免非法文件名字符。
func sanitizeFileName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "unknown"
	}
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return r.Replace(n)
}

// logf 输出框架级日志（写到全局 logger）。
// 格式：[groupPath][crawlerName] message
func (e *Engine) logf(path, name, format string, args ...any) {
	e.engineLogger.Printf("[%s][%s] %s", cleanGroupPath(path), name, fmt.Sprintf(format, args...))
}
