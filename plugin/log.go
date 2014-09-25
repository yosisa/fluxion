package plugin

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

type logger struct {
	Name     string
	Prefix   string
	emitFunc func(*event.Record)
}

func (l *logger) emit(lv level, msg string) {
	lvStr := lv.String()
	v := map[string]interface{}{
		"name":    l.Name,
		"level":   lvStr,
		"message": l.Prefix + msg,
	}
	l.emitFunc(event.NewRecord("fluxion.log."+lvStr, v))
}

func (l *logger) log(lv level, v ...interface{}) {
	l.emit(lv, fmt.Sprint(v...))
}

func (l *logger) logf(lv level, format string, v ...interface{}) {
	l.emit(lv, fmt.Sprintf(format, v...))
}

func (l *logger) Critical(v ...interface{}) {
	l.log(lvCritical, v...)
}

func (l *logger) Criticalf(format string, v ...interface{}) {
	l.logf(lvCritical, format, v...)
}

func (l *logger) Error(v ...interface{}) {
	l.log(lvError, v...)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	l.logf(lvError, format, v...)
}

func (l *logger) Warning(v ...interface{}) {
	l.log(lvWarning, v...)
}

func (l *logger) Warningf(format string, v ...interface{}) {
	l.logf(lvWarning, format, v...)
}

func (l *logger) Notice(v ...interface{}) {
	l.log(lvNotice, v...)
}

func (l *logger) Noticef(format string, v ...interface{}) {
	l.logf(lvNotice, format, v...)
}

func (l *logger) Info(v ...interface{}) {
	l.log(lvInfo, v...)
}

func (l *logger) Infof(format string, v ...interface{}) {
	l.logf(lvInfo, format, v...)
}

func (l *logger) Debug(v ...interface{}) {
	l.log(lvDebug, v...)
}

func (l *logger) Debugf(format string, v ...interface{}) {
	l.logf(lvDebug, format, v...)
}

func (l *logger) Fatal(v ...interface{}) {
	l.log(lvCritical, v...)
	os.Exit(1)
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l.logf(lvCritical, format, v...)
	os.Exit(1)
}
