package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

var levelMap = map[string]int{
	"FATAL": 1,
	"ERROR": 2,
	"WARN":  3,
	"INFO":  4,
	"DEBUG": 5,
}

type EaseLogger struct {
	logWriter *log.Logger
	mu        sync.Mutex // 互斥锁保证并发写安全
	out       io.Writer
	level     int
}

// NewLogger 创建日志实例
// out: 日志输出目标（如 os.Stdout、文件句柄等）
// prefix: 日志全局前缀（如 [easecrawler]）
// flags: 日志标志（如 log.LstdFlags 包含时间戳）
func NewLogger(out io.Writer, prefix string, flags int, level string) *EaseLogger {
	if out == nil {
		out = os.Stdout // 兜底，避免 nil writer
	}
	return &EaseLogger{
		logWriter: log.New(out, prefix, flags),
		mu:        sync.Mutex{},
		out:       out,
		level:     levelMap[level],
	}
}

// InitLogger 初始化全局日志实例（方便用户自定义）
func InitLogger(out io.Writer, prefix string, flags int, level string) {
	Logger = NewLogger(out, prefix, flags, level)
}

func (l *EaseLogger) SetPrefix(prefix string) {
	l.logWriter.SetPrefix(prefix)
}

func (l *EaseLogger) GetPrefix() string {
	return l.logWriter.Prefix()
}

func (l *EaseLogger) SetLevelOfOutput(level string) {
	l.level = levelMap[level]
}

// 通用日志方法，提取重复逻辑
func (l *EaseLogger) log(level string, format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level > levelMap[level] {
		return
	}
	fullFormat := fmt.Sprintf("[%s] %s", level, format)
	l.logWriter.Printf(fullFormat, v...)
}

// ========== INFO 级别日志 ==========
func (l *EaseLogger) Infof(format string, v ...any) {
	l.log("INFO", format, v...)
}

func (l *EaseLogger) Infoln(v ...any) {
	// 动态生成格式化字符串，贴合标准库 ln 方法的空格分隔行为
	format := strings.Repeat("%v ", len(v))
	format = strings.TrimSuffix(format, " ")
	l.log("INFO", format, v...)
}

func (l *EaseLogger) Info(v ...any) {
	l.log("INFO", "%v", v...)
}

// ========== ERROR 级别日志 ==========
func (l *EaseLogger) Errorf(format string, v ...any) {
	l.log("ERROR", format, v...)
}

func (l *EaseLogger) Errorln(v ...any) {
	format := strings.Repeat("%v ", len(v))
	format = strings.TrimSuffix(format, " ")
	l.log("ERROR", format, v...)
}

func (l *EaseLogger) Error(v ...any) {
	l.log("ERROR", "%v", v...)
}

// ========== WARN 级别日志 ==========
func (l *EaseLogger) Warnf(format string, v ...any) {
	l.log("WARN", format, v...)
}

func (l *EaseLogger) Warnln(v ...any) {
	format := strings.Repeat("%v ", len(v))
	format = strings.TrimSuffix(format, " ")
	l.log("WARN", format, v...)
}

func (l *EaseLogger) Warn(v ...any) {
	l.log("WARN", "%v", v...)
}

// ========== FATAL 级别日志 ==========
func (l *EaseLogger) Fatalf(format string, v ...any) {
	l.log("FATAL", format, v...)
	os.Exit(1) // 符合标准库 Fatal 行为：打印后退出
}

func (l *EaseLogger) Fatalln(v ...any) {
	format := strings.Repeat("%v ", len(v))
	format = strings.TrimSuffix(format, " ")
	l.log("FATAL", format, v...)
	os.Exit(1)
}

func (l *EaseLogger) Fatal(v ...any) {
	l.log("FATAL", "%v", v...)
	os.Exit(1)
}

// ========== DEBUG 级别日志 ==========
func (l *EaseLogger) Debugf(format string, v ...any) {
	l.log("DEBUG", format, v...)
}

func (l *EaseLogger) Debugln(v ...any) {
	format := strings.Repeat("%v ", len(v))
	format = strings.TrimSuffix(format, " ")
	l.log("DEBUG", format, v...)
}

func (l *EaseLogger) Debug(v ...any) {
	l.log("DEBUG", "%v", v...)
}

// 全局日志实例，默认输出到标准输出，前缀 [ease]，包含标准日志标志
var Logger *EaseLogger = NewLogger(os.Stdout, "[ease] ", log.LstdFlags, "INFO")

// 全局日志方法
// ========== INFO 级别日志 ==========
func Infof(format string, v ...any) {
	Logger.Infof(format, v...)
}

func Infoln(v ...any) {
	Logger.Infoln(v...)
}

func Info(v ...any) {
	Logger.Info(v...)
}

// ========== ERROR 级别日志 ==========
func Errorf(format string, v ...any) {
	Logger.Errorf(format, v...)
}

func Errorln(v ...any) {
	Logger.Errorln(v...)
}

func Error(v ...any) {
	Logger.Error(v...)
}

// ========== WARN 级别日志 ==========
func Warnf(format string, v ...any) {
	Logger.Warnf(format, v...)
}

func Warnln(v ...any) {
	Logger.Warnln(v...)
}

func Warn(v ...any) {
	Logger.Warn(v...)
}

// ========== FATAL 级别日志 ==========
func Fatalf(format string, v ...any) {
	Logger.Fatalf(format, v...)
}

func Fatalln(v ...any) {
	Logger.Fatalln(v...)
}

func Fatal(v ...any) {
	Logger.Fatal(v...)
}

// ========== DEBUG 级别日志 ==========
func Debugf(format string, v ...any) {
	Logger.Debugf(format, v...)
}

func Debugln(v ...any) {
	Logger.Debugln(v...)
}

func Debug(v ...any) {
	Logger.Debug(v...)
}
