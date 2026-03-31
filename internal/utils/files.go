package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func Md5ByLocalFile(filePath string) (string, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// 创建一个md5哈希对象
	hasher := md5.New()

	// 将文件内容读入哈希对象
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// 计算MD5值并转换为十六进制字符串
	md5Sum := hasher.Sum(nil)
	return hex.EncodeToString(md5Sum), nil
}

func GetFileName(path string) string {
	// 从路径中提取文件名
	fileName := path
	if i := strings.LastIndex(fileName, "/"); i != -1 {
		fileName = fileName[i+1:]
	}
	return fileName
}
