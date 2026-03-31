package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type JsonValue map[string]*json.RawMessage

func (j JsonValue) String(col string) string {
	var str string
	if j[col] != nil {
		json.Unmarshal(*j[col], &str)
	}
	return str
}

func (j JsonValue) Int64(col string) int64 {
	var i int64
	if j[col] != nil {
		err := json.Unmarshal(*j[col], &i)
		if err != nil {
			str := j.String(col)
			i, _ = strconv.ParseInt(str, 10, 64)
		}
	}
	return i
}

func (j JsonValue) Ints64(col string) []int64 {
	var i []int64
	if j[col] != nil {
		json.Unmarshal(*j[col], &i)

	}
	return i
}

func (j JsonValue) Uint64(col string) uint64 {
	var i uint64
	if j[col] != nil {
		err := json.Unmarshal(*j[col], &i)
		if err != nil {
			str := j.String(col)
			i, _ = strconv.ParseUint(str, 10, 64)
		}
	}
	return i
}

func (j JsonValue) Uints64(col string) []uint64 {
	var i []uint64
	if j[col] != nil {
		json.Unmarshal(*j[col], &i)

	}
	return i
}

func (j JsonValue) Ints(col string) []int {
	var i []int
	if j[col] != nil {
		json.Unmarshal(*j[col], &i)

	}
	return i
}
func (j JsonValue) Bool(col string) bool {
	var b bool
	if j[col] != nil {
		json.Unmarshal(*j[col], &b)
	}
	return b
}

func (j JsonValue) Int(col string) int {
	return int(j.Int64(col))
}

func (j JsonValue) MapSI(col string) map[string]interface{} {
	var m = map[string]interface{}{}
	if j[col] != nil {
		json.Unmarshal(*j[col], &m)
	}
	return m
}
func (j JsonValue) MapSS(col string) map[string]string {
	var m = map[string]string{}
	if j[col] != nil {
		json.Unmarshal(*j[col], &m)
	}
	fmt.Printf("%v%v", m, j[col])
	return m
}

// func MapToStruct(s any, v any) error {
// 	if s == nil {
// 		return nil
// 	}
// 	b, err := json.Marshal(s)
// 	if err != nil {
// 		return err
// 	}
// 	return json.Unmarshal(b, v)
// }

func Serialize(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte{}
	}
	return b
}

func Deserialize(b []byte, v any) error {
	err := json.Unmarshal(b, v)
	if err != nil {
		return err
	}
	return nil
}
