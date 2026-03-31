package muxhttp

import "reflect"

// 通过json Tag中的omitempty 过滤结构体
func FilterStructWithTag(target any) any {
	ty := reflect.TypeOf(target)
	val := reflect.ValueOf(target)
	if ty.Kind() == reflect.Pointer {
		ty = ty.Elem()
		val = val.Elem()
	}
	// 数组
	if ty.Kind() == reflect.Slice {
		for i := 0; i < val.Len(); i++ {
			FilterStructWithTag(val.Index(i).Interface())
		}
		return target
	}
	num := val.NumField()
	for i := 0; i < num; i++ {
		f := ty.Field(i)
		_, ok := f.Tag.Lookup("ignore")
		if ok {
			val.Field(i).Set(reflect.Zero(f.Type))
		}
	}
	return target
}
