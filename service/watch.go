package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

// DefaultDebounceDuration 文件变动防抖观察窗口期
var DefaultDebounceDuration = 200 * time.Millisecond

// DebounceAggregator 文件变动事件防抖合并器
type DebounceAggregator struct {
	duration time.Duration
	eventsCh chan debounceItem
	done     chan struct{}
	handler  func(files []string)
}

type debounceItem struct {
	event string
	name  string
}

// NewDebounceAggregator 创建防抖聚合器
func NewDebounceAggregator(duration time.Duration, handler func(files []string)) *DebounceAggregator {
	if duration <= 0 {
		duration = DefaultDebounceDuration
	}
	da := &DebounceAggregator{
		duration: duration,
		eventsCh: make(chan debounceItem, 128),
		done:     make(chan struct{}),
		handler:  handler,
	}
	go da.loop()
	return da
}

// Push 压入文件变动事件
func (da *DebounceAggregator) Push(event, name string) {
	select {
	case <-da.done:
		return
	default:
	}

	select {
	case da.eventsCh <- debounceItem{event: event, name: name}:
	default:
		// 通道满时安全丢弃或由现有事件覆盖聚合
	}
}

// Close 关闭防抖器并释放资源
func (da *DebounceAggregator) Close() {
	select {
	case <-da.done:
		return
	default:
		close(da.done)
	}
}

func (da *DebounceAggregator) loop() {
	var timer *time.Timer
	var timerCh <-chan time.Time
	pending := make(map[string]bool)

	for {
		select {
		case <-da.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case item := <-da.eventsCh:
			pending[item.name] = true
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(da.duration)
			timerCh = timer.C

		case <-timerCh:
			timer = nil
			timerCh = nil

			if len(pending) == 0 {
				continue
			}

			files := make([]string, 0, len(pending))
			for f := range pending {
				files = append(files, f)
			}
			pending = make(map[string]bool)

			if da.handler != nil {
				da.handler(files)
			}
		}
	}
}

// Watch the application code change for hot update
func Watch(srv *http.Server, interrupt chan uint8) (err error) {

	if application.App == nil {
		return fmt.Errorf("Application is not initialized")
	}

	internalInterrupt := make(chan uint8, 1)

	aggregator := NewDebounceAggregator(DefaultDebounceDuration, func(files []string) {
		handleBatchedReload(srv, files)
	})
	defer aggregator.Close()

	// 转发外部中断信号
	go func() {
		select {
		case sig := <-interrupt:
			aggregator.Close()
			internalInterrupt <- sig
		case <-aggregator.done:
			return
		}
	}()

	return application.App.Watch(func(event, name string) {
		if strings.Contains(event, "CHMOD") {
			return
		}
		aggregator.Push(event, name)
	}, internalInterrupt)
}

func handleBatchedReload(srv *http.Server, files []string) {
	hasModels := false
	hasApis := false

	for _, file := range files {
		clean := strings.TrimPrefix(file, "/")
		if strings.HasPrefix(clean, "models") {
			hasModels = true
		}
		if strings.HasPrefix(clean, "apis") {
			hasApis = true
		}
	}

	// 1. 批量触发单次重载
	fmt.Printf("[Watch] Batched %d changed file(s): %s\n", len(files), strings.Join(files, ", "))
	err := engine.Reload(config.Conf, engine.LoadOption{Action: "watch"})
	if err != nil {
		fmt.Println(color.RedString("[Watch] Reload: %s", err.Error()))
		return
	}
	fmt.Println(color.GreenString("[Watch] Reload Completed"))

	// 2. 数据库模型变更提示
	if hasModels {
		fmt.Println(color.YellowString("[Watch] Models changed (Please run 'yao migrate' manually if schema changed)"))
	}

	// 3. API 路由热替换 (Hot Swap)
	if hasApis && srv != nil {
		err = Restart(srv, config.Conf)
		if err != nil {
			fmt.Println(color.RedString("[Watch] Restart: %s", err.Error()))
			return
		}
		fmt.Println(color.GreenString("[Watch] Restart Completed (Hot Swap)"))
	}
}

