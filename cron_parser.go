package lite_scheduler

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// CronExpr Cron表达式解析结果
type CronExpr struct {
	Second     []int // 0-59
	Minute     []int // 0-59
	Hour       []int // 0-23
	DayOfMonth []int // 1-31
	Month      []int // 1-12
	DayOfWeek  []int // 0-6 (0=Sunday)
}

// ParseCron 解析Cron表达式
// 支持格式: 秒 分 时 日 月 周
// 示例: "0 */5 * * * *" 每5分钟执行
func ParseCron(expr string) (*CronExpr, error) {
	fields := strings.Fields(expr)
	if len(fields) != 6 {
		return nil, errors.New("cron expression must have 6 fields")
	}

	cron := &CronExpr{}
	var err error

	if cron.Second, err = parseField(fields[0], 0, 59); err != nil {
		return nil, errors.New("invalid second field: " + err.Error())
	}
	if cron.Minute, err = parseField(fields[1], 0, 59); err != nil {
		return nil, errors.New("invalid minute field: " + err.Error())
	}
	if cron.Hour, err = parseField(fields[2], 0, 23); err != nil {
		return nil, errors.New("invalid hour field: " + err.Error())
	}
	if cron.DayOfMonth, err = parseField(fields[3], 1, 31); err != nil {
		return nil, errors.New("invalid day of month field: " + err.Error())
	}
	if cron.Month, err = parseField(fields[4], 1, 12); err != nil {
		return nil, errors.New("invalid month field: " + err.Error())
	}
	if cron.DayOfWeek, err = parseField(fields[5], 0, 6); err != nil {
		return nil, errors.New("invalid day of week field: " + err.Error())
	}

	return cron, nil
}

// parseField 解析单个字段
func parseField(field string, min, max int) ([]int, error) {
	var result []int

	// 处理逗号分隔的多个值
	parts := strings.Split(field, ",")
	for _, part := range parts {
		values, err := parseRange(part, min, max)
		if err != nil {
			return nil, err
		}
		result = append(result, values...)
	}

	// 去重并排序
	return unique(result), nil
}

// parseRange 解析范围表达式
func parseRange(expr string, min, max int) ([]int, error) {
	var result []int
	step := 1

	// 处理步长 */5 或 1-10/2
	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) != 2 {
			return nil, errors.New("invalid step expression")
		}
		var err error
		step, err = strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return nil, errors.New("invalid step value")
		}
		expr = parts[0]
	}

	// 处理 *
	if expr == "*" {
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// 处理范围 1-10
	if strings.Contains(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) != 2 {
			return nil, errors.New("invalid range expression")
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, errors.New("invalid range start")
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, errors.New("invalid range end")
		}
		if start < min || end > max || start > end {
			return nil, errors.New("range out of bounds")
		}
		for i := start; i <= end; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// 单个值
	val, err := strconv.Atoi(expr)
	if err != nil {
		return nil, errors.New("invalid value")
	}
	if val < min || val > max {
		return nil, errors.New("value out of bounds")
	}
	return []int{val}, nil
}

// unique 去重
func unique(nums []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0)
	for _, n := range nums {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result
}

// NextTime 计算下次执行时间
func (c *CronExpr) NextTime(from time.Time) time.Time {
	// 从下一秒开始
	t := from.Add(time.Second).Truncate(time.Second)

	// 最多搜索4年
	maxTime := from.Add(4 * 365 * 24 * time.Hour)

	for t.Before(maxTime) {
		// 检查月份
		if !contains(c.Month, int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}

		// 检查日期
		if !contains(c.DayOfMonth, t.Day()) || !contains(c.DayOfWeek, int(t.Weekday())) {
			t = t.Add(24 * time.Hour).Truncate(24 * time.Hour)
			continue
		}

		// 检查小时
		if !contains(c.Hour, t.Hour()) {
			t = t.Add(time.Hour).Truncate(time.Hour)
			continue
		}

		// 检查分钟
		if !contains(c.Minute, t.Minute()) {
			t = t.Add(time.Minute).Truncate(time.Minute)
			continue
		}

		// 检查秒
		if !contains(c.Second, t.Second()) {
			t = t.Add(time.Second)
			continue
		}

		return t
	}

	return time.Time{}
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
