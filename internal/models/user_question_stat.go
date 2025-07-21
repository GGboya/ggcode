package models

import "time"

type DailyStat struct {
	Date  time.Time
	Count int64
}

type CheckInStat struct {
	Date       time.Time
	StudyCount int
}
