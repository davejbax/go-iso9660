package encode

import (
	"github.com/davejbax/go-iso9660/internal/spec"
	"time"
)

func AsDateTime(t time.Time) spec.DateTime {
	t = t.UTC()
	return spec.DateTime{
		YearsSince1900:            uint8(t.Year() - 1900),
		Month:                     uint8(t.Month()),
		Day:                       uint8(t.Day()),
		Hour:                      uint8(t.Hour()),
		Minute:                    uint8(t.Minute()),
		Second:                    uint8(t.Second()),
		GMTOffsetIn15MinIntervals: 0,
	}
}
