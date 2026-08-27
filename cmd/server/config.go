package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	addr             string
	database         string
	selfcheck        bool
	selfcheckTimeout time.Duration
}

func parseConfig() (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return config{}, fmt.Errorf("PORT 必须是端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", port)
	}
	var cfg config
	flag.StringVar(&cfg.addr, "addr", defaultAddr, "HTTP 监听地址（仅允许回环地址）")
	flag.StringVar(&cfg.database, "db", "caption-audit.db", "SQLite 数据库路径")
	flag.BoolVar(&cfg.selfcheck, "selfcheck", false, "启动真实 HTTP 监听并执行完整业务自检后退出")
	flag.DurationVar(&cfg.selfcheckTimeout, "selfcheck-timeout", 15*time.Second, "自检总超时")
	flag.Parse()
	if err := validateAddress(cfg.addr); err != nil {
		return config{}, err
	}
	if cfg.selfcheckTimeout <= 0 {
		return config{}, fmt.Errorf("selfcheck-timeout 必须大于零")
	}
	if cfg.selfcheck && cfg.database == "caption-audit.db" {
		cfg.database = fmt.Sprintf("file:selfcheck-%d?mode=memory&cache=shared", os.Getpid())
	}
	return cfg, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 格式无效: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("addr 端口必须在 1 到 65535 之间")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("addr 仅允许回环地址，拒绝 %q", host)
	}
	return nil
}
