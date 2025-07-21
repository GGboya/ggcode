package models

import "time"

type DailyStat struct {
	Date  time.Time
	Count int64
}

// CheckInStat 打卡统计
type CheckInStat struct {
	CheckedInToday   bool  `json:"checked_in_today"`    // 今日是否已打卡
	TotalCheckInDays int64 `json:"total_check_in_days"` // 总打卡天数
	ConsecutiveDays  int64 `json:"consecutive_days"`    // 当前连续打卡天数
	BestStreak       int64 `json:"best_streak"`         // 历史最长连续天数
}
