package event

import "time"

type Record struct {
	Tag   string
	Time  time.Time
	Value interface{}
}

func NewRecord(tag string, v interface{}) *Record {
	return NewRecordWithTime(tag, time.Now(), v)
}

func NewRecordWithTime(tag string, time time.Time, v interface{}) *Record {
	return &Record{Tag: tag, Time: time, Value: v}
}
