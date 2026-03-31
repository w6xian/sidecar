package utils

import (
	"testing"
)

func TestToNumber(t *testing.T) {
	type args struct {
		str string
	}
	type result struct {
		num float64
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "123.456",
			args: args{
				str: "123.456",
			},
			want: result{
				num: 123.456,
			},
		},
		{
			name: "123",
			args: args{
				str: "123",
			},
			want: result{
				num: 123,
			},
		},
		{
			name: "0.123",
			args: args{
				str: "0.123",
			},
			want: result{
				num: 0.123,
			},
		},
		{
			name: "0",
			args: args{
				str: "0",
			},
			want: result{
				num: 0,
			},
		},
		{
			name: "123.456.789",
			args: args{
				str: "123.456.789",
			},
			want: result{
				num: 123.456,
			},
		},
		{
			name: "123.456.789.012",
			args: args{
				str: "123.456.789.012",
			},
			want: result{
				num: 123.456,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToNumber(tt.args.str); got != tt.want.num {
				t.Errorf("ToNumber() = %v, want %v", got, tt.want.num)
			}
		})
	}

}

func TestMaxString(t *testing.T) {

	tests := []struct {
		name string
		args string
		max  int
		want string
	}{
		{
			name: "刘贤",
			max:  1,
			args: "刘贤",
			want: "刘",
		},
		{
			name: "混合字母与空格",
			max:  5,
			args: "SST 刘贤",
			want: "SST 刘",
		},
		{
			name: "混合字母与空格后向前截取",
			max:  -5,
			args: "SST 刘贤",
			want: "ST 刘贤",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxString(tt.args, tt.max); got != tt.want {
				t.Errorf("MaxString() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestSqlParse(t *testing.T) {

	tests := []struct {
		name string
		sql  string
		args []any
		want string
	}{
		{
			name: "字符串替换",
			sql:  "select * from table where id = ? and name = ?",
			args: []any{1, "张三"},
			want: "select * from table where id = 1 and name = '张三'",
		},
		{
			name: "混合字母与空格",
			sql:  "select * from table where name = ?",
			args: []any{"SST 刘贤"},
			want: "select * from table where name = 'SST 刘贤'",
		},
		{
			name: "混合字母与空格后向前截取",
			sql:  "select * from table where name = ?",
			args: []any{"SST 刘贤"},
			want: "select * from table where name = 'SST 刘贤'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SqlParse(tt.sql, tt.args...); got != tt.want {
				t.Errorf("SqlFilter() = %v, want %v", got, tt.want)
			}
		})
	}

}
