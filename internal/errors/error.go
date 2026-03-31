package errors

import (
	"errors"
	"fmt"

	"github.com/w6xian/sidecar/internal/i18n"
	"github.com/w6xian/sidecar/internal/message"
)

func New(msg string) error {
	return errors.New(msg)
}

func NewL(lang i18n.IParse, msgId string, def string, args ...string) error {
	fs := []i18n.Field{}
	for i, arg := range args {
		fs = append(fs, i18n.String(fmt.Sprintf("%d", i), arg))
	}
	return errors.New(lang.L(msgId, def, fs...))
}

// 完成：手机号已存在
func MobileExists(lang i18n.IParse, mobile string) error {
	return errors.New(lang.L(message.MOBILE_EXISTS, "手机号已存在", i18n.String("mobile", mobile)))
}
