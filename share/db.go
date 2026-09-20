package share

import (
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

var dbKeepAliveStop chan struct{}

// applyAllPoolSettings 配置连接池生命周期与连接数
func applyAllPoolSettings(manager *capsule.Manager, dbconfig config.Database) {
	maxIdle := dbconfig.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	maxOpen := dbconfig.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	idleTime := dbconfig.ConnMaxIdleTime
	if idleTime <= 0 {
		idleTime = 180 // 默认 3 分钟，提前于防火墙超时
	}
	lifetime := dbconfig.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = 1800 // 默认 30 分钟
	}

	if manager.Connections != nil {
		manager.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.DB.SetMaxIdleConns(maxIdle)
				conn.DB.SetMaxOpenConns(maxOpen)
				conn.DB.SetConnMaxIdleTime(time.Duration(idleTime) * time.Second)
				conn.DB.SetConnMaxLifetime(time.Duration(lifetime) * time.Second)
			}
			return true
		})
	}
}

// startDBKeepAlive 启动后台长连接定时探活保活协程
func startDBKeepAlive(manager *capsule.Manager) {
	if dbKeepAliveStop != nil {
		close(dbKeepAliveStop)
	}
	dbKeepAliveStop = make(chan struct{})
	stop := dbKeepAliveStop

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if capsule.Global == nil || capsule.Global.Connections == nil {
					continue
				}
				capsule.Global.Connections.Range(func(key, value any) bool {
					if conn, ok := value.(*capsule.Connection); ok {
						if err := conn.Ping(3 * time.Second); err != nil {
							log.Warn("[DBKeepAlive] %s ping failed: %v", conn.Config.Name, err)
						}
					}
					return true
				})
			}
		}
	}()
}

// DBConnect 建立数据库连接
func DBConnect(dbconfig config.Database) (err error) {

	if dbconfig.Primary == nil {
		return fmt.Errorf("YAO_DB_PRIMARY was not set")
	}

	manager := capsule.New()
	for i, dsn := range dbconfig.Primary {
		_, err = manager.Add(fmt.Sprintf("primary-%d", i), dbconfig.Driver, dsn, false)
		if err != nil {
			return err
		}
	}

	if dbconfig.Secondary != nil {
		for i, dsn := range dbconfig.Secondary {
			_, err = manager.Add(fmt.Sprintf("secondary-%d", i), dbconfig.Driver, dsn, true)
			if err != nil {
				return err
			}
		}
	}

	manager.SetAsGlobal()

	// 配置连接池参数与生命周期
	applyAllPoolSettings(manager, dbconfig)

	// 启动初次探活
	go func() {
		for _, c := range manager.Pool.Primary {
			err = c.Ping(5 * time.Second)
			if err != nil {
				log.Error("%s error %v", c.Config.Name, err.Error())
			}
		}
	}()

	// 启动常驻后台探活保活协程
	startDBKeepAlive(manager)

	return err
}

// DBClose close the database connections
func DBClose() error {
	if dbKeepAliveStop != nil {
		close(dbKeepAliveStop)
		dbKeepAliveStop = nil
	}

	messages := []string{}
	capsule.Global.Connections.Range(func(key, value any) bool {
		log.Trace("[DBClose] %s", key)
		if conn, ok := value.(*capsule.Connection); ok {
			err := conn.Close()
			if err != nil {
				messages = append(messages, err.Error())
			}
		}
		return true
	})

	if len(messages) > 0 {
		msg := fmt.Sprintf("[DBClose] %s ", strings.Join(messages, ";"))
		log.Error("%s", msg)
		return fmt.Errorf("%s", msg)
	}

	return nil
}
