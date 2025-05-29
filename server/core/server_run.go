package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type server interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(context.Context) error
}

// initServer 启动服务并实现优雅关闭，支持TLS
func initServer(address string, router *gin.Engine, readTimeout, writeTimeout time.Duration) {
	// TLS配置
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// 创建服务
	srv := &http.Server{
		Addr:           address,
		Handler:        router,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,
	}

	// 在goroutine中启动服务
	go func() {
		var err error

		// 检查是否启用TLS
		if global.GVA_CONFIG.NfcRelay.Security.EnableTLS {
			certFile := global.GVA_CONFIG.NfcRelay.Security.CertFile
			keyFile := global.GVA_CONFIG.NfcRelay.Security.KeyFile

			if certFile == "" || keyFile == "" {
				zap.L().Fatal("TLS已启用但未配置证书文件路径")
			}

			zap.L().Info("启动HTTPS服务器",
				zap.String("address", address),
				zap.String("certFile", certFile))
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			zap.L().Warn("⚠️  服务器正在以HTTP模式运行，建议启用TLS以确保安全")
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("listen: %s\n", err)
			zap.L().Error("server启动失败", zap.Error(err))
			os.Exit(1)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	// kill (无参数) 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT
	// kill -9 发送 syscall.SIGKILL，但是无法被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("关闭WEB服务...")

	// 设置5秒的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("WEB服务关闭异常", zap.Error(err))
	}

	zap.L().Info("WEB服务已关闭")
}
