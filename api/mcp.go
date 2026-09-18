package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	gouApi "github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/mcp/server"
	"github.com/yaoapp/gou/mcp/types"
	gouModel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// ProjectAllToMCP 统一扫描投影 APIs 与 Models 到 MCP Tools
func ProjectAllToMCP(models ...string) int {
	count := ProjectAPIsToMCP()
	count += ProjectModelsToMCP(models...)
	return count
}

// ProjectModelsToMCP 扫描已载入的 Model，并投影为 MCP CRUD 工具（find, search, save, delete）
func ProjectModelsToMCP(modelIDs ...string) int {
	filter := make(map[string]bool)
	for _, m := range modelIDs {
		filter[m] = true
	}

	count := 0
	srv := server.GetServer("default")

	for id, mod := range gouModel.Models {
		if len(filter) > 0 && !filter[id] {
			continue
		}

		cleanID := strings.ReplaceAll(id, ".", "_")
		desc := mod.MetaData.Name
		if desc == "" {
			desc = id
		}

		// 1. find: 根据 ID 查询
		findTool := types.Tool{
			Name:        fmt.Sprintf("model_%s_find", cleanID),
			Description: fmt.Sprintf("Query single record of %s by primary ID", desc),
			InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Record ID"}},"required":["id"]}`),
		}
		targetModel := id
		srv.RegisterTool(findTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			idVal := args["id"]
			p, err := process.Of(fmt.Sprintf("models.%s.find", targetModel), idVal, map[string]interface{}{})
			if err != nil {
				return nil, err
			}
			p.WithContext(ctx)
			if err := p.Execute(); err != nil {
				return nil, err
			}
			return p.Value(), nil
		})
		count++

		// 2. search: 分页搜索
		searchTool := types.Tool{
			Name:        fmt.Sprintf("model_%s_search", cleanID),
			Description: fmt.Sprintf("Search and paginate records of %s", desc),
			InputSchema: json.RawMessage(`{"type":"object","properties":{"page":{"type":"integer","description":"Page number (default 1)"},"pagesize":{"type":"integer","description":"Page size (default 20)"},"where":{"type":"object","description":"Query conditions"}},"required":[]}`),
		}
		srv.RegisterTool(searchTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			page := 1
			pagesize := 20
			if pVal, ok := args["page"].(float64); ok && pVal > 0 {
				page = int(pVal)
			}
			if psVal, ok := args["pagesize"].(float64); ok && psVal > 0 {
				pagesize = int(psVal)
			}
			param := map[string]interface{}{}
			if wVal, ok := args["where"].(map[string]interface{}); ok {
				param["where"] = wVal
			}

			p, err := process.Of(fmt.Sprintf("models.%s.paginate", targetModel), param, page, pagesize)
			if err != nil {
				return nil, err
			}
			p.WithContext(ctx)
			if err := p.Execute(); err != nil {
				return nil, err
			}
			return p.Value(), nil
		})
		count++

		// 3. save: 保存/更新记录
		saveTool := types.Tool{
			Name:        fmt.Sprintf("model_%s_save", cleanID),
			Description: fmt.Sprintf("Create or update record of %s", desc),
			InputSchema: json.RawMessage(`{"type":"object","properties":{"payload":{"type":"object","description":"Record data payload"}},"required":["payload"]}`),
		}
		srv.RegisterTool(saveTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			payload := args["payload"]
			if payload == nil {
				payload = args
			}
			p, err := process.Of(fmt.Sprintf("models.%s.save", targetModel), payload)
			if err != nil {
				return nil, err
			}
			p.WithContext(ctx)
			if err := p.Execute(); err != nil {
				return nil, err
			}
			return p.Value(), nil
		})
		count++

		// 4. delete: 删除记录
		delTool := types.Tool{
			Name:        fmt.Sprintf("model_%s_delete", cleanID),
			Description: fmt.Sprintf("Delete record of %s by primary ID", desc),
			InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Record ID to delete"}},"required":["id"]}`),
		}
		srv.RegisterTool(delTool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			idVal := args["id"]
			p, err := process.Of(fmt.Sprintf("models.%s.delete", targetModel), idVal)
			if err != nil {
				return nil, err
			}
			p.WithContext(ctx)
			if err := p.Execute(); err != nil {
				return nil, err
			}
			return p.Value(), nil
		})
		count++
	}

	return count
}

// ProjectAPIsToMCP 扫描已载入的 API，并将所有声明了 mcp 的路由投影注册为 MCP Tools
func ProjectAPIsToMCP() int {
	count := 0
	for apiID, apiObj := range gouApi.APIs {
		for _, path := range apiObj.HTTP.Paths {
			if path.MCP == nil || !path.MCP.Enable {
				continue
			}

			group := path.MCP.Group
			if group == "" {
				group = "default"
			}

			srv := server.GetServer(group)
			tool := buildToolFromPath(apiID, path)

			// 提取端点关联的 Guard（优先 path.Guard，缺省使用 api.Guard）
			guard := path.Guard
			if guard == "" {
				guard = apiObj.HTTP.Guard
			}

			// 捕获循环变量
			targetPath := path
			targetGuard := guard
			srv.RegisterTool(tool, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				// 1. 先通过 Auth Adapter 执行 API Guard 鉴权
				guardSID, guardAuth, err := executeGuards(ctx, targetGuard)
				if err != nil {
					return nil, err
				}

				procArgs := bindProcessArgs(targetPath.In, args)
				p, err := process.Of(targetPath.Process, procArgs...)
				if err != nil {
					return nil, fmt.Errorf("failed to create process %s: %w", targetPath.Process, err)
				}
				defer p.Dispose()

				// 上下文与 Session 透传
				p.WithContext(ctx)

				// 优先使用 Guard 提取的 SID，其次使用 args 中的 __sid
				finalSID := guardSID
				if finalSID == "" {
					if sid, ok := args["__sid"].(string); ok && sid != "" {
						finalSID = sid
					}
				}
				if finalSID != "" {
					p.WithSID(finalSID)
				}

				// 优先使用 Guard 提取的 authorized，其次使用 args 中的 __authorized
				finalAuth := guardAuth
				if finalAuth == nil {
					if auth, ok := args["__authorized"].(map[string]interface{}); ok {
						finalAuth = auth
					}
				}
				if finalAuth != nil {
					global := map[string]interface{}{"__authorized": finalAuth}
					p.WithGlobal(global)
				}

				err = p.Execute()
				if err != nil {
					return nil, err
				}
				return p.Value(), nil
			})

			log.Trace("[MCP] Projected API %s %s to MCP Tool '%s' in group '%s'", path.Method, path.Path, tool.Name, group)
			count++
		}
	}
	return count
}

// MountMCPRoutes 挂载 MCP 协议端点到 Gin 路由树上
func MountMCPRoutes(router *gin.Engine, root ...string) {
	// 先执行一次投影
	ProjectAPIsToMCP()

	base := "/v1/mcps"
	if len(root) > 0 && root[0] != "" {
		base = strings.TrimRight(root[0], "/")
	}

	// 1. 分组路由: /v1/mcps/:group/*
	groupRouter := router.Group(base + "/:group")
	{
		groupRouter.GET("/sse", func(c *gin.Context) {
			group := c.Param("group")
			server.GetServer(group).SSEHandler(c)
		})

		groupRouter.POST("/messages", func(c *gin.Context) {
			group := c.Param("group")
			server.GetServer(group).MessagesHandler(c)
		})

		groupRouter.POST("", func(c *gin.Context) {
			group := c.Param("group")
			server.GetServer(group).DirectHandler(c)
		})
	}

	// 2. 默认全局路由: /v1/mcps/* (映射到 default server)
	defaultRouter := router.Group(base)
	{
		defaultRouter.GET("/sse", func(c *gin.Context) {
			server.GetServer("default").SSEHandler(c)
		})

		defaultRouter.POST("/messages", func(c *gin.Context) {
			server.GetServer("default").MessagesHandler(c)
		})

		defaultRouter.POST("", func(c *gin.Context) {
			server.GetServer("default").DirectHandler(c)
		})
	}
}

// buildToolFromPath 根据 Path 配置构造具有深度与准确 Schema 的 MCP Tool
func buildToolFromPath(apiID string, path gouApi.Path) types.Tool {
	toolName := path.MCP.Name
	if toolName == "" {
		toolName = formatMCPToolName(apiID, path.Path, path.Method)
	}

	desc := path.MCP.Description
	if desc == "" {
		desc = path.Description
	}
	if desc == "" {
		desc = path.Label
	}
	if desc == "" {
		desc = fmt.Sprintf("API endpoint: %s %s", path.Method, path.Path)
	}

	inputSchemaObj := map[string]interface{}{
		"type": "object",
	}

	properties := map[string]interface{}{}
	required := []string{}

	// 1. 优先使用显式声明的 mcp.params
	if len(path.MCP.Params) > 0 {
		for pName, pParam := range path.MCP.Params {
			prop := map[string]interface{}{
				"type": pParam.Type,
			}
			if pParam.Description != "" {
				prop["description"] = pParam.Description
			}
			if len(pParam.Enum) > 0 {
				prop["enum"] = pParam.Enum
			}
			properties[pName] = prop
			if pParam.Required {
				required = append(required, pName)
			}
		}
	} else {
		// 2. 智能 Schema 合成器：若未显式指定 params，从 path.In 语义推导
		synthesizedProps, synthesizedReq := synthesizeSchemaFromIn(path.In)
		for k, v := range synthesizedProps {
			properties[k] = v
		}
		required = append(required, synthesizedReq...)
	}

	inputSchemaObj["properties"] = properties
	if len(required) > 0 {
		inputSchemaObj["required"] = required
	}

	schemaBytes, _ := json.Marshal(inputSchemaObj)

	return types.Tool{
		Name:        toolName,
		Description: desc,
		InputSchema: schemaBytes,
	}
}

// synthesizeSchemaFromIn 从 API 的 In 数组推导基础 Schema 属性
func synthesizeSchemaFromIn(in []interface{}) (map[string]interface{}, []string) {
	props := map[string]interface{}{}
	required := []string{}

	for _, item := range in {
		str, ok := item.(string)
		if !ok {
			continue
		}

		// $param.id -> 提取 id
		if strings.HasPrefix(str, "$param.") || strings.HasPrefix(str, ":param.") {
			name := strings.TrimPrefix(str, "$param.")
			name = strings.TrimPrefix(name, ":param.")
			props[name] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Path parameter: %s", name),
			}
			required = append(required, name)
		} else if strings.HasPrefix(str, "$query.") || strings.HasPrefix(str, ":query.") {
			name := strings.TrimPrefix(str, "$query.")
			name = strings.TrimPrefix(name, ":query.")
			props[name] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Query parameter: %s", name),
			}
		} else if str == ":payload" || str == "$payload" {
			props["payload"] = map[string]interface{}{
				"type":        "object",
				"description": "Request body payload",
			}
		}
	}

	return props, required
}

func formatMCPToolName(id, pathStr, method string) string {
	clean := strings.Trim(pathStr, "/")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "-", "_")
	clean = strings.ReplaceAll(clean, ":", "")
	clean = strings.ReplaceAll(clean, ".", "_")
	if clean == "" {
		clean = strings.ToLower(method)
	}
	cleanID := strings.ReplaceAll(id, ".", "_")
	cleanID = strings.ReplaceAll(cleanID, "/", "_")
	if cleanID != "" && !strings.HasPrefix(clean, cleanID) {
		return fmt.Sprintf("%s_%s", cleanID, clean)
	}
	return clean
}

// bindProcessArgs 将 MCP 传入的 JSON 参数绑定为 Process 执行入参
func bindProcessArgs(in []interface{}, args map[string]interface{}) []interface{} {
	if len(in) == 0 {
		if len(args) > 0 {
			return []interface{}{args}
		}
		return []interface{}{}
	}

	res := make([]interface{}, len(in))
	for i, paramDef := range in {
		paramStr, ok := paramDef.(string)
		if !ok {
			res[i] = paramDef
			continue
		}

		// 处理 :payload 或 $payload
		if paramStr == ":payload" || paramStr == "$payload" {
			// 若 args 中有名为 "payload" 的嵌套对象则优先提取，否则整包作为 payload
			if nestedPayload, has := args["payload"]; has {
				res[i] = nestedPayload
			} else {
				res[i] = args
			}
			continue
		}

		cleanKey := strings.TrimPrefix(paramStr, ":")
		cleanKey = strings.TrimPrefix(cleanKey, "$")
		parts := strings.Split(cleanKey, ".")
		if len(parts) == 2 {
			key := parts[1]
			if val, has := args[key]; has {
				res[i] = val
				continue
			}
		}

		if val, has := args[cleanKey]; has {
			res[i] = val
			continue
		}

		res[i] = nil
	}
	return res
}

// executeGuards 执行 API 声明的 Guard 链
// 返回:
//   sid: 鉴权产生的会话 ID
//   authorized: 鉴权产生的全局安全上下文对象
//   err: 若鉴权失败被拦截返回对应错误
func executeGuards(ctx context.Context, guardStr string) (string, map[string]interface{}, error) {
	if guardStr == "" || guardStr == "-" {
		return "", nil, nil
	}

	reqInfo, ok := server.GetRequestInfo(ctx)
	if !ok || reqInfo == nil {
		reqInfo = &server.RequestInfo{
			Header: make(http.Header),
			Query:  make(map[string][]string),
		}
	}

	// 构造轻量 gin.Context 用于运行 Guard
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", reqInfo.Path, nil)
	if err != nil {
		req, _ = http.NewRequest("POST", "/", nil)
	}
	req.Header = reqInfo.Header.Clone()
	if len(reqInfo.Query) > 0 {
		q := req.URL.Query()
		for k, vs := range reqInfo.Query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		req.URL.RawQuery = q.Encode()
	}
	c.Request = req

	guards := strings.Split(guardStr, ",")
	for _, name := range guards {
		name = strings.TrimSpace(name)
		if name == "" || name == "-" {
			continue
		}

		handler, has := gouApi.GetGuard(name)
		if !has {
			handler = gouApi.ProcessGuard(name)
		}

		var guardPanic error
		func() {
			defer func() {
				if r := recover(); r != nil {
					guardPanic = fmt.Errorf("guard '%s' panic: %v", name, r)
					c.Abort()
				}
			}()
			handler(c)
		}()

		if guardPanic != nil {
			return "", nil, guardPanic
		}

		if c.IsAborted() {
			errMsg := fmt.Sprintf("Unauthorized: access denied by guard '%s'", name)
			if w.Body.Len() > 0 {
				errMsg = fmt.Sprintf("Unauthorized: %s", strings.TrimSpace(w.Body.String()))
			}
			return "", nil, errors.New(errMsg)
		}
	}

	var sid string
	var authorized map[string]interface{}

	if sidVal, has := c.Get("__sid"); has {
		if sidStr, ok := sidVal.(string); ok && sidStr != "" {
			sid = sidStr
		}
	}

	if authVal, has := c.Get("__authorized"); has {
		if authMap, ok := authVal.(map[string]interface{}); ok {
			authorized = authMap
		}
	} else if globalVal, has := c.Get("__global"); has {
		if globalMap, ok := globalVal.(map[string]interface{}); ok {
			authorized = globalMap
		}
	}

	return sid, authorized, nil
}
