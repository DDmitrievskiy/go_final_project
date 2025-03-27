package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "20060102"

var (
	ErrInvalidFormat   = errors.New("неверный формат правила")
	ErrInvalidDate     = errors.New("некорректная дата")
	ErrInvalidDay      = errors.New("недопустимое значение дня")
	ErrInvalidMonth    = errors.New("недопустимое значение месяца")
	ErrUnsupportedRule = errors.New("неподдерживаемый формат правила")
	ErrExceededMaxDays = errors.New("превышено максимальное количество дней")
)

func NextDate(now time.Time, dateStr string, repeat string) (string, error) {
	date, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		return "", ErrInvalidDate
	}

	if repeat == "" {
		return "", ErrInvalidFormat
	}

	if repeat == "d 1" && !date.After(now) {
		return now.Format(dateFormat), nil
	}

	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", ErrInvalidFormat
	}

	switch parts[0] {
	case "d":
		return handleDailyRule(now, date, parts)
	case "y":
		return handleYearlyRule(now, date)
	case "w":
		return handleWeeklyRule(now, date, parts)
	case "m":
		return handleMonthlyRule(now, date, parts)
	default:
		return "", ErrInvalidFormat
	}
}

func handleDailyRule(now, date time.Time, parts []string) (string, error) {
	if len(parts) != 2 {
		return "", ErrInvalidFormat
	}

	days, err := strconv.Atoi(parts[1])
	if err != nil || days < 1 || days > 400 {
		return "", ErrExceededMaxDays
	}

	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	next := date
	for {
		next = next.AddDate(0, 0, days)
		if next.After(now) {
			break
		}
	}
	return next.Format(dateFormat), nil
}

func handleYearlyRule(now, date time.Time) (string, error) {
	next := date
	for {
		next = next.AddDate(1, 0, 0)

		if date.Month() == 2 && date.Day() == 29 {

			nextYear := next.Year()
			if !isLeap(nextYear) {
				next = time.Date(nextYear, 3, 1, 0, 0, 0, 0, time.UTC)
			}
		}

		if next.After(now) {
			break
		}
	}
	return next.Format(dateFormat), nil
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func handleWeeklyRule(now, date time.Time, parts []string) (string, error) {
	if len(parts) != 2 {
		return "", ErrInvalidFormat
	}

	days, err := parseDays(parts[1], 1, 7)
	if err != nil {
		return "", ErrInvalidDay
	}

	var next time.Time
	if now.After(date) {
		next = now.AddDate(0, 0, 1)
	} else {
		next = date.AddDate(0, 0, 1)
	}

	for i := 0; i <= 7; i++ {
		weekday := int(next.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if contains(days, weekday) && next.After(now) {
			return next.Format(dateFormat), nil
		}
		next = next.AddDate(0, 0, 1)
	}

	return "", ErrInvalidDay
}

func handleMonthlyRule(now, date time.Time, parts []string) (string, error) {
	if len(parts) < 2 || len(parts) > 3 {
		return "", ErrInvalidFormat
	}

	days, err := parseDays(parts[1], -2, 31)
	if err != nil {
		return "", ErrInvalidDay
	}

	var months []int
	if len(parts) == 3 {
		months, err = parseDays(parts[2], 1, 12)
		if err != nil {
			return "", ErrInvalidMonth
		}
	}

	next := date.AddDate(0, 0, 1)
	for i := 0; i < 366*3; i++ {
		if len(months) > 0 && !contains(months, int(next.Month())) {
			next = time.Date(next.Year(), next.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			continue
		}

		lastDay := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

		for _, d := range days {
			var targetDay int

			if d < 0 {
				targetDay = lastDay + d + 1
			} else {
				targetDay = d
			}

			if targetDay > lastDay || targetDay < 1 {
				continue
			}

			if next.Day() == targetDay && next.After(now) {
				return next.Format(dateFormat), nil
			}
		}

		next = next.AddDate(0, 0, 1)
	}

	return "", fmt.Errorf("%w: не удалось найти подходящую дату", ErrInvalidDay)
}

func parseDays(s string, min, max int) ([]int, error) {
	var result []int
	for _, v := range strings.Split(s, ",") {
		num, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidFormat, v)
		}

		if num < min || num > max {
			return nil, fmt.Errorf("%w: %d не в диапазоне %d-%d",
				ErrInvalidDay, num, min, max)
		}

		result = append(result, num)
	}

	return result, nil
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
