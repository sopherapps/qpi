package lib

import "time"

// GetUtcNow returns the current UTC time formatted as a string in
// the format "YYYY-mm-dd HH:MM:SS.000Z".
func GetUtcNow() string {
	return ToUtcString(time.Now())
}

// ToUtcString converts a time.Time to a string in the format
// "YYYY-mm-dd HH:MM:SS.000Z".
func ToUtcString(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000Z")
}
