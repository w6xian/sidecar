package song

import (
	"context"

	"github.com/w6xian/sidecar/pkg/media"
)

func Setup(ctx context.Context) error {
	// 启动成功
	player := media.NewNotify()
	go player.Play(media.SETUP)
	return nil
}

func PrinterError(ctx context.Context) error {
	// 打印机错误
	player := media.NewNotify()
	go player.Play(media.PRINTER_ERROR)
	return nil
}
func OrderNew(ctx context.Context) error {
	// 新订单
	player := media.NewNotify()
	go player.Play(media.ORDER_NEW)
	return nil
}
