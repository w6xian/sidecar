package def

func GetString(i string, defualt string) string {
	if i == "" {
		return defualt
	}
	return i
}

// 泛型函数
func Get[T comparable](i T, defualt T) T {
	if i == defualt {
		return defualt
	}
	return i
}
