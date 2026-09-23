package dbadmin

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
)

//go:embed dist/*
var distFS embed.FS

// Enabled 全局启用标志
var Enabled = false

// SetEnabled 设置开启状态
func SetEnabled(val bool) {
	Enabled = val
}

// IsEnabled 检查是否开启数据库 Web Admin
func IsEnabled() bool {
	if Enabled {
		return true
	}
	env := strings.ToLower(os.Getenv("YAO_DB_ADMIN"))
	return env == "true" || env == "1" || env == "yes" || env == "on"
}

// Mount 挂载数据库管理系统的静态页面与 RESTful API 路由
func Mount(router *gin.Engine, cfg config.Config) {
	if !IsEnabled() {
		return
	}

	distSub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return
	}

	serveHTML := func(c *gin.Context) {
		data, err := fs.ReadFile(distSub, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load DB Admin index.html")
			return
		}
		// 若环境变量配置了 YAO_DB_ADMIN_ROOT (例如 /syd)，注入到 HTML head 优先使用
		if root := strings.TrimSpace(os.Getenv("YAO_DB_ADMIN_ROOT")); root != "" {
			cleanRoot := "/" + strings.Trim(root, "/")
			inject := "<script>window.__YAO_DB_ADMIN_ROOT__ = " + strconv.Quote(cleanRoot) + ";</script>\n  "
			data = []byte(strings.Replace(string(data), "<head>", "<head>\n  "+inject, 1))
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}

	mountRoutes := func(base string) {
		base = "/" + strings.Trim(base, "/")

		// 1. API 路由组
		apiGroup := router.Group(base + "/api")
		apiGroup.Use(authGuard(cfg))
		{
			apiGroup.GET("/status", handleStatus)
			apiGroup.GET("/tables", handleGetTables)
			apiGroup.GET("/tables/:name/schema", handleGetTableSchema)
			apiGroup.GET("/tables/:name/data", handleGetTableData)
			apiGroup.POST("/tables/:name/data", handleInsertTableData)
			apiGroup.PUT("/tables/:name/data", handleUpdateTableData)
			apiGroup.DELETE("/tables/:name/data", handleDeleteTableData)
			apiGroup.POST("/tables/:name/batch", handleBatchCommit)
			apiGroup.GET("/tables/:name/ddl", handleGetTableDDL)
			apiGroup.POST("/sql/execute", handleExecuteSQL)
		}

		// 2. 静态资源路由 (支持 base, CSS, JS)
		router.GET(base, serveHTML)
		router.GET(base+"/", serveHTML)
		router.GET(base+"/tabulator.min.js", func(c *gin.Context) {
			data, err := fs.ReadFile(distSub, "tabulator.min.js")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
		})
		router.GET(base+"/tabulator_midnight.min.css", func(c *gin.Context) {
			data, err := fs.ReadFile(distSub, "tabulator_midnight.min.css")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "text/css; charset=utf-8", data)
		})
	}

	// 1. 默认挂载标准根路径 /__yao/db
	mountRoutes("/__yao/db")

	// 2. 若显式配置了 YAO_DB_ADMIN_ROOT 环境变量 (如 /syd)，则额外挂载对应前缀路由
	if envRoot := strings.TrimSpace(os.Getenv("YAO_DB_ADMIN_ROOT")); envRoot != "" {
		cleanRoot := "/" + strings.Trim(envRoot, "/")
		if cleanRoot != "/" && cleanRoot != "/__yao" && cleanRoot != "/__yao/db" {
			mountRoutes(cleanRoot + "/__yao/db")
		}
	}
}

// authGuard 认证中间件
func authGuard(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 若显式配置 YAO_DB_ADMIN_AUTH=false，则直接免密放行
		if strings.ToLower(os.Getenv("YAO_DB_ADMIN_AUTH")) == "false" {
			c.Next()
			return
		}

		// 1. 检查环境变量配置的独立安全密钥
		adminKey := os.Getenv("YAO_DB_ADMIN_KEY")
		if adminKey != "" {
			reqKey := c.GetHeader("X-DB-Admin-Key")
			if reqKey == "" {
				reqKey = c.Query("key")
			}
			if reqKey == adminKey {
				c.Next()
				return
			}
		}

		// 2. 检查 Yao Admin JWT 身份（支持 Bearer 头与 Cookie __tk）
		token := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if token == "" {
			if cookie, err := c.Cookie("__tk"); err == nil && cookie != "" {
				token = cookie
			}
		}

		if token != "" {
			if claims, err := helper.JwtVerify(token); err == nil && claims != nil {
				c.Set("__sid", claims.SID)
				c.Next()
				return
			}
		}

		// 3. 在开发模式且未设置任何独立密钥时，允许本地便捷访问
		if adminKey == "" && token == "" && cfg.Mode == "development" && os.Getenv("YAO_DB_ADMIN_AUTH") != "true" {
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问：请在请求头提供有效的 X-DB-Admin-Key 或管理员 Bearer Token",
		})
		c.Abort()
	}
}
