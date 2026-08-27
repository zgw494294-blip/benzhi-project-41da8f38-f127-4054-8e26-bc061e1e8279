package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	store "benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/store/sqlite"
	webui "benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseConfig()
	if err != nil {
		return err
	}
	repository, err := store.Open(cfg.database)
	if err != nil {
		return err
	}
	defer repository.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := repository.CheckIntegrity(ctx); err != nil {
		cancel()
		return fmt.Errorf("启动存储校验: %w", err)
	}
	cancel()
	service := application.NewService(repository)
	handler := webui.New(service)
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()
	slog.Info("无障碍字幕审校台已启动", "addr", listener.Addr().String())
	if cfg.selfcheck {
		return runCheckAndStop(server, listener, serveErr, cfg.selfcheckTimeout)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		slog.Info("收到退出信号", "signal", sig.String())
	case err := <-serveErr:
		if err != nil {
			return err
		}
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}

func runCheckAndStop(server *http.Server, listener net.Listener, serveErr <-chan error, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	checkErr := make(chan error, 1)
	go func() { checkErr <- webui.RunSelfcheck(ctx, "http://"+listener.Addr().String()) }()
	var result error
	select {
	case err := <-checkErr:
		result = err
	case err := <-serveErr:
		if err == nil {
			result = fmt.Errorf("自检期间服务意外停止")
		} else {
			result = err
		}
	case <-ctx.Done():
		result = fmt.Errorf("自检超时: %w", ctx.Err())
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil && result == nil {
		result = err
	}
	if result == nil {
		slog.Info("完整 HTTP 业务自检通过")
	}
	return result
}
