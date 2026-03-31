package utils

import (
	"testing"
)

func TestPinyin(t *testing.T) {

	tests := []struct {
		name string
		args string
		want string
	}{

		{
			name: "永",
			args: "永",
			want: "y",
		},
		{
			name: "嗯",
			args: "嗯",
			want: "n",
		},
		{
			name: "爱",
			args: "爱",
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Pinyin(tt.args)
			if got != tt.want {
				t.Errorf("Pinyin() = %v, want %v", got, tt.want)
			} else {
				t.Logf("Pinyin() = %v, want %v", got, tt.want)
			}
		})
	}

}
