package main

import (
	"context"
	"fmt"
	"go_web_scaffolding/dao/mysql"
	"go_web_scaffolding/dao/redis"
	"go_web_scaffolding/routes"
	"go_web_scaffolding/settings"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var logger *zap.Logger
var sugarLogger *zap.SugaredLogger

func main() {
	// 1. 加载配置
	if err := settings.Init(); err != nil {
		fmt.Printf("init setting failed error:%v\n", err)
		return
	}

	// 2. 初始化日志
	if err := logger.Init(settings.Conf.LogConfig); err != nil {
		fmt.Printf("init setting failed error:%v\n", err)
		return
	}
	// 延迟注册一下，把缓冲区的文件追加到日志文件中
	defer zap.L().Sync()
	// zap.ReplaceGlobals(lg)后 通过zap.L()调用
	zap.L().Debug("logger init success...")

	// 3. 初始化MySQL连接
	if err := mysql.Init(settings.Conf.MySQLConfig); err != nil {
		fmt.Printf("init setting failed error:%v\n", err)
		return
	}
	defer mysql.Close()

	// 4. 初始化Redis连接
	if err := redis.Init(); err != nil {
		fmt.Printf("init setting failed error:%v\n", err)
		return
	}
	defer redis.Close()
	// 5. 注册路由
	r := routes.Setup()
	// 6. 启动服务（优雅关机）
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("app.port")),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号量来优雅关闭服务器，为关闭服务器设置一个5秒的超时
	quit := make(chan os.Signal, 1) // 创建一个接收信号的通道
	// kill 默认会发送syscall.SIGTERM信号
	// kill -2 发送 syscall.SIGINT 信号，我们常用的Ctrl+C就是触发系统的SIGINT信号
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	// signal.Notify把收到的 syscall.SIGINT或syscall.SIGTERM 信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会阻塞
	<-quit                                               // 阻塞在此处，当接收到上述两种信号时才会往下执行
	zap.L().Info("Shutdown Server ...")
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 5秒内优雅关闭服务（将未处理玩的请求处理完再关闭服务），超过5秒就超时退出
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown", zap.Error(err))
	}

}
