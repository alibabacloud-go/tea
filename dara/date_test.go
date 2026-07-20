package dara

import (
	"testing"
	"time"
)

func TestConstructWithNow(t *testing.T) {
	date := &Date{date: time.Now()}
	currentTime := time.Now()
	if currentTime.Format("2006-01-02 15:04:05") != date.Format("yyyy-MM-dd hh:mm:ss") {
		t.Errorf("Expected %v, got %v", currentTime.Format("2006-01-02 15:04:05"), date.Format("yyyy-MM-dd hh:mm:ss"))
	}
}

func TestConstructWithDateTimeString(t *testing.T) {
	datetime := "2023-03-01T12:00:00Z" // Use RFC3339 format
	date, err := NewDate(datetime)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if datetime != date.Format("yyyy-MM-ddThh:mm:ssZ") {
		t.Errorf("Expected %v, got %v", datetime, date.Format("yyyy-MM-ddThh:mm:ssZ"))
	}
}

func TestConstructWithWrongType(t *testing.T) {
	_, err := NewDate("20230301 12:00:00 +0000 UTC")
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}

func TestConstructWithUTC(t *testing.T) {
	datetimeUTC := "2023-03-01T12:00:00Z"
	dateWithUTC, err := NewDate(datetimeUTC)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	referenceDateTime, _ := time.Parse(time.RFC3339, datetimeUTC)
	if referenceDateTime.Unix() != dateWithUTC.Unix() {
		t.Errorf("Expected %v, got %v", referenceDateTime.Unix(), dateWithUTC.Unix())
	}

	formattedDateTime := dateWithUTC.UTC()
	expectedFormattedDateTime := referenceDateTime.UTC().Format("2006-01-02 15:04:05.000000000 -0700 MST")
	if formattedDateTime != expectedFormattedDateTime {
		t.Errorf("Expected %v, got %v", expectedFormattedDateTime, formattedDateTime)
	}
}

func TestFormat(t *testing.T) {
	datetime := "2023-03-01T12:00:00Z"
	date, _ := NewDate(datetime)
	expected := "2023-03-01 12:00 PM"
	if result := date.Format("yyyy-MM-dd hh:mm a"); result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestUTC(t *testing.T) {
	datetime := "2023-03-01T12:00:00+08:00"
	date, _ := NewDate(datetime)
	expected := "2023-03-01 04:00:00.000000000 +0000 UTC"
	if result := date.UTC(); result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestUnix(t *testing.T) {
	datetime := "1970-01-01T00:00:00Z"
	date, _ := NewDate(datetime)
	if result := date.Unix(); result != 0 {
		t.Errorf("Expected 0, got %v", result)
	}

	datetime = "2023-12-31T08:00:00+08:00"
	date, _ = NewDate(datetime)
	if result := date.Unix(); result != 1703980800 {
		t.Errorf("Expected 1703980800, got %v", result)
	}
}

func TestAddSub(t *testing.T) {
	datetime := "2023-03-01T12:00:00Z"
	date, _ := NewDate(datetime)
	date = date.Add(1, "day")
	expectedDate := time.Date(2023, 3, 2, 12, 0, 0, 0, time.UTC)
	if date.date != expectedDate {
		t.Errorf("Expected %v, got %v", expectedDate, date.date)
	}
	date = date.Sub(1, "day") // Subtract 1 day
	expectedDate = time.Date(2023, 3, 1, 12, 0, 0, 0, time.UTC)
	if date.date != expectedDate {
		t.Errorf("Expected %v, got %v", expectedDate, date.date)
	}
}

func TestDiff(t *testing.T) {
	datetime1 := "2023-03-01T12:00:00Z"
	datetime2 := "2023-04-01T12:00:00Z"
	date1, _ := NewDate(datetime1)
	date2, _ := NewDate(datetime2)
	diffInSeconds := date1.Diff("seconds", date2)
	if diffInSeconds != -31*24*60*60 {
		t.Errorf("Expected %v, got %v", -31*24*60*60, diffInSeconds)
	}

	if date1.Diff("minutes", date2) != -31*24*60 {
		t.Errorf("unexpected minutes diff: %v", date1.Diff("minutes", date2))
	}
	if date1.Diff("hours", date2) != -31*24 {
		t.Errorf("unexpected hours diff: %v", date1.Diff("hours", date2))
	}
	if date1.Diff("days", date2) != -31 {
		t.Errorf("unexpected days diff: %v", date1.Diff("days", date2))
	}
	if date1.Diff("weeks", date2) != -4 {
		t.Errorf("unexpected weeks diff: %v", date1.Diff("weeks", date2))
	}
	// months uses (diffDate - receiver) calendar months
	if date1.Diff("months", date2) != 1 {
		t.Errorf("unexpected months diff: %v", date1.Diff("months", date2))
	}
	if date1.Diff("years", date2) != 0 {
		t.Errorf("unexpected years diff: %v", date1.Diff("years", date2))
	}
	if date1.Diff("invalid", date2) != 0 {
		t.Errorf("expected 0 for invalid unit")
	}
}

func TestAddSubUnits(t *testing.T) {
	datetime := "2023-03-01T12:00:00Z"
	date, _ := NewDate(datetime)

	units := []struct {
		unit     string
		amount   int
		addExpect time.Time
		subExpect time.Time
	}{
		{"second", 1, time.Date(2023, 3, 1, 12, 0, 1, 0, time.UTC), time.Date(2023, 3, 1, 11, 59, 59, 0, time.UTC)},
		{"minute", 1, time.Date(2023, 3, 1, 12, 1, 0, 0, time.UTC), time.Date(2023, 3, 1, 11, 59, 0, 0, time.UTC)},
		{"hour", 1, time.Date(2023, 3, 1, 13, 0, 0, 0, time.UTC), time.Date(2023, 3, 1, 11, 0, 0, 0, time.UTC)},
		{"week", 1, time.Date(2023, 3, 8, 12, 0, 0, 0, time.UTC), time.Date(2023, 2, 22, 12, 0, 0, 0, time.UTC)},
		{"month", 1, time.Date(2023, 4, 1, 12, 0, 0, 0, time.UTC), time.Date(2023, 2, 1, 12, 0, 0, 0, time.UTC)},
		{"year", 1, time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC), time.Date(2022, 3, 1, 12, 0, 0, 0, time.UTC)},
	}

	for _, tc := range units {
		base, _ := NewDate(datetime)
		result := base.Add(tc.amount, tc.unit)
		if result == nil || !result.date.Equal(tc.addExpect) {
			t.Errorf("Add(%d, %s): expected %v, got %v", tc.amount, tc.unit, tc.addExpect, result)
		}

		base, _ = NewDate(datetime)
		result = base.Sub(tc.amount, tc.unit)
		if result == nil || !result.date.Equal(tc.subExpect) {
			t.Errorf("Sub(%d, %s): expected %v, got %v", tc.amount, tc.unit, tc.subExpect, result)
		}
	}

	if date.Add(1, "invalid") != nil {
		t.Error("expected nil for invalid Add unit")
	}
	if date.Sub(1, "invalid") != nil {
		t.Error("expected nil for invalid Sub unit")
	}
}

func TestHourMinuteSecond(t *testing.T) {
	datetime := "2023-03-01T12:34:56Z"
	date, _ := NewDate(datetime)
	if result := date.Hour(); result != 12 {
		t.Errorf("Expected 12, got %d", result)
	}
	if result := date.Minute(); result != 34 {
		t.Errorf("Expected 34, got %d", result)
	}
	if result := date.Second(); result != 56 {
		t.Errorf("Expected 56, got %d", result)
	}
}

func TestMonthYearDay(t *testing.T) {
	datetime := "2023-03-01T12:00:00Z"
	date, _ := NewDate(datetime)
	if result := date.Month(); result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
	if result := date.Year(); result != 2023 {
		t.Errorf("Expected 2023, got %d", result)
	}
	if result := date.DayOfMonth(); result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestDayOfWeekWeekOfYear(t *testing.T) {
	datetime := "2023-03-01 00:00:00"
	date, _ := NewDate(datetime)
	if result := date.DayOfWeek(); result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
	if result := date.WeekOfYear(); result != 9 {
		t.Errorf("Expected 9, got %d", result)
	}

	datetime1 := "2023-12-31T12:00:00Z"
	date1, _ := NewDate(datetime1)
	if result := date1.DayOfMonth(); result != 31 {
		t.Errorf("Expected 31, got %d", result)
	}
	if result := date1.DayOfWeek(); result != 7 {
		t.Errorf("Expected 7, got %d", result)
	}
	if result := date1.WeekOfYear(); result != 52 {
		t.Errorf("Expected 52, got %d", result)
	}
}
