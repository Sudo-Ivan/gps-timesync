package nmea

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidNMEAData is returned when NMEA data is invalid.
var ErrInvalidNMEAData = errors.New("invalid NMEA data")

// ParseNMEATime parses time and date from NMEA sentence.
// It expects time in HHMMSS format and date in DDMMYY format.
func ParseNMEATime(timeStr, dateStr string) (time.Time, error) {
	if len(timeStr) < 6 || len(dateStr) != 6 {
		return time.Time{}, ErrInvalidNMEAData
	}

	hour := timeStr[0:2]
	min := timeStr[2:4]
	sec := timeStr[4:6]
	day := dateStr[0:2]
	month := dateStr[2:4]
	year := "20" + dateStr[4:6]

	timeStr = fmt.Sprintf("%s-%s-%s %s:%s:%s", year, month, day, hour, min, sec)
	return time.Parse("2006-01-02 15:04:05", timeStr)
}
