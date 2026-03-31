package errors

import (
	"errors"

	"github.com/w6xian/sidecar/internal/i18n"
)

type IErrorL interface {
	Error() string
}

func NewErrorL(messageId string, message string) *ErrorL {
	return &ErrorL{
		MessageId: messageId,
		Message:   message,
		Default:   message,
	}
}

type ErrorL struct {
	MessageId string `json:"id"`
	Message   string `json:"text"`
	Default   string `json:"-"`
}

func (e *ErrorL) Error() string {
	return e.Message
}

func (e *ErrorL) Ei18n(lang i18n.IParse) error {
	return errors.New(lang.L(e.MessageId, e.Default, i18n.String("error", e.Message)))
}

// 数据库查询错误
func SelectError(tableId string, err error) *ErrorL {
	return &ErrorL{
		MessageId: tableId + "_select_error",
		Message:   err.Error(),
	}
}

// 数据库更新错误
func UpdateError(tableId string, err error) *ErrorL {
	return &ErrorL{
		MessageId: tableId + "_update_error",
		Message:   err.Error(),
	}
}

// 数据库插入错误
func InsertError(tableId string, err error) *ErrorL {
	return &ErrorL{
		MessageId: tableId + "_insert_error",
		Message:   err.Error(),
	}
}

// 数据库删除错误
func DeleteError(tableId string, err error) *ErrorL {
	return &ErrorL{
		MessageId: tableId + "_delete_error",
		Message:   err.Error(),
	}
}
