package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/helper"

	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/dashboard"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/list"
	"github.com/yaoapp/yao/widgets/table"
)

// Guards middlewares
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt":       guardBearerJWT,   // Bearer JWT
	"query-jwt":        guardQueryJWT,    // Get JWT Token from query string  "__tk"
	"cross-origin":     guardCrossOrigin, // Cross-Origin Resource Sharing
	"cookie-trace":     guardCookieTrace, // Set sid cookie
	"cookie-jwt":       guardCookieJWT,   // Get JWT Token from cookie "__tk"
	"widget-table":     table.Guard,      // Widget Table Guard
	"widget-list":      list.Guard,       // Widget List Guard
	"widget-form":      form.Guard,       // Widget Form Guard
	"widget-chart":     chart.Guard,      // Widget Chart Guard
	"widget-dashboard": dashboard.Guard,  // Widget Dashboard Guard
}

// guardCookieTrace set sid cookie
func guardCookieTrace(c *gin.Context) {
	sid, err := c.Cookie("sid")
	if err != nil {
		sid = uuid.New().String()
		c.SetCookie("sid", sid, 0, "/", "", false, true)
		c.Set("__sid", sid)
		c.Next()
		return
	}
	c.Set("__sid", sid)
	return
}

// Cookie Cookie JWT
func guardCookieJWT(c *gin.Context) {
	tokenString, err := c.Cookie("__tk")
	if err != nil || tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	claims, err := helper.JwtVerify(tokenString)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "message": "Invalid token", "context": err.Error()})
		c.Abort()
		return
	}
	c.Set("__sid", claims.SID)
	hydrateGlobalSession(c, claims.SID)
	return
}

// JWT Bearer JWT
func guardBearerJWT(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	claims, err := helper.JwtVerify(tokenString)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "message": "Invalid token", "context": err.Error()})
		c.Abort()
		return
	}
	c.Set("__sid", claims.SID)
	hydrateGlobalSession(c, claims.SID)
}

// JWT Query JWT
func guardQueryJWT(c *gin.Context) {
	tokenString := c.Query("__tk")
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	claims, err := helper.JwtVerify(tokenString)
	if err != nil {
		c.JSON(401, gin.H{"code": 401, "message": "Invalid token", "context": err.Error()})
		c.Abort()
		return
	}
	c.Set("__sid", claims.SID)
	hydrateGlobalSession(c, claims.SID)
}

// hydrateGlobalSession 将 Session 存储中的会话状态注水至 Gin 上下文的 __global 中
func hydrateGlobalSession(c *gin.Context, sid string) {
	if sid == "" {
		return
	}
	var global map[string]interface{}
	if v, exists := c.Get("__global"); exists {
		if m, ok := v.(map[string]interface{}); ok {
			global = m
		}
	}
	if global == nil {
		global = make(map[string]interface{})
	}

	// 1. 尝试从 Session 存储中全量检出会话
	if dump, err := session.Global().ID(sid).Dump(); err == nil && dump != nil {
		for k, val := range dump {
			if _, exists := global[k]; !exists {
				global[k] = val
			}
		}
	}

	// 2. 检查请求头中的活跃上下文覆盖 (如 H5/工作台切换器 X-Active-Campus-Id)
	if c.Request != nil && c.Request.Header != nil {
		if cid := c.Request.Header.Get("X-Active-Campus-Id"); cid != "" {
			if _, exists := global["active_campus_id"]; !exists {
				global["active_campus_id"] = cid
			}
		}
		if did := c.Request.Header.Get("X-Active-Dept-Id"); did != "" {
			if _, exists := global["active_dept_id"]; !exists {
				global["active_dept_id"] = did
			}
		}
		if wid := c.Request.Header.Get("X-Active-Ward-Id"); wid != "" {
			if _, exists := global["active_ward_id"]; !exists {
				global["active_ward_id"] = wid
			}
		}
	}

	c.Set("__global", global)
}

// CORS Cross Origin
func guardCrossOrigin(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}
	c.Next()
}
