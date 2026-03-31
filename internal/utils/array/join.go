package array

import (
	"fmt"
	"strings"
)

const maxInt = int(^uint(0) >> 1)

func Join[T CommonType](elems []T, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%v", elems[0])
	}

	var n int
	if len(sep) > 0 {
		if len(sep) >= maxInt/(len(elems)-1) {
			panic("strings: Join output length overflow")
		}
		n += len(sep) * (len(elems) - 1)
	}
	for _, elem := range elems {
		elemStr := fmt.Sprintf("%v", elem)
		if len(elemStr) > maxInt-n {
			panic("strings: Join output length overflow")
		}
		n += len(elemStr)
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(fmt.Sprintf("%v", elems[0]))
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(fmt.Sprintf("%v", s))
	}
	return b.String()
}
