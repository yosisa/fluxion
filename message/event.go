package message

import "time"

type Event struct {
	Tag    string                 `codec:"tag"`
	Time   time.Time              `codec:"time"`
	Record map[string]interface{} `codec:"record"`
}

func NewEvent(tag string, v map[string]interface{}) *Event {
	return NewEventWithTime(tag, time.Now(), v)
}

func NewEventWithTime(tag string, time time.Time, r map[string]interface{}) *Event {
	return &Event{Tag: tag, Time: time, Record: r}
}
