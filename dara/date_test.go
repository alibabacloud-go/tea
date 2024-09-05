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
