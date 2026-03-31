package array

// 并集
func Intersect[T comparable](arr1, arr2 []T) []T {
	var inter []T
	mp := make(map[T]bool)
	for _, s := range arr1 {
		if _, ok := mp[s]; !ok {
			mp[s] = true
		}
	}
	for _, s := range arr2 {
		if _, ok := mp[s]; ok {
			inter = append(inter, s)
		}
	}
	return inter
}
