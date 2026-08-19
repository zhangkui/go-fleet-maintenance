// Package main 是车队车辆与维保调度系统的 API 入口。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhangkui/go-fleet-maintenance/internal/auth"
	"github.com/zhangkui/go-fleet-maintenance/internal/config"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/migration"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/redisx"
	"github.com/zhangkui/go-fleet-maintenance/internal/platform/seed"
	mysqlrepo "github.com/zhangkui/go-fleet-maintenance/internal/repository/mysql"
	"github.com/zhangkui/go-fleet-maintenance/internal/service"
	httpx "github.com/zhangkui/go-fleet-maintenance/internal/transport/http"
	"github.com/zhangkui/go-fleet-maintenance/internal/transport/http/handler"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}

	// 连接 MySQL。
	db, err := mysql.Open(cfg.MySQLDSN)
	if err != nil {
		slog.Error("打开 MySQL 失败", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// 等待 MySQL 就绪。
	if err := waitFor(db.PingContext, 60, time.Second); err != nil {
		slog.Error("等待 MySQL 就绪失败", "err", err)
		os.Exit(1)
	}

	// 连接 Redis。
	rdb, err := redisx.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisKeyPrefix)
	if err != nil {
		slog.Error("打开 Redis 失败", "err", err)
		os.Exit(1)
	}
	if err := waitFor(rdb.Ping, 60, time.Second); err != nil {
		slog.Error("等待 Redis 就绪失败", "err", err)
		os.Exit(1)
	}

	// 执行迁移（幂等）。
	mr := migration.New(db)
	ran, err := mr.Run(context.Background())
	if err != nil {
		slog.Error("迁移失败", "err", err)
		os.Exit(1)
	}
	slog.Info("迁移完成", "applied", ran)

	// 构造事务管理器与仓储集合。
	transactor := mysqlrepo.NewTransactor(db)
	stores := transactor.Stores()

	// 审计服务。
	auditSvc := service.NewAuditService(stores.Audit)

	// 认证与令牌。
	tokens := auth.NewTokenService(cfg.JWTSecret, cfg.TokenIssuer, cfg.AccessTokenTTL)
	authSvc := service.NewAuthService(stores.Users, stores.Roles, stores.Sessions, auditSvc, rdb, tokens, cfg.RefreshTokenTTL, cfg.LoginRateLimit, cfg.LoginRateWindow)

	// 创建默认管理员与内置角色/权限。
	if err := seed.DefaultAdmin(context.Background(), stores.Users, stores.Roles, stores.Permissions, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		slog.Error("创建默认管理员失败", "err", err)
		os.Exit(1)
	}

	// 业务服务。
	userSvc := service.NewUserService(stores.Users, auditSvc)
	rbacSvc := service.NewRBACService(stores.Roles, stores.Permissions, auditSvc)
	vehicleSvc := service.NewVehicleService(stores.Vehicles, transactor, auditSvc, rdb)
	driverSvc := service.NewDriverService(stores.Drivers, auditSvc)
	tripSvc := service.NewTripService(stores.Trips, stores.Vehicles, stores.Fuel, stores.Maintenance, transactor, auditSvc, rdb)
	fuelSvc := service.NewFuelService(stores.Fuel, stores.Vehicles, transactor, auditSvc, rdb)
	maintenanceSvc := service.NewMaintenanceService(stores.Maintenance, stores.Vehicles, stores.Parts, transactor, auditSvc, rdb)
	partSvc := service.NewPartService(stores.Parts, transactor, auditSvc)
	reminderSvc := service.NewReminderService(stores.Reminders, stores.Vehicles, stores.Drivers, stores.Maintenance, transactor)
	reportSvc := service.NewReportService(stores.Reports, rdb)

	// 权限校验器。
	checker := handler.NewPermissionChecker(stores.Roles)

	// 处理器集合。
	handlers := &httpx.Handlers{
		Auth:        handler.NewAuthHandler(authSvc, cfg.RequestMaxBytes),
		User:        handler.NewUserHandler(userSvc, cfg.RequestMaxBytes),
		RBAC:        handler.NewRBACHandler(rbacSvc, cfg.RequestMaxBytes),
		Vehicle:     handler.NewVehicleHandler(vehicleSvc, cfg.RequestMaxBytes),
		Driver:      handler.NewDriverHandler(driverSvc, cfg.RequestMaxBytes),
		Trip:        handler.NewTripHandler(tripSvc, cfg.RequestMaxBytes),
		Fuel:        handler.NewFuelHandler(fuelSvc, cfg.RequestMaxBytes),
		Maintenance: handler.NewMaintenanceHandler(maintenanceSvc, cfg.RequestMaxBytes),
		Part:        handler.NewPartHandler(partSvc, cfg.RequestMaxBytes),
		Reminder:    handler.NewReminderHandler(reminderSvc, cfg.RequestMaxBytes),
		Report:      handler.NewReportHandler(reportSvc, cfg.RequestMaxBytes),
		Health:      handler.NewHealthHandler(db, rdb),
	}

	router := httpx.New(tokens, checker, cfg.RequestMaxBytes)
	root := router.Handler(handlers)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// 优雅关闭。
	go func() {
		slog.Info("HTTP 服务启动", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务退出", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("收到退出信号，开始优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = rdb.Rdb().Close()
	slog.Info("已关闭")
}

// waitFor 反复重试 fn 直到成功或超过最大次数。
func waitFor(fn func(context.Context) error, max int, interval time.Duration) error {
	ctx := context.Background()
	var err error
	for i := 0; i < max; i++ {
		if err = fn(ctx); err == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return err
}
