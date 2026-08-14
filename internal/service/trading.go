package service

import (
	"time"
)

// EffectiveTradingDate 计算指定时间的交易归属日期：
//   - 周一至周五 15:00（含）前发生的操作归属当日；
//   - 15:00 后顺延至下一个交易日；
//   - 周末发生的操作顺延至下一个交易日。
//
// 简化实现：不考虑法定节假日（后续可扩展节假日历）。
func EffectiveTradingDate(now time.Time) time.Time {
	local := now.Local()
	result := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	// 周末顺延
	for isWeekend(result) {
		result = result.AddDate(0, 0, 1)
	}
	// 15:00 后顺延至下一交易日
	if !isWeekend(local) && local.Hour() >= 15 {
		result = result.AddDate(0, 0, 1)
		for isWeekend(result) {
			result = result.AddDate(0, 0, 1)
		}
	}
	return result
}

func isWeekend(t time.Time) bool {
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	default:
		return false
	}
}

// IsTradingDay 判断日期是否为交易日（周一至周五）。
func IsTradingDay(t time.Time) bool {
	return !isWeekend(t)
}
