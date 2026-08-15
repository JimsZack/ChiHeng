// Package scheduler 实现定投计划的自动执行：交易时点触发、幂等账本防重复。
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/JimsZack/ChiHeng/internal/service"
)

// Runner 执行到期定投计划（由 app 层注入，避免 scheduler 依赖具体服务实现细节）。
type Runner interface {
	// ExecuteDue 检查并执行所有到期的定投计划，返回执行条数。
	ExecuteDue(ctx context.Context) (int, error)
}

// Scheduler 定时器：在交易时段内周期性触发 ExecuteDue。
type Scheduler struct {
	runner   Runner
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
}

// New 创建调度器。interval 为检查周期（默认 60s）。
func New(runner Runner, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Scheduler{
		runner:   runner,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start 启动调度循环（非阻塞，后台 goroutine）。
func (s *Scheduler) Start() {
	s.once.Do(func() {
		go s.loop()
	})
}

// Stop 停止调度循环并等待退出。
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Scheduler) loop() {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.runOnce()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.runOnce()
		}
	}
}

func (s *Scheduler) runOnce() {
	// 仅在交易日时段执行（周一至周五 09:30–15:00，简化：不含节假日历）。
	if !isTradingWindow(time.Now()) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runWithRecover(ctx, s.runner); err != nil {
		slog.Error("执行定投计划失败", "error", err)
		return
	}
}

func runWithRecover(ctx context.Context, runner Runner) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("执行定投计划 panic", "recover", r)
			err = service.ErrPanic
		}
	}()
	count, err := runner.ExecuteDue(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("定投计划执行完成", "count", count)
	}
	return nil
}

// isTradingWindow 判断当前是否为交易日交易时段（周一至周五 09:30–15:00）。
func isTradingWindow(now time.Time) bool {
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	local := now.Local()
	minutes := local.Hour()*60 + local.Minute()
	return minutes >= 9*60+30 && minutes <= 15*60
}
