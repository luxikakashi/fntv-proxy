package main

import (
	"fntv-proxy/internal/config"
	"fntv-proxy/internal/emby"
	"fntv-proxy/internal/proxy"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := os.Getenv("CONFIG")
	if err := config.Load(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	config.Watch(func() {
		log.Println("🔄 配置已更新")
	})

	fntvServer, err := proxy.NewServer(config.Global)
	if err != nil {
		log.Fatalf("创建飞牛代理服务器失败: %v", err)
	}

	var embyServer *emby.Server
	if config.Global.Emby.IsEnabled() {
		embyServer, err = emby.NewServer(config.Global)
		if err != nil {
			log.Fatalf("创建 Emby 代理服务器失败: %v", err)
		}
	}

	log.Printf("🚀 FNTV Proxy 启动")
	log.Printf("   飞牛监听: %s", config.Global.GetListenAddr())
	log.Printf("   飞牛目标: %s", config.Global.GetTargetAddr())
	if config.Global.Emby.IsEnabled() {
		log.Printf("   Emby监听: %s", config.Global.Emby.GetListenAddr())
		log.Printf("   Emby目标: %s", config.Global.Emby.GetTargetAddr())
	}
	log.Printf("   日志级别: %s", config.Global.GetLogLevel())
	log.Printf("   缓存TTL: %v", config.Global.GetCacheTTL())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("🛑 正在关闭...")
		if embyServer != nil {
			embyServer.Stop()
		}
		fntvServer.Stop()
	}()

	if embyServer != nil {
		go func() {
			if err := embyServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Emby 代理启动失败: %v", err)
			}
		}()
	}

	if err := fntvServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("飞牛代理启动失败: %v", err)
	}
}
