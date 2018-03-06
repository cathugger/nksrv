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
	LogPrintX(section string, lvl Level, v ...interface{})
	LogPrintlnX(section string, lvl Level, v ...interface{})
	LogPrintfX(section string, lvl Level, fmt string, v ...interface{})
	LockWriteX(section string, lvl Level)
	UnlockWriteX()
	io.Writer
}

type Logger interface {
	LogPrint(lvl Level, v ...interface{})
	LogPrintln(lvl Level, v ...interface{})
	LogPrintf(lvl Level, fmt string, v ...interface{})
	LockWrite(lvl Level)
	UnlockWrite()
	io.Writer
}

type LogToX struct {
	section string
	logx    LoggerX
}

func (l LogToX) LogPrint(lvl Level, v ...interface{})   { l.logx.LogPrintX(l.section, lvl, v...) }
func (l LogToX) LogPrintln(lvl Level, v ...interface{}) { l.logx.LogPrintlnX(l.section, lvl, v...) }
func (l LogToX) LogPrintf(lvl Level, fmt string, v ...interface{}) {
	l.logx.LogPrintfX(l.section, lvl, fmt, v...)
}
func (l LogToX) LockWrite(lvl Level)                { l.logx.LockWriteX(l.section, lvl) }
func (l LogToX) UnlockWrite()                       { l.logx.UnlockWriteX() }
func (l LogToX) Write(b []byte) (int, error)        { return l.logx.Write(b) }
func NewLogToX(logx LoggerX, section string) LogToX { return LogToX{section: section, logx: logx} }

var _ Logger = LogToX{}

type WriteToLog struct {
	log Logger
}

func NewWriteToLog(log Logger, lvl Level) WriteToLog {
	log.LockWrite(lvl)
	return WriteToLog{log: log}
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
