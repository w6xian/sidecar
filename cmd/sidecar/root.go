package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/i18n"
	"github.com/w6xian/sidecar/internal/sidecar"
	"github.com/w6xian/sidecar/internal/song"
	"github.com/w6xian/sidecar/internal/timex"
	"github.com/w6xian/sidecar/internal/utils/id"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/utils"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

var rootCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "Sidecar is a lightweight process manager and script executor",
	Long:  `Sidecar allows you to manage processes and execute scripts with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		h := &Root{
			Context: context.Background(),
			Profile: config.NewProfile(),
		}
		h.FlagSet = flag.NewFlagSet("sidecar", flag.ContinueOnError)
		h.Run(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().String("path", "", "working directory for service mode")
	rootCmd.PersistentFlags().String("token", "", "service token")
	_ = rootCmd.PersistentFlags().MarkHidden("token")
}

type Root struct {
	Context context.Context
	Profile *config.Profile
	FlagSet *flag.FlagSet
	pool    []*exec.Cmd
}

func (h *Root) Run(cmd *cobra.Command, args []string) error {
	runPath, _ := cmd.PersistentFlags().GetString("path")
	if runPath != "" {
		if err := os.Chdir(runPath); err != nil {
			return err
		}
	}
	token, _ := cmd.PersistentFlags().GetString("token")
	if token == "" {
		token = id.ShortID()
	}

	ctx, cancel := context.WithCancel(h.Context)
	defer cancel()

	pidFile := "sidecar.pid"
	pidManager := utils.NewPIDManagerFromConfig(pidFile)
	err := pidManager.WritePID()
	if err != nil {
		fmt.Println(err.Error())
	}
	defer pidManager.RemovePID()

	// 读取配置文件
	h.initConfig()
	// 需要先initProfile
	// i18n
	h.initLanguage()
	// 初始化时间
	timex.InitLocation(h.Profile.Timezone)

	// 初始化日志
	logLevel := h.Profile.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}
	optsLog := h.Profile.Logger
	hook := &lumberjack.Logger{
		Filename:   optsLog.FilePath + optsLog.Filename,
		MaxSize:    optsLog.MaxSize, // megabytes
		MaxBackups: optsLog.MaxBackups,
		MaxAge:     optsLog.MaxAge,   //days
		Compress:   optsLog.Compress, // disabled by default
		LocalTime:  optsLog.LocalTime,
	}
	defer hook.Close()

	logger.Init(optsLog, hook)
	defer logger.Sync()

	logger.L().Info("sidecar client starting",
		zap.String("version", version),
		zap.String("build_time", buildTime),
	)

	if h.Profile.Sidecar.ServerAddr == "" {
		logger.L().Fatal("server_addr is required")
	}

	if len(h.Profile.Services) == 0 {
		logger.L().Fatal("at least one service must be configured")
	}

	// 创建并启动客户端
	client := sidecar.NewClient(h.Profile)

	// 后台运行
	go func() {
		if err := client.Start(); err != nil {
			logger.L().Fatal("client start failed", zap.Error(err))
		}
	}()

	logger.L().Info("sidecar client started",
		zap.String("server", h.Profile.Sidecar.ServerAddr),
		zap.Int("services", len(h.Profile.Services)),
	)
	// 启动成功声音
	song.Setup(ctx)
	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.L().Info("received signal, shutting down",
		zap.String("signal", sig.String()),
	)
	client.Stop()
	return nil
}

func (h *Root) initConfig() {
	h.FlagSet.String("config", "config", "path to config file, multiple files separated by commas")
	configFile := h.FlagSet.Lookup("config").Value.String()

	// 文件里读取配置
	cs := strings.Split(configFile, ",")
	var parser config.Unmarshal
	for _, f := range cs {
		parser = config.FromFiles(f, config.YAML)
	}
	parser.Unmarshal(h.Profile)
	parser.Unmarshal(&h.Profile.Sidecar)
	parser.Unmarshal(&h.Profile.Services)
	parser.Unmarshal(&h.Profile.Logger)
}

func (h *Root) initLanguage() {
	// 初始化 i18n
	h.FlagSet.String("lang", "locales", "path to language files")
	langDir := h.FlagSet.Lookup("lang").Value.String()
	err := i18n.Init(langDir)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}
