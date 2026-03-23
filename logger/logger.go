package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level 日志级别，数值越大表示级别越高（更重要）
type Level int

const (
	DebugLevel Level = iota + 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var levelToString = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

var stringToLevel = map[string]Level{
	"DEBUG": DebugLevel,
	"INFO":  InfoLevel,
	"WARN":  WarnLevel,
	"ERROR": ErrorLevel,
	"FATAL": FatalLevel,
}

func parseLevel(level string) (Level, bool) {
	if lv, ok := stringToLevel[strings.ToUpper(strings.TrimSpace(level))]; ok {
		return lv, true
	}
	return InfoLevel, false
}

type EaseLogger struct {
	logWriter *log.Logger
	mu        sync.Mutex // 互斥锁保证并发写安全
	level     Level
	exitFunc  func(code int)
}

// NewLogger 创建日志实例
// out: 日志输出目标（如 os.Stdout、文件句柄等）
// prefix: 日志全局前缀（如 [easecrawler]）
// flags: 日志标志（如 log.LstdFlags 包含时间戳）
// level: 日志级别（DEBUG/INFO/WARN/ERROR/FATAL），无效值默认 INFO
func NewLogger(out io.Writer, prefix string, flags int) *EaseLogger {
	if out == nil {
		out = os.Stdout // 兜底，避免 nil writer
	}
	return &EaseLogger{
		logWriter: log.New(out, prefix, flags),
		level:     InfoLevel,
		exitFunc:  os.Exit,
	}
}

func NewLoggerWithLevel(out io.Writer, prefix string, flags int, level string) *EaseLogger {
	if out == nil {
		out = os.Stdout
	}
	lv, _ := parseLevel(level)
	return &EaseLogger{
		logWriter: log.New(out, prefix, flags),
		level:     lv,
		exitFunc:  os.Exit,
	}
}

// NewLoggerMultiWriter 使用多个 writer 输出日志，默认配置与全局 Logger 一致。
// writers 为空或全为 nil 时，会回退到 os.Stdout。
func NewLoggerMultiWriter(writers ...io.Writer) *EaseLogger {
	validWriters := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		if w != nil {
			validWriters = append(validWriters, w)
		}
	}

	if len(validWriters) == 0 {
		return NewLogger(os.Stdout, "[ease] ", log.LstdFlags)
	}

	return NewLogger(io.MultiWriter(validWriters...), "[ease] ", log.LstdFlags)
}

// InitLogger 初始化全局日志实例（方便用户自定义）
func InitLogger(out io.Writer, prefix string, flags int, level string) {
	Logger = NewLoggerWithLevel(out, prefix, flags, level)
}

func (l *EaseLogger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logWriter.SetPrefix(prefix)
}

func (l *EaseLogger) GetPrefix() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logWriter.Prefix()
}

// SetLevelOfOutput 设置日志输出级别，返回是否设置成功。
// level 无效时不会变更当前级别，并返回 false。
func (l *EaseLogger) SetLevelOfOutput(level string) bool {
	lv, ok := parseLevel(level)
	if !ok {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = lv
	return true
}

// GetLevel 返回当前日志级别字符串。
func (l *EaseLogger) GetLevel() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if name, ok := levelToString[l.level]; ok {
		return name
	}
	return "INFO"
}

func (l *EaseLogger) shouldLog(level Level) bool {
	return level >= l.level
}

// 通用日志方法，提取重复逻辑
func (l *EaseLogger) log(level Level, format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.shouldLog(level) {
		return
	}
	fullFormat := fmt.Sprintf("[%s] %s", levelToString[level], format)
	l.logWriter.Printf(fullFormat, v...)
}

func (l *EaseLogger) logln(level Level, v ...any) {
	msg := strings.TrimSuffix(fmt.Sprintln(v...), "\n")
	l.log(level, "%s", msg)
}

func (l *EaseLogger) logplain(level Level, v ...any) {
	msg := fmt.Sprint(v...)
	l.log(level, "%s", msg)
}

// ========== INFO 级别日志 ==========
func (l *EaseLogger) Infof(format string, v ...any) {
	l.log(InfoLevel, format, v...)
}

func (l *EaseLogger) Infoln(v ...any) {
	l.logln(InfoLevel, v...)
}

func (l *EaseLogger) Info(v ...any) {
	l.logplain(InfoLevel, v...)
}

// ========== ERROR 级别日志 ==========
func (l *EaseLogger) Errorf(format string, v ...any) {
	l.log(ErrorLevel, format, v...)
}

func (l *EaseLogger) Errorln(v ...any) {
	l.logln(ErrorLevel, v...)
}

func (l *EaseLogger) Error(v ...any) {
	l.logplain(ErrorLevel, v...)
}

// ========== WARN 级别日志 ==========
func (l *EaseLogger) Warnf(format string, v ...any) {
	l.log(WarnLevel, format, v...)
}

func (l *EaseLogger) Warnln(v ...any) {
	l.logln(WarnLevel, v...)
}

func (l *EaseLogger) Warn(v ...any) {
	l.logplain(WarnLevel, v...)
}

// ========== FATAL 级别日志 ==========
func (l *EaseLogger) Fatalf(format string, v ...any) {
	l.log(FatalLevel, format, v...)
	l.exitFunc(1) // 符合标准库 Fatal 行为：打印后退出
}

func (l *EaseLogger) Fatalln(v ...any) {
	l.logln(FatalLevel, v...)
	l.exitFunc(1)
}

func (l *EaseLogger) Fatal(v ...any) {
	l.logplain(FatalLevel, v...)
	l.exitFunc(1)
}

// ========== DEBUG 级别日志 ==========
func (l *EaseLogger) Debugf(format string, v ...any) {
	l.log(DebugLevel, format, v...)
}

func (l *EaseLogger) Debugln(v ...any) {
	l.logln(DebugLevel, v...)
}

func (l *EaseLogger) Debug(v ...any) {
	l.logplain(DebugLevel, v...)
}

// 全局日志实例，默认输出到标准输出，前缀 [ease]，包含标准日志标志
var Logger *EaseLogger = NewLogger(os.Stdout, "[ease] ", log.LstdFlags)

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
