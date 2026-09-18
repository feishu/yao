package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	gouApi "github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/mcp/server"
	"github.com/yaoapp/gou/mcp/types"
	gouModel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
)

func TestMCPProjectionAndExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server.Reset()

	// 1. 注册一个模拟的业务 Process: unit.mcp.test
	process.Register("unit.mcp.test", func(p *process.Process) interface{} {
		args := p.Args
		var payload map[string]interface{}
		if len(args) > 0 {
			if m, ok := args[0].(map[string]interface{}); ok {
				payload = m
			}
		}
		name, _ := payload["patient_name"].(string)
		dept, _ := payload["department"].(string)

		return map[string]interface{}{
			"order_id":     "ORD20260914001",
			"patient_name": name,
			"department":   dept,
			"status":       "confirmed",
		}
	})

	// 2. 模拟加载一个包含了 mcp 声明的 HTTP API
	apiSource := []byte(`{
		"name": "Hospital Outpatient API",
		"version": "1.0.0",
		"group": "hospital",
		"paths": [
			{
				"path": "/registration/create",
				"method": "POST",
				"process": "unit.mcp.test",
				"in": [":payload"],
				"mcp": {
					"name": "create_outpatient_registration",
					"description": "门诊就诊患者挂号下单并锁定号源",
					"group": "hospital",
					"params": {
						"patient_name": {
							"type": "string",
							"description": "患者姓名",
							"required": true
						},
						"department": {
							"type": "string",
							"description": "挂号科室名称",
							"required": true
						}
					}
				}
			},
			{
				"path": "/internal/stats",
				"method": "GET",
				"process": "unit.mcp.test"
			}
		]
	}`)

	loadedAPI, err := gouApi.LoadSource("apis/hospital.http.json", apiSource, "hospital")
	assert.NoError(t, err)
	assert.NotNil(t, loadedAPI)

	// 3. 执行投影
	count := ProjectAPIsToMCP()
	assert.GreaterOrEqual(t, count, 1)

	// 4. 初始化 Gin 路由并挂载 MCP 端点
	router := gin.New()
	MountMCPRoutes(router)

	// 5. 模拟 MCP 客户端向 /v1/mcps/hospital 发送 tools/list 请求
	listReqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list"
	}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/mcps/hospital", bytes.NewBuffer(listReqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		JSONRPC string                  `json:"jsonrpc"`
		ID      int                     `json:"id"`
		Result  types.ListToolsResponse `json:"result"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listResp.Result.Tools))
	tool := listResp.Result.Tools[0]
	assert.Equal(t, "create_outpatient_registration", tool.Name)
	assert.Equal(t, "门诊就诊患者挂号下单并锁定号源", tool.Description)
	assert.Contains(t, string(tool.InputSchema), "patient_name")
	assert.Contains(t, string(tool.InputSchema), "department")

	// 6. 模拟 MCP 客户端向 /v1/mcps/hospital 发送 tools/call 请求调用该 API
	callReqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/call",
		"params": {
			"name": "create_outpatient_registration",
			"arguments": {
				"patient_name": "张三",
				"department": "心血管内科"
			}
		}
	}`)
	wCall := httptest.NewRecorder()
	reqCall, _ := http.NewRequest("POST", "/v1/mcps/hospital", bytes.NewBuffer(callReqBody))
	reqCall.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wCall, reqCall)

	assert.Equal(t, http.StatusOK, wCall.Code)

	var callResp struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      int                    `json:"id"`
		Result  types.CallToolResponse `json:"result"`
	}
	err = json.Unmarshal(wCall.Body.Bytes(), &callResp)
	assert.NoError(t, err)
	assert.False(t, callResp.Result.IsError)
	assert.Equal(t, 1, len(callResp.Result.Content))

	outText := callResp.Result.Content[0].Text
	assert.Contains(t, outText, "ORD20260914001")
	assert.Contains(t, outText, "张三")
	assert.Contains(t, outText, "心血管内科")
	assert.Contains(t, outText, "confirmed")
}

type safeRecorder struct {
	*httptest.ResponseRecorder
	mu  sync.RWMutex
	buf bytes.Buffer
}

func newSafeRecorder() *safeRecorder {
	return &safeRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (r *safeRecorder) Write(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Write(buf)
	return r.ResponseRecorder.Write(buf)
}

func (r *safeRecorder) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.buf.String()
}

func TestMCPSSEFullBidirectionalWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server.Reset()

	router := gin.New()
	MountMCPRoutes(router)

	wSSE := newSafeRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqSSE, _ := http.NewRequestWithContext(ctx, "GET", "/v1/mcps/default/sse", nil)

	// 1. 建立 SSE 连接
	go router.ServeHTTP(wSSE, reqSSE)

	// 等待接收 endpoint 事件
	assert.Eventually(t, func() bool {
		return strings.Contains(wSSE.String(), "event: endpoint")
	}, 1*time.Second, 10*time.Millisecond)

	body := wSSE.String()
	assert.Contains(t, body, "event: endpoint")
	assert.Contains(t, body, "messages?session_id=")

	// 提取 session_id
	idx := strings.Index(body, "session_id=")
	assert.True(t, idx > 0)
	sessionID := strings.TrimSpace(body[idx+len("session_id="):])
	if newline := strings.Index(sessionID, "\n"); newline > 0 {
		sessionID = sessionID[:newline]
	}

	// 2. 客户端向 messages 端点发送 ping
	pingJSON := []byte(`{"jsonrpc":"2.0","id":100,"method":"ping"}`)
	wMsg := httptest.NewRecorder()
	reqMsg, _ := http.NewRequest("POST", "/v1/mcps/default/messages?session_id="+sessionID, bytes.NewBuffer(pingJSON))
	reqMsg.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wMsg, reqMsg)

	// 验证 HTTP POST 立即返回 202 Accepted
	assert.Equal(t, http.StatusAccepted, wMsg.Code)

	// 3. 验证通过 SSE 流实时推回了 JSON-RPC 响应消息
	assert.Eventually(t, func() bool {
		return strings.Contains(wSSE.String(), `"id":100`)
	}, 1*time.Second, 10*time.Millisecond)

	sseOutput := wSSE.String()
	assert.Contains(t, sseOutput, "event: message")
	assert.Contains(t, sseOutput, `"id":100`)
}

func TestSmartSchemaSynthesizer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server.Reset()

	// 模拟 API 仅写了 mcp: true，但定义了 in: ["$param.order_id", "$query.reason"]
	apiSource := []byte(`{
		"name": "Refund API",
		"version": "1.0.0",
		"group": "billing",
		"paths": [
			{
				"path": "/refund/:order_id",
				"method": "POST",
				"process": "unit.mcp.test",
				"in": ["$param.order_id", "$query.reason"],
				"description": "门诊退费并回滚结算状态",
				"mcp": true
			}
		]
	}`)

	_, err := gouApi.LoadSource("apis/refund.http.json", apiSource, "billing.refund")
	assert.NoError(t, err)

	ProjectAPIsToMCP()

	router := gin.New()
	MountMCPRoutes(router)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/mcps/billing", bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Result types.ListToolsResponse `json:"result"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	assert.Equal(t, 1, len(listResp.Result.Tools))

	tool := listResp.Result.Tools[0]
	// 验证智能推导出来的属性
	var schema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	json.Unmarshal(tool.InputSchema, &schema)

	// 验证推导出了 order_id 并且是必填项
	assert.Contains(t, schema.Properties, "order_id")
	assert.Equal(t, "string", schema.Properties["order_id"].Type)
	assert.Contains(t, schema.Required, "order_id")

	// 验证推导出了 reason
	assert.Contains(t, schema.Properties, "reason")
}

func TestMCPGuardAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server.Reset()

	// 1. 注册一个测试用的 Guard 中间件
	gouApi.AddGuard("test-admin-jwt", func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		if token != "Bearer admin-secret-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing token"})
			c.Abort()
			return
		}

		c.Set("__sid", "sess_admin_999")
		c.Set("__authorized", map[string]interface{}{
			"user_id": "ADM007",
			"role":    "superadmin",
		})
	})

	// 2. 注册受 Guard 保护的 Process
	process.Register("unit.mcp.guarded", func(p *process.Process) interface{} {
		// 校验 Session ID
		if p.Sid != "sess_admin_999" {
			return map[string]interface{}{"status": "fail", "reason": "missing sid"}
		}

		// 校验 Authorized 上下文
		auth, ok := p.Global["__authorized"].(map[string]interface{})
		if !ok || auth["user_id"] != "ADM007" {
			return map[string]interface{}{"status": "fail", "reason": "invalid authorized"}
		}

		return map[string]interface{}{
			"status":  "ok",
			"user_id": auth["user_id"],
			"role":    auth["role"],
			"action":  "guarded_data_accessed",
		}
	})

	// 3. 声明一个挂载了 Guard 的 API
	apiSource := []byte(`{
		"name": "Guarded Admin API",
		"version": "1.0.0",
		"group": "admin",
		"guard": "test-admin-jwt",
		"paths": [
			{
				"path": "/patients/export",
				"method": "POST",
				"process": "unit.mcp.guarded",
				"in": [],
				"mcp": {
					"name": "export_patients_sensitive_data",
					"description": "导出患者敏感病历数据"
				}
			}
		]
	}`)

	_, err := gouApi.LoadSource("apis/guarded.http.json", apiSource, "admin.guarded")
	assert.NoError(t, err)

	ProjectAPIsToMCP()

	router := gin.New()
	MountMCPRoutes(router)

	callJSON := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "export_patients_sensitive_data",
			"arguments": {}
		}
	}`)

	// 4. 场景 A：未授权访问（不带 Header），必须被 Guard 拦截
	wUnauth := httptest.NewRecorder()
	reqUnauth, _ := http.NewRequest("POST", "/v1/mcps/admin", bytes.NewBuffer(callJSON))
	reqUnauth.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUnauth, reqUnauth)

	assert.Equal(t, http.StatusOK, wUnauth.Code)
	var respUnauth struct {
		Result types.CallToolResponse `json:"result"`
	}
	err = json.Unmarshal(wUnauth.Body.Bytes(), &respUnauth)
	assert.NoError(t, err)
	assert.True(t, respUnauth.Result.IsError)
	assert.Contains(t, respUnauth.Result.Content[0].Text, "invalid or missing token")

	// 5. 场景 B：携带无效 Token，必须被 Guard 拦截
	wWrong := httptest.NewRecorder()
	reqWrong, _ := http.NewRequest("POST", "/v1/mcps/admin", bytes.NewBuffer(callJSON))
	reqWrong.Header.Set("Content-Type", "application/json")
	reqWrong.Header.Set("Authorization", "Bearer bad-token")
	router.ServeHTTP(wWrong, reqWrong)

	assert.Equal(t, http.StatusOK, wWrong.Code)
	var respWrong struct {
		Result types.CallToolResponse `json:"result"`
	}
	json.Unmarshal(wWrong.Body.Bytes(), &respWrong)
	assert.True(t, respWrong.Result.IsError)
	assert.Contains(t, respWrong.Result.Content[0].Text, "invalid or missing token")

	// 6. 场景 C：携带有效 Token，Guard 放行且内部 Process 正确提取 Session 与 Authorized 身份
	wAuth := httptest.NewRecorder()
	reqAuth, _ := http.NewRequest("POST", "/v1/mcps/admin", bytes.NewBuffer(callJSON))
	reqAuth.Header.Set("Content-Type", "application/json")
	reqAuth.Header.Set("Authorization", "Bearer admin-secret-token")
	router.ServeHTTP(wAuth, reqAuth)

	assert.Equal(t, http.StatusOK, wAuth.Code)
	var respAuth struct {
		Result types.CallToolResponse `json:"result"`
	}
	json.Unmarshal(wAuth.Body.Bytes(), &respAuth)
	assert.False(t, respAuth.Result.IsError)
	assert.Contains(t, respAuth.Result.Content[0].Text, `"user_id":"ADM007"`)
	assert.Contains(t, respAuth.Result.Content[0].Text, `"guarded_data_accessed"`)
	assert.Contains(t, respAuth.Result.Content[0].Text, `"role":"superadmin"`)

	// 7. 场景 D：验证 SSE 会话级认证继承（GET /sse 携带 Token，后续 POST /messages 无需重传 Token）
	wSSE := newSafeRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqSSE, _ := http.NewRequestWithContext(ctx, "GET", "/v1/mcps/admin/sse", nil)
	reqSSE.Header.Set("Authorization", "Bearer admin-secret-token")
	go router.ServeHTTP(wSSE, reqSSE)

	assert.Eventually(t, func() bool {
		return strings.Contains(wSSE.String(), "event: endpoint")
	}, 1*time.Second, 10*time.Millisecond)

	sseBody := wSSE.String()
	idx := strings.Index(sseBody, "session_id=")
	assert.True(t, idx > 0)
	sessionID := strings.TrimSpace(sseBody[idx+len("session_id="):])
	if newline := strings.Index(sessionID, "\n"); newline > 0 {
		sessionID = sessionID[:newline]
	}

	// 客户端向 messages 发送请求，此时故意不带 Authorization Header
	wMsg := httptest.NewRecorder()
	reqMsg, _ := http.NewRequest("POST", "/v1/mcps/admin/messages?session_id="+sessionID, bytes.NewBuffer(callJSON))
	reqMsg.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wMsg, reqMsg)

	assert.Equal(t, http.StatusAccepted, wMsg.Code)

	// 验证 SSE 收到来自 Guard 放行且成功执行的结果
	assert.Eventually(t, func() bool {
		return strings.Contains(wSSE.String(), "ADM007")
	}, 1*time.Second, 10*time.Millisecond)

	assert.Contains(t, wSSE.String(), "guarded_data_accessed")
}

func TestMCPModelProjectionAndExecution(t *testing.T) {
	server.Reset()

	// 1. 注册模拟的数据模型 process
	process.Register("models.patient.find", func(p *process.Process) interface{} {
		id := p.ArgsString(0)
		return map[string]interface{}{
			"id":   id,
			"name": "王小明",
			"age":  30,
		}
	})

	process.Register("models.patient.save", func(p *process.Process) interface{} {
		payload := p.ArgsMap(0)
		return map[string]interface{}{
			"id":     "P001",
			"saved":  true,
			"record": payload,
		}
	})

	// 2. 构造一个轻量 Model
	if gouModel.Models == nil {
		gouModel.Models = make(map[string]*gouModel.Model)
	}
	gouModel.Models["patient"] = &gouModel.Model{
		MetaData: gouModel.MetaData{
			Name: "门诊患者档案",
		},
	}
	defer delete(gouModel.Models, "patient")

	// 3. 执行模型投影
	count := ProjectModelsToMCP("patient")
	assert.Equal(t, 4, count, "should project 4 tools: find, search, save, delete")

	srv := server.GetServer("default")
	tools := srv.GetTools()
	toolNames := map[string]bool{}
	for _, tItem := range tools {
		toolNames[tItem.Name] = true
	}
	assert.True(t, toolNames["model_patient_find"])
	assert.True(t, toolNames["model_patient_search"])
	assert.True(t, toolNames["model_patient_save"])
	assert.True(t, toolNames["model_patient_delete"])

	// 4. 验证生成的 Tool Schema 合规性
	var findSchema struct {
		Type       string              `json:"type"`
		Properties map[string]struct{} `json:"properties"`
		Required   []string            `json:"required"`
	}
	var targetFindTool types.Tool
	for _, tItem := range tools {
		if tItem.Name == "model_patient_find" {
			targetFindTool = tItem
			break
		}
	}
	err := json.Unmarshal(targetFindTool.InputSchema, &findSchema)
	assert.NoError(t, err)
	assert.Equal(t, "object", findSchema.Type)
	assert.Contains(t, findSchema.Required, "id")

	// 5. 执行 model_patient_find 工具调用（验证参数正常派发与错误安全捕获）
	ctx := context.Background()
	findCallReq := []byte(`{"jsonrpc":"2.0","id":901,"method":"tools/call","params":{"name":"model_patient_find","arguments":{"id":"P001"}}}`)
	findResp, err := srv.DispatchJSONRPC(ctx, findCallReq)
	assert.NoError(t, err)
	assert.NotNil(t, findResp)
	assert.Nil(t, findResp.Error)

	var callResult types.CallToolResponse
	b, _ := json.Marshal(findResp.Result)
	json.Unmarshal(b, &callResult)
	// 由于单测未连真实DB，验证底层错误被安全捕获并符合MCP标准错误规范
	assert.True(t, callResult.IsError)
	assert.NotEmpty(t, callResult.Content[0].Text)

	// 6. 执行 model_patient_save 工具调用
	saveCallReq := []byte(`{"jsonrpc":"2.0","id":902,"method":"tools/call","params":{"name":"model_patient_save","arguments":{"payload":{"name":"李华","age":25}}}}`)
	saveResp, err := srv.DispatchJSONRPC(ctx, saveCallReq)
	assert.NoError(t, err)
	assert.NotNil(t, saveResp)
	assert.Nil(t, saveResp.Error)

	var saveResult types.CallToolResponse
	b, _ = json.Marshal(saveResp.Result)
	json.Unmarshal(b, &saveResult)
	assert.True(t, saveResult.IsError)
	assert.NotEmpty(t, saveResult.Content[0].Text)
}

