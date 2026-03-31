package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/i18n"
	"github.com/w6xian/sidecar/internal/server"
	"github.com/w6xian/sidecar/internal/store"
	"github.com/w6xian/sidecar/internal/store/db"
	"github.com/w6xian/sidecar/internal/timex"
	"github.com/w6xian/sidecar/pkg/auth"
	"github.com/w6xian/sidecar/pkg/logger"
	"github.com/w6xian/sidecar/pkg/utils"
)

func daemonCommand(ctx context.Context, appName string) *cobra.Command {
	h := &Deamon{
		Context: ctx,
		Profile: config.NewProfile(),
	}
	h.FlagSet = flag.NewFlagSet(appName, flag.ContinueOnError)
	cmd := &cobra.Command{
		Use:   "sidecar",
		Short: "sidecar is a cache manager",
		RunE:  h.Run,
	}
	return cmd
}

type Deamon struct {
	Context context.Context
	Profile *config.Profile
	FlagSet *flag.FlagSet
	pool    []*exec.Cmd
}

func (h *Deamon) Run(cmd *cobra.Command, args []string) error {

	ctx, cancel := context.WithCancel(h.Context)
	defer cancel()

	pidFile := "app.pid"
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

	optsLog := h.Profile.Logger
	// 初始化日志
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

	logger.L().Info("sidecar server starting",
		zap.String("version", version),
		zap.String("build_time", buildTime),
	)

	// 数据库 这样设计是为了方便以后换数据库引擎
	dbDriver, err := db.NewDBDriver(h.Profile)
	if err != nil {
		cancel()
		pidManager.RemovePID()
		slog.Error("failed to create db driver", "error", err)
		return err
	}
	// 存储
	storeInstance, err := store.New(dbDriver, h.Profile, logger.L())
	if err != nil {
		cancel()
		pidManager.RemovePID()
		slog.Error("failed to create store", "error", err)
		return err
	}

	if err = storeInstance.Migrate(ctx); err != nil {
		cancel()
		pidManager.RemovePID()
		slog.Error("failed to migrate", "error", err)
		return err
	}
	// 读取配置
	httpAddr := h.Profile.Server.HTTPAddr
	if httpAddr == "" {
		httpAddr = ":8443"
	}
	grpcAddr := h.Profile.Server.GRPCAddr
	if grpcAddr == "" {
		grpcAddr = ":9443"
	}
	proxyTimeout := h.Profile.Server.ProxyTimeout
	if proxyTimeout == 0 {
		proxyTimeout = 30 * time.Second
	}

	// 鉴权

	authenticator := auth.NewTokenAuth(logger.L(), storeInstance)

	// IP 白名单
	whitelistIPs := h.Profile.Auth.IPWhitelist
	ipWhitelist := auth.NewIPWhitelist(whitelistIPs)

	// 创建核心组件
	router := server.NewRouter()
	hub := server.NewHub(router)
	go hub.Run()

	handler := server.NewHandler(hub, router, authenticator, ipWhitelist, proxyTimeout)
	gateway := server.NewGateway(server.GatewayConfig{
		HTTPAddr:     httpAddr,
		GRPCAddr:     grpcAddr,
		ProxyTimeout: proxyTimeout,
	}, handler, router, hub)

	// 启动 gRPC 网关
	go func() {
		if err := gateway.StartGRPC(); err != nil {
			logger.L().Fatal("gRPC gateway failed", zap.Error(err))
		}
	}()

	// 启动 HTTP 网关
	go func() {
		if err := gateway.StartHTTP(); err != nil {
			logger.L().Fatal("HTTP gateway failed", zap.Error(err))
		}
	}()

	logger.L().Info("server started",
		zap.String("http_addr", httpAddr),
		zap.String("grpc_addr", grpcAddr),
	)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.L().Info("received signal, shutting down", zap.String("signal", sig.String()))
	return nil
}

func (h *Deamon) initConfig() {
	h.FlagSet.String("config", "config", "path to config file, multiple files separated by commas")
	configFile := h.FlagSet.Lookup("config").Value.String()

	// 文件里读取配置
	cs := strings.Split(configFile, ",")
	var parser config.Unmarshal
	for _, f := range cs {
		parser = config.FromFiles(f, config.YAML)
	}
	parser.Unmarshal(h.Profile)
	parser.Unmarshal(&h.Profile.Server)
	parser.Unmarshal(&h.Profile.Auth)
	parser.Unmarshal(&h.Profile.Logger)

}

func (h *Deamon) initLanguage() {
	// 初始化 i18n
	h.FlagSet.String("lang", "locales", "path to language files")
	langDir := h.FlagSet.Lookup("lang").Value.String()
	err := i18n.Init(langDir)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}
