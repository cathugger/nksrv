package logx

import (
	"io"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	NOTICE
	WARN
	ERROR
	CRITICAL
	LevelCount
)

type LoggerX interface {
	Level() Level
	LogPrintX(section string, lvl Level, v ...interface{})
	LogPrintlnX(section string, lvl Level, v ...interface{})
	LogPrintfX(section string, lvl Level, fmt string, v ...interface{})
	LockWriteX(section string, lvl Level) bool
	io.WriteCloser
}

type Logger interface {
	Level() Level
	LogPrint(lvl Level, v ...interface{})
	LogPrintln(lvl Level, v ...interface{})
	LogPrintf(lvl Level, fmt string, v ...interface{})
	LockWrite(lvl Level) bool
	io.WriteCloser
}

var _ Logger = LogToX{}

type LogToX struct {
	section string
	logx    LoggerX
}

func (l LogToX) Level() Level {
	return l.logx.Level()
}
func (l LogToX) LogPrint(lvl Level, v ...interface{}) {
	l.logx.LogPrintX(l.section, lvl, v...)
}
func (l LogToX) LogPrintln(lvl Level, v ...interface{}) {
	l.logx.LogPrintlnX(l.section, lvl, v...)
}
func (l LogToX) LogPrintf(lvl Level, fmt string, v ...interface{}) {
	l.logx.LogPrintfX(l.section, lvl, fmt, v...)
}
func (l LogToX) LockWrite(lvl Level) bool {
	return l.logx.LockWriteX(l.section, lvl)
}
func (l LogToX) Close() error {
	return l.logx.Close()
}
func (l LogToX) Write(b []byte) (int, error) {
	return l.logx.Write(b)
}
func NewLogToX(logx LoggerX, section string) LogToX {
	return LogToX{section: section, logx: logx}
}

var _ Logger = (*LogToXLevel)(nil)

type LogToXLevel struct {
	section string
	logx    LoggerX
	lw      io.WriteCloser
	level   Level
}

func (l *LogToXLevel) Level() Level {
	return l.level
}
func (l *LogToXLevel) LogPrint(lvl Level, v ...interface{}) {
	if lvl >= l.level {
		l.logx.LogPrintX(l.section, lvl, v...)
	}
}
func (l *LogToXLevel) LogPrintln(lvl Level, v ...interface{}) {
	if lvl >= l.level {
		l.logx.LogPrintlnX(l.section, lvl, v...)
	}
}
func (l *LogToXLevel) LogPrintf(lvl Level, fmt string, v ...interface{}) {
	if lvl >= l.level {
		l.logx.LogPrintfX(l.section, lvl, fmt, v...)
	}
}
func (l *LogToXLevel) LockWrite(lvl Level) bool {
	if lvl >= l.level && l.logx.LockWriteX(l.section, lvl) {
		l.lw = l.logx
		return true
	} else {
		l.lw = nilLogWriter{}
		return false
	}
}
func (l *LogToXLevel) Close() error {
	return l.lw.Close()
}
func (l *LogToXLevel) Write(b []byte) (int, error) {
	return l.lw.Write(b)
}
func NewLogToXLevel(logx LoggerX, section string, l Level) LogToXLevel {
	dl := logx.Level()
	if dl > l {
		l = dl
	}
	return LogToXLevel{section: section, logx: logx, level: l}
}

var _ io.WriteCloser = nilLogWriter{}

type nilLogWriter struct{}

func (nilLogWriter) Close() error {
	return nil
}
func (nilLogWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func NewWriteToLog(log Logger, lvl Level) io.WriteCloser {
	doit := log.LockWrite(lvl)
	if doit {
		return log
	} else {
		return nilLogWriter{}
	}
}
