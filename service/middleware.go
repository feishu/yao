package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/sui/api"
)

// Middlewares the middlewares
var Middlewares = []gin.HandlerFunc{
	withInFlightTrack,
	withRecovery,
	gin.Logger(),
	withStaticFileServer,
}

// maxConcurrency 全局网关最大在途并发限制，0 为不限制
var maxConcurrency int64 = 0

func init() {
	if val := os.Getenv("YAO_MAX_CONCURRENCY"); val != "" {
		if limit, err := strconv.ParseInt(val, 10, 64); err == nil && limit > 0 {
			maxConcurrency = limit
		}
	}
}

// SetMaxConcurrency 动态配置网关最大在途并发限制（0 为不限制）
func SetMaxConcurrency(limit int64) {
	atomic.StoreInt64(&maxConcurrency, limit)
}

// GetMaxConcurrency 获取网关最大在途并发限制
func GetMaxConcurrency() int64 {
	return atomic.LoadInt64(&maxConcurrency)
}

// withInFlightTrack 全局在途请求追踪与过载保护，支持平滑热重载优雅排空与过载快速失败
func withInFlightTrack(c *gin.Context) {
	limit := atomic.LoadInt64(&maxConcurrency)
	leave, ok := share.InFlightTryEnter(limit)
	if !ok {
		log.Warn("[Gateway] concurrency overload limit %d reached (in-flight %d), shedding load", limit, share.GlobalInFlight.Count())
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"code":    http.StatusTooManyRequests,
			"message": "Server overloaded, please retry later",
		})
		return
	}
	defer leave()
	c.Next()
}

// withRecovery 全局网关未捕获 Panic 恢复与安全兜底中间件，防止进程异常退出并保障在途请求排空
func withRecovery(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			stack := string(debug.Stack())
			log.Error("[Gateway Recovery] panic recovered: %s\nstack:\n%s", errStr, stack)

			if c.Writer != nil && !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "Internal Server Error",
				})
			} else {
				c.Abort()
			}
		}
	}()
	c.Next()
}

// withStaticFileServer static file server
func withStaticFileServer(c *gin.Context) {

	// Handle API, websocket & internal __yao routes (like dbadmin, including subpaths)
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/websocket/") ||
		strings.Contains(path, "/__yao/") ||
		strings.HasSuffix(path, "/__yao/db") {
		c.Next()
		return
	}

	// Xgen 1.0
	// if length >= AdminRootLen && c.Request.URL.Path[0:AdminRootLen] == AdminRoot {
	// 	c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, c.Request.URL.Path[0:AdminRootLen-1])
	// 	XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
	// 	c.Abort()
	// 	return
	// }

	// __yao_admin_root
	// if length >= 18 && c.Request.URL.Path[0:18] == "/__yao_admin_root/" {
	// 	c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/__yao_admin_root")
	// 	XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
	// 	c.Abort()
	// 	return
	// }

	// Rewrite
	for _, rewrite := range rewriteRules {
		// log.Debug("Rewrite: %s => %s", c.Request.URL.Path, rewrite.Replacement)
		if matches := rewrite.Pattern.FindStringSubmatch(c.Request.URL.Path); matches != nil {
			c.Set("rewrite", true)
			c.Set("matches", matches)
			c.Request.URL.Path = rewrite.Pattern.ReplaceAllString(c.Request.URL.Path, rewrite.Replacement)
			// rewriteOriginalPath := c.Request.URL.Path
			// log.Trace("Rewrite FindStringSubmatch Matched: %s => %s", rewriteOriginalPath, rewrite.Replacement)
			break
		}
	}

	// Sui file server
	if strings.HasSuffix(c.Request.URL.Path, ".sui") {

		// Default index.sui
		if filepath.Base(c.Request.URL.Path) == ".sui" {
			c.Request.URL.Path = strings.TrimSuffix(c.Request.URL.Path, ".sui") + "index.sui"
		}

		r, code, err := api.NewRequestContext(c)
		if err != nil {
			log.Error("Sui Reqeust Error: %s", err.Error())
			c.AbortWithStatusJSON(code, gin.H{"code": code, "message": err.Error()})
			return
		}

		html, code, err := r.Render()
		if err != nil {
			if code == 301 || code == 302 {
				url := err.Error()
				// fmt.Println("Redirect to: ", url)
				c.Redirect(code, url)
				c.Done()
				return
			}

			log.Error("Sui Render Error: %s", err.Error())
			c.AbortWithStatusJSON(code, gin.H{"code": code, "message": err.Error()})
			return
		}

		// Gzip Compression option
		if share.App.Static.DisableGzip == false && strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			if _, err := gz.Write([]byte(html)); err != nil {
				log.Error("GZIP Compression Error: %s", err.Error())
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if err := gz.Close(); err != nil {
				log.Error("GZIP Close Error: %s", err.Error())
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Accept-Ranges", "bytes")
			c.Header("Content-Encoding", "gzip")
			c.Data(http.StatusOK, "text/html", buf.Bytes())
			c.Done()
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, html)
		c.Next()
		return
	}

	// static file server
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
