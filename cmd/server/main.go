package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ops-platform/internal/server"
)

func main() {
	cfg := server.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := server.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer pool.Close()

	if err := server.Migrate(ctx, pool); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	auth := server.NewAuth(pool, cfg.JWTSecret)
	if err := auth.SeedAdmin(ctx, cfg.AdminUser, cfg.AdminPassword); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}

	hub := server.NewHub()
	remote := server.NewRemoteManager(pool, hub)
	srv := server.New(cfg, pool, auth, hub, remote)
	go srv.OfflineLoop(ctx)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Routes(srv),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("服务端启动: http://%s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务端退出: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Println("正在关闭...")
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = httpSrv.Shutdown(shutdownCtx)
}
