package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 外部参数
// -srcDir=dist -originalText=http://proxy.51d.ink -newText=http://localhost:8080
func main() {

	// 访问命令行参数的值
	srcDir := ""
	originalText := ""
	newText := ""
	flag.StringVar(&srcDir, "srcDir", srcDir, "输入目录")
	flag.StringVar(&originalText, "originalText", originalText, "原始文本")
	flag.StringVar(&newText, "newText", newText, "新文本")

	// 解析命令行参数
	flag.Parse()

	fmt.Println("srcDir:", srcDir, "originalText:", originalText, "newText:", newText)

	// 检查参数是否为空
	if srcDir == "" || originalText == "" || newText == "" {
		fmt.Println("Error: srcDir, originalText, and newText are required.")
		return
	}

	//

	// 遍历文件夹
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 如果是文件，则进行替换操作
		if !info.IsDir() {
			fmt.Println("Process file:", path)
			// 读取文件内容
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// 替换内容
			data = []byte(strings.ReplaceAll(string(data), originalText, newText))

			// 写回文件
			err = os.WriteFile(path, data, info.Mode())
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Batch replace completed.")
}
