package logger

import (
	"os"
	"path/filepath"

	"github.com/w6xian/sidecar/internal/pathx"
)

func Panic(err error) {
	logStr(err.Error())
}

func logStr(content string) error {
	return tLog([]byte(content + "\n"))
}

func tLog(content []byte) error {
	f, err := os.OpenFile(filepath.Join(pathx.GetCurrentAbPath(), "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
