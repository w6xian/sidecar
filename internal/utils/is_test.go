package utils

import (
	"testing"
)

func TestIsNumber(t *testing.T) {
	type args struct {
		str string
	}
	type result struct {
		num bool
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
				num: true,
			},
		},
		{
			name: "123",
			args: args{
				str: "123",
			},
			want: result{
				num: true,
			},
		},
		{
			name: "0.123",
			args: args{
				str: "123",
			},
			want: result{
				num: true,
			},
		},
		{
			name: "a1232",
			args: args{
				str: "a1232",
			},
			want: result{
				num: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNumber(tt.args.str); got != tt.want.num {
				t.Errorf("IsNumber() = %v, want %v", got, tt.want.num)
			}
		})
	}
}
