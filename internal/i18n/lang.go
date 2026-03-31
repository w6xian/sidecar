package i18n

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle          *i18n.Bundle
	defaultLanguage = language.Chinese
)

// 初始化 i18n bundle
func Init(dir string) error {
	bundle = i18n.NewBundle(defaultLanguage)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	err := reloadTranslations(dir)
	if err != nil {
		return err
	}
	go WatchTranslationFiles(dir)
	return nil
}

func reloadTranslations(dir string) error {
	// 加载所有翻译文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			filePath := path.Join(dir, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			_, err = bundle.ParseMessageFileBytes(data, filePath)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func WatchTranslationFiles(dir string) {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	watcher.Add(dir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if strings.HasSuffix(event.Name, ".toml") {
					// 重新加载翻译文件
					// log.Println("Reloading translation file:", event.Name)
					err := reloadTranslations(dir)
					if err != nil {
						log.Println("Error reloading translation file:", err)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Translation watcher error:", err)
		}
	}
}

// 获取针对特定语言的本地化器
func NewLocalizer(lang string) *i18n.Localizer {
	// 如果传入空字符串，使用默认语言
	if lang == "" {
		return i18n.NewLocalizer(bundle, defaultLanguage.String())
	}

	// 创建包含回退语言的本地化器
	return i18n.NewLocalizer(bundle, lang, defaultLanguage.String())
}

// T 是简化的翻译函数，用于没有参数的简单字符串
func T(lang, messageID string, def string) string {
	localizer := NewLocalizer(lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})

	if err != nil {
		// 记录错误但返回 messageID 作为回退
		return def
	}

	return msg
}

type D map[string]any

// TWithData 用于包含模板数据的翻译
func TWithData(lang, messageID string, def string, templateData D) string {
	localizer := NewLocalizer(lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})

	if err != nil {
		return def
	}

	return msg
}

// TPlural 用于处理复数形式
func TPlural(lang, messageID string, count any, def string, templateData D) string {
	if templateData == nil {
		templateData = make(D)
	}

	// 确保 templateData 中包含 Count
	templateData["Count"] = count

	localizer := NewLocalizer(lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  count,
		TemplateData: templateData,
	})

	if err != nil {
		return def
	}

	return msg
}
