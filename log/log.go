package log

import (
	"fmt"
	"os"

	"github.com/yosisa/fluxion/event"
)

type level int

func (l level) String() string {
	switch l {
	case lvDebug:
		return "debug"
	case lvInfo:
		return "info"
	case lvNotice:
		return "notice"
	case lvWarning:
		return "warning"
	case lvError:
		return "error"
	case lvCritical:
		return "critical"
	}
	return ""
}

const (
	lvDebug level = iota
	lvInfo
	lvNotice
	lvWarning
	lvError
	lvCritical
)

type Logger struct {
	Name     string
	Prefix   string
	EmitFunc func(*event.Record)
}

func (l *Logger) emit(lv level, msg string) {
	lvStr := lv.String()
	v := map[string]interface{}{
		"name":    l.Name,
		"level":   lvStr,
		"message": l.Prefix + msg,
	}
	l.EmitFunc(event.NewRecord("fluxion.log."+lvStr, v))
}

func (l *Logger) log(lv level, v ...interface{}) {
	l.emit(lv, fmt.Sprint(v...))
}

func (l *Logger) logf(lv level, format string, v ...interface{}) {
	l.emit(lv, fmt.Sprintf(format, v...))
}

func (l *Logger) Critical(v ...interface{}) {
	l.log(lvCritical, v...)
}

func (l *Logger) Criticalf(format string, v ...interface{}) {
	l.logf(lvCritical, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.log(lvError, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logf(lvError, format, v...)
}

func (l *Logger) Warning(v ...interface{}) {
	l.log(lvWarning, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.logf(lvWarning, format, v...)
}

func (l *Logger) Notice(v ...interface{}) {
	l.log(lvNotice, v...)
}

func (l *Logger) Noticef(format string, v ...interface{}) {
	l.logf(lvNotice, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.log(lvInfo, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.logf(lvInfo, format, v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.log(lvDebug, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logf(lvDebug, format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.log(lvCritical, v...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logf(lvCritical, format, v...)
	os.Exit(1)
}
