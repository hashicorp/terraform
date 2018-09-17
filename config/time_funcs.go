package main

import (
	"fmt"
	"time"
)

type DateTimeDuration struct {
	Years        int
	Months       int
	Days         int
	Hours        int
	Minutes      int
	Seconds      int
	Milliseconds int
	Microseconds int
	Nanoseconds  int
}

func AddStructDate(d DateTimeDuration, t time.Time) time.Time {
	nt := t.AddDate(d.Years, d.Months, d.Days)
	return nt
}

func main() {
	dtd := DateTimeDuration{Years: 1, Months: 2, Days: 5}
	nd := AddStructDate(dtd, time.Date(2018, time.September, 16, 0, 0, 0, 0, time.UTC))
	fmt.Printf("Check out my sweet date %s", nd)
}
