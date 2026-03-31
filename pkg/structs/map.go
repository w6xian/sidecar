package structs

import "github.com/w6xian/sidecar/internal/utils"

type M map[string]string

func (k M) GetString(key string, defaultValue ...string) string {
	if k == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	if val, ok := k[key]; ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (k M) GetInt64(key string, defaultValue ...int64) int64 {
	if k == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	if val, ok := k[key]; ok {
		return utils.GetInt64(val)
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (k M) GetBool(key string, defaultValue ...bool) bool {
	if k == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	if val, ok := k[key]; ok {
		return utils.GetInt64(val) != 0
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

func (k M) GetFloat64(key string, defaultValue ...float64) float64 {
	if k == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	if val, ok := k[key]; ok {
		return utils.GetFloat64(val)
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}
