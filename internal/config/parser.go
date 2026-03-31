package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

/**
* 如果是目录，需要一个文件一个文件的绑定
 */

type Unmarshal interface {
	Unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error
}
type Parser struct {
}

func (p *Parser) Unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.Unmarshal(rawVal, opts...)
}

func FromFiles(f string, t string) Unmarshal {
	env := GetMode()
	parser := &Parser{}
	fi, err := os.Stat(f)
	if err != nil {
		d := fmt.Sprintf("%s.%s.%s", f, env, t)
		if _, err := os.Stat(d); err == nil {
			readAllConfig(d, t)
			return parser
		} else {
			panic(err)
		}
	}
	if fi.IsDir() {
		f = f + "/" + env + "/"
	}
	readAllConfig(f, t)
	return parser
}

func GetMode() string {
	env := os.Getenv("RUN_MODE")
	if env == "" {
		env = "dev"
	}
	return env
}

func readAllConfig(fPath string, configType string) {
	f, err := os.Stat(fPath)
	if err != nil {
		panic(err)
	}
	viper.SetConfigType(configType)
	if f.IsDir() {
		viper.AddConfigPath(fPath)
		filepath.Walk(fPath, func(f string, info os.FileInfo, err error) error {
			if err != nil {
				panic(err)
			}
			if !info.IsDir() {
				ff := strings.Split(f, string(os.PathSeparator))
				filename := ff[len(ff)-1]
				filesuffix := path.Ext(filename)
				if filesuffix[1:] == configType {
					readConfigFile(filename)
				}
			}
			return nil
		})
	} else {
		filename := fPath
		filenameall := path.Base(filename)
		dir := getCurrentAbPath(filename)
		filesuffix := path.Ext(filename)
		if filesuffix[1:] == configType {
			viper.AddConfigPath(dir)
			readConfigFile(filenameall)
		}

	}
}

func readConfigFile(filename string) {
	ff := strings.Split(filename, ".")
	ff = ff[0 : len(ff)-1]
	filename = strings.Join(ff, ".")
	viper.SetConfigName(filename)
	err := viper.MergeInConfig()
	if err != nil {
		panic(err)
	}
}

func getCurrentAbPath(f string) string {
	var aPath []string
	if strings.Contains(f, "\\") {
		aPath = strings.Split(f, "\\")
	} else {
		aPath = strings.Split(f, "/")
	}
	dir := strings.Join(aPath[0:len(aPath)-1], string(os.PathSeparator))
	if dir == "" {
		dir = "./"
	}
	return dir
}
