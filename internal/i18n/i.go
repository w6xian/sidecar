package i18n

type IParse interface {
	// String 获取翻译后的字符串
	// {{.Name}} 替换为参数中的 Name 字段,Field 会转成Map[string]interface{}
	L(key string, def string, args ...Field) string
}
