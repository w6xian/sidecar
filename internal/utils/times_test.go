package utils

import (
	"testing"
	"time"
)

// TestUnixTimeFromStr 测试 UnixTimeFromStr 函数
func TestUnixTimeFromStr(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"2026-01-14 18:56:13", 1768388173},
	}

	for _, test := range tests {
		location, _ := time.LoadLocation("Asia/Shanghai")
		result := TimeL(location, test.expected)
		if result.Format("2006-01-02 15:04:05") != test.input {
			t.Errorf("UnixTimeL(%d) = %s; want %s", test.expected, result.Format("2006-01-02 15:04:05"), test.input)
		}
		if UnixTimeFromStr(test.input, "Asia/Shanghai") != test.expected {
			t.Errorf("UnixTimeFromStr(%s) = %d; want %d", test.input, UnixTimeFromStr(test.input, "Asia/Shanghai"), test.expected)
		}
	}
}
