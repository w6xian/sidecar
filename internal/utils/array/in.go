package array

func InArray[T CommonType](e T, arr []T) bool {
	for _, x := range arr {
		if x == e {
			return true
		}
	}
	return false
}
