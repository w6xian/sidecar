package media

import (
	"embed"
	_ "embed"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type Notify struct {
	FileName string
	media    beep.StreamSeekCloser
	done     chan bool
	loading  chan bool
	Lists    []string
}

//go:embed notice/*.mp3
var noticeFS embed.FS

const (
	SETUP         = "notice/setup.mp3"         // 程序启动
	CONNECTED     = "notice/connected.mp3"     // sock链接成功
	CONNECT_FAIL  = "notice/connect_fail.mp3"  // sock连接失败
	ORDER_NEW     = "notice/order_new.mp3"     // 新订单
	ERROR         = "notice/error.mp3"         // 失败
	TODO          = "notice/new_todo.mp3"      // 新工单
	MESSAGE       = "notice/new_msg.mp3"       // 新消息
	PRINT         = "notice/new_print.mp3"     // 新打印任务
	ORDER         = "notice/new_order.mp3"     // 新订单（短）
	LOGIN         = "notice/login.mp3"         // 登录
	SETTING       = "notice/setting.mp3"       // 登录
	PRINTER_ERROR = "notice/printer_error.mp3" // 打印机错误
)

var notice *Notify
var once sync.Once

func NewNotify() *Notify {
	once.Do(func() {
		notice = &Notify{
			FileName: "./notice/notice.mp3",
			done:     make(chan bool, 1),
			loading:  make(chan bool, 1),
		}
	})
	return notice
}

func (m *Notify) Play(tar ...string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	// 播放列表
	if len(m.Lists) > 0 && len(tar) > 0 {
		lst := len(m.Lists) - 1
		n := m.Lists[lst]
		if tar[0] == n {
			return
		}
	}
	m.Lists = append(m.Lists, tar...)

	m.loading <- true
	defer func() {
		<-m.loading
	}()
	// 用于数据同步，当播放完毕的时候，回调函数中通过chan通知主goroutine
	source := m.FileName
	if len(tar) > 0 {
		source = tar[0]
	}

	if m.media != nil {
		speaker.Close()
		speaker.Clear()
		m.done <- true
	}

	var mp3media beep.StreamSeekCloser
	var format beep.Format
	// 检查embed.FS是否有文件存在,如果存在用存在的文件，否则查看本地目录中是否有文件存在,如果不存在用默认文件
	// 读出embed.FS中文件给mp3.Decode
	mp3File, err := noticeFS.Open(source)
	if err == nil {
		defer mp3File.Close()
		mp3media, format, err = mp3.Decode(mp3File)
		if err != nil {
			log.Fatal(err)
			return
		}
	} else {
		// 1. 打开mp3文件
		audioFile, err := os.Open(source)
		fmt.Println(audioFile)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer audioFile.Close()
		// 对文件进行解码
		mp3media, format, err = mp3.Decode(audioFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	m.media = mp3media
	defer func() {
		m.media = nil
	}()
	// SampleRate is the number of samples per second. 采样率
	// 通过采样率来更改播放速度, 只能通过整形倍数变换粒度太粗
	sr := format.SampleRate * 1
	if err := speaker.Init(sr, sr.N(time.Second/10)); err == nil {

		// 重新采样 对然对电脑有信心还是先按照good进行测试，以免渲染不了
		// 从1开始升高音质，确保Mp3的本身是高音质的不然听着差别不大
		//var quality int = 6
		//resample := beep.Resample(quality, format.SampleRate, sr, audioStreamer)

		// 这里播放音乐
		speaker.Play(beep.Seq(m.media, beep.Callback(func() {
			// 播放完成调用回调函数
			m.done <- true
		})))

		// 增加控制信息
		// 增加控制信息
		<-m.done
		// 播放完成后，清空列表
		idx := slices.Index(m.Lists, tar[0])
		if idx != -1 {
			m.Lists = append(m.Lists[:idx], m.Lists[idx+1:]...)
		}
		speaker.Close()
		m.media.Close()
	}

}
