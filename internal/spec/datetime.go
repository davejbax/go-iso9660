package spec

import "time"

// DateTime is a numerical representation of a date and time
//
// ECMA-119 (5th ed.) §10.1.6
type DateTime struct {
	YearsSince1900            uint8
	Month                     uint8
	Day                       uint8
	Hour                      uint8
	Minute                    uint8
	Second                    uint8
	GMTOffsetIn15MinIntervals int8
}

func (d DateTime) Time() time.Time {
	return time.Date(
		int(d.YearsSince1900)+1900,
		time.Month(d.Month),
		int(d.Day),
		int(d.Hour),
		int(d.Minute),
		int(d.Second),
		0,
		time.FixedZone("", int(d.GMTOffsetIn15MinIntervals*15*60)),
	)
}

// LongDateTime is a character (digit) representation of date and time
//
// ECMA-119 (5th ed.) §9.4.27.2
type LongDateTime struct {
	YearDigits                [4]uint8
	MonthDigits               [2]uint8
	DayDigits                 [2]uint8
	HourDigits                [2]uint8
	MinuteDigits              [2]uint8
	SecondDigits              [2]uint8
	CentisecondsDigits        [2]uint8
	GMTOffsetIn15MinIntervals uint8
}

// ZeroLongDateTime represents the zero-value of the [LongDateTime] type
//
// ECMA-119 (5th ed.) §9.4.27.2
var ZeroLongDateTime = LongDateTime{
	YearDigits:                [4]uint8{'0', '0', '0', '0'},
	MonthDigits:               [2]uint8{'0', '0'},
	DayDigits:                 [2]uint8{'0', '0'},
	HourDigits:                [2]uint8{'0', '0'},
	MinuteDigits:              [2]uint8{'0', '0'},
	SecondDigits:              [2]uint8{'0', '0'},
	CentisecondsDigits:        [2]uint8{'0', '0'},
	GMTOffsetIn15MinIntervals: 0,
}
