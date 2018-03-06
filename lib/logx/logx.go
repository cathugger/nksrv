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
	UnlockWriteX()
	io.Writer
}

type LogWriter interface {
	LockWrite(lvl Level) bool
	UnlockWrite()
	io.Writer
}

type Logger interface {
	Level() Level
	LogPrint(lvl Level, v ...interface{})
	LogPrintln(lvl Level, v ...interface{})
	LogPrintf(lvl Level, fmt string, v ...interface{})
	LogWriter
}

type LogToX struct {
	section string
	logx    LoggerX
}

func (l LogToX) Level() Level                           { return l.Level() }
func (l LogToX) LogPrint(lvl Level, v ...interface{})   { l.logx.LogPrintX(l.section, lvl, v...) }
func (l LogToX) LogPrintln(lvl Level, v ...interface{}) { l.logx.LogPrintlnX(l.section, lvl, v...) }
func (l LogToX) LogPrintf(lvl Level, fmt string, v ...interface{}) {
	l.logx.LogPrintfX(l.section, lvl, fmt, v...)
}
func (l LogToX) LockWrite(lvl Level) bool           { return l.logx.LockWriteX(l.section, lvl) }
func (l LogToX) UnlockWrite()                       { l.logx.UnlockWriteX() }
func (l LogToX) Write(b []byte) (int, error)        { return l.logx.Write(b) }
func NewLogToX(logx LoggerX, section string) LogToX { return LogToX{section: section, logx: logx} }

var _ Logger = LogToX{}

type nilLogWriter struct{}

func (nilLogWriter) LockWrite(lvl Level) bool    { return false }
func (nilLogWriter) UnlockWrite()                {}
func (nilLogWriter) Write(b []byte) (int, error) { return len(b), nil }

type WriteToLog struct {
	log LogWriter
}

func NewWriteToLog(log Logger, lvl Level) WriteToLog {
	doit := log.LockWrite(lvl)
	if doit {
		return WriteToLog{log: log}
	} else {
		return WriteToLog{log: nilLogWriter{}}
	}
}

func (w WriteToLog) Write(b []byte) (int, error) {
	return w.log.Write(b)
}

func (w WriteToLog) Close() error {
	w.log.UnlockWrite()
	return nil
}

var _ io.Writer = WriteToLog{}
var _ io.Closer = WriteToLog{}
