package array

func Map[T any](arr []T, call func(v T) T) []T {
	var a []T
	for _, v := range arr {
		a = append(a, call(v))
	}
	return a

}

func Slice[T any, R any](arr []T, call func(v T) R) []R {
	var a []R
	for _, v := range arr {
		a = append(a, call(v))
	}
	return a

}

func Keys[K comparable, V any](m map[K]V) []K {
	var keys []K
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
