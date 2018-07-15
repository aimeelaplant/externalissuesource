package dateutil

import "time"

// Compares how many months from dateOne to dateTwo.
func CompareMonths(dateOne time.Time, dateTwo time.Time) int {
	return int(dateOne.Sub(dateTwo).Hours() / 24 / 30)
}
