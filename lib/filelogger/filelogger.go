package filelogger

import (
	"../logx"
	"bufio"
	"fmt"
	colorable "github.com/mattn/go-colorable"
	isatty "github.com/mattn/go-isatty"
	"os"
	"sync"
	"time"
)

type useColor int

const (
	ColorAuto useColor = iota
	ColorOn
	ColorOff
)

type logLevels [logx.LevelCount][]byte

var levelstrings = [2]logLevels{
	// uncolored
	{
		logx.DEBUG:    []byte("   DEBUG"),
		logx.INFO:     []byte("    INFO"),
		logx.NOTICE:   []byte("  NOTICE"),
		logx.WARN:     []byte(" WARNING"),
		logx.ERROR:    []byte("   ERROR"),
		logx.CRITICAL: []byte("CRITICAL"),
	},
	// colored
	{
		logx.DEBUG:    []byte("\033[37m   DEBUG\033[0m"),
		logx.INFO:     []byte("\033[34m    INFO\033[0m"),
		logx.NOTICE:   []byte("\033[32m  NOTICE\033[0m"),
		logx.WARN:     []byte("\033[33m WARNING\033[0m"),
		logx.ERROR:    []byte("\033[31m   ERROR\033[0m"),
		logx.CRITICAL: []byte("\033[35mCRITICAL\033[0m"),
	},
}

var formatstrings = [2]string{
	// uncolored
	" %s [%s] ",
	// colored
	" %s [\033[36m%s\033[0m] ",
}

type day struct {
	Y int
	M time.Month
	D int
}

var _ logx.LoggerX = (*FileLogger)(nil)

type FileLogger struct {
	w splitter
	d day
	f *os.File
	l sync.Mutex
	t uint
	m logx.Level
	n bool
}

func nowTime() time.Time {
	return time.Now()
}

func NewFileLogger(f *os.File, logLevel logx.Level, c useColor) (*FileLogger, error) {
	l := &FileLogger{f: f, t: uint(c) & 1, m: logLevel}
	fd := f.Fd()
	if c != ColorOff && (isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)) {
		l.w.w = bufio.NewWriter(colorable.NewColorable(f))
		l.t = 1
	} else {
		l.w.w = bufio.NewWriter(f)
	}
	return l, nil
}

func (l *FileLogger) Level() logx.Level {
	return l.m
}

func (l *FileLogger) writeTime(t time.Time) {
	var d day
	d.Y, d.M, d.D = t.Date()
	h, m, s := t.Hour(), t.Minute(), t.Second()
	//_, z := t.Zone()
	if l.t != 0 {
		if l.d != d {
			l.d = d
			fmt.Fprintf(&l.w, "\033[1mdate is %d-%02d-%02d\033[0m\n", d.Y, d.M, d.D)
		}
		fmt.Fprintf(&l.w.p, "%02d:%02d:%02d", h, m, s)
	} else {
		fmt.Fprintf(&l.w.p, "%d-%02d-%02d %02d:%02d:%02d", d.Y, d.M, d.D, h, m, s)
	}
}

func (l *FileLogger) prepareWrite(section string, lvl logx.Level, t time.Time) {
	l.w.reset()
	l.writeTime(t)
	fmt.Fprintf(&l.w.p, formatstrings[l.t], levelstrings[l.t][lvl], section)
}

func (l *FileLogger) LogPrintX(section string, lvl logx.Level, v ...interface{}) {
	if l.m > lvl {
		return
	}

	t := time.Now().UTC()

	l.l.Lock()
	defer l.l.Unlock()

	l.prepareWrite(section, lvl, t)

	fmt.Fprint(&l.w, v...)
	l.w.finish()
}

func (l *FileLogger) LogPrintlnX(section string, lvl logx.Level, v ...interface{}) {
	if l.m > lvl {
		return
	}

	t := time.Now().UTC()

	l.l.Lock()
	defer l.l.Unlock()

	l.prepareWrite(section, lvl, t)

	fmt.Fprintln(&l.w, v...)
	l.w.finish()
}

func (l *FileLogger) LogPrintfX(section string, lvl logx.Level, fmts string, v ...interface{}) {
	if l.m > lvl {
		return
	}

	t := time.Now().UTC()

	l.l.Lock()
	defer l.l.Unlock()

	l.prepareWrite(section, lvl, t)

	fmt.Fprintf(&l.w, fmts, v...)
	l.w.finish()
}

func (l *FileLogger) LockWriteX(section string, lvl logx.Level) bool {
	if l.m > lvl {
		return false
	}

	t := nowTime()
	l.l.Lock()
	l.prepareWrite(section, lvl, t)

	return true
}

func (l *FileLogger) Close() error {
	l.w.finish()
	l.l.Unlock()
	return nil
}

func (l *FileLogger) Write(b []byte) (int, error) {
	return l.w.Write(b)
}
