package dbadmin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

func setupTestDB(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	os.Setenv("YAO_DB_ADMIN", "true")
	os.Setenv("YAO_DB_ADMIN_AUTH", "false") // 测试免密
	SetEnabled(true)

	// 初始化 xun capsule 内存数据库
	manager := capsule.New()
	_, err := manager.Add("default", "sqlite3", "file::memory:?cache=shared", false)
	assert.NoError(t, err)
	manager.SetAsGlobal()

	// 创建测试表
	_, err = manager.Primary()
	assert.NoError(t, err)

	db := capsule.Global.Pool.Primary[0].DB
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(64) NOT NULL,
			email VARCHAR(128),
			age INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS test_empty_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title VARCHAR(100) NOT NULL,
			description TEXT,
			status INTEGER DEFAULT 0
		);
	`)
	assert.NoError(t, err)

	router := gin.New()
	Mount(router, config.Config{Mode: "development"})
	return router
}

func TestDBAdmin_StatusAndTables(t *testing.T) {
	router := setupTestDB(t)

	// 1. 状态检查
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__yao/db/api/status", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var status map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &status)
	assert.NoError(t, err)
	assert.Equal(t, true, status["connected"])
	assert.Equal(t, "sqlite3", status["driver"])

	// 2. 表列表检查
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db/api/tables", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var tables []TableInfo
	err = json.Unmarshal(w.Body.Bytes(), &tables)
	assert.NoError(t, err)
	hasTestUsers := false
	for _, tab := range tables {
		if tab.Name == "test_users" {
			hasTestUsers = true
			break
		}
	}
	assert.True(t, hasTestUsers, "表列表中应包含 test_users")
}

func TestDBAdmin_SchemaAndCRUD(t *testing.T) {
	router := setupTestDB(t)

	// 1. 获取结构
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__yao/db/api/tables/test_users/schema", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var schema TableSchema
	err := json.Unmarshal(w.Body.Bytes(), &schema)
	assert.NoError(t, err)
	assert.Equal(t, "test_users", schema.Name)
	assert.NotEmpty(t, schema.Columns)

	// 2. 插入单条
	body, _ := json.Marshal(map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   25,
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/__yao/db/api/tables/test_users/data", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var insertRes MutationResponse
	err = json.Unmarshal(w.Body.Bytes(), &insertRes)
	assert.NoError(t, err)
	assert.True(t, insertRes.Success)

	// 3. 分页查询
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db/api/tables/test_users/data?page=1&page_size=10", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var pageData PaginatedData
	err = json.Unmarshal(w.Body.Bytes(), &pageData)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pageData.Total)
	assert.Equal(t, "Alice", pageData.Data[0]["name"])

	// 4. 更新单条
	updateBody, _ := json.Marshal(map[string]interface{}{
		"primary_key": map[string]interface{}{"name": "Alice"},
		"data":        map[string]interface{}{"age": 26},
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/__yao/db/api/tables/test_users/data", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. 删除单条
	delBody, _ := json.Marshal(map[string]interface{}{
		"primary_key": map[string]interface{}{"name": "Alice"},
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/__yao/db/api/tables/test_users/data", bytes.NewBuffer(delBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 再次查询验证删除
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db/api/tables/test_users/data?page=1&page_size=10", nil)
	router.ServeHTTP(w, req)
	err = json.Unmarshal(w.Body.Bytes(), &pageData)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), pageData.Total)
}

func TestDBAdmin_ExecuteSQL(t *testing.T) {
	router := setupTestDB(t)

	// 1. 测试 INSERT SQL
	insertSQL, _ := json.Marshal(SQLRequest{
		SQL: "INSERT INTO test_users (name, email, age) VALUES ('Bob', 'bob@test.com', 30);",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/__yao/db/api/sql/execute", bytes.NewBuffer(insertSQL))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res1 SQLResponse
	err := json.Unmarshal(w.Body.Bytes(), &res1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), res1.Affected)

	// 2. 测试 SELECT SQL
	selectSQL, _ := json.Marshal(SQLRequest{
		SQL: "SELECT id, name, age FROM test_users WHERE name = 'Bob';",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/__yao/db/api/sql/execute", bytes.NewBuffer(selectSQL))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res2 SQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &res2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), res2.Total)
	assert.Contains(t, res2.Columns, "name")
	assert.Equal(t, "Bob", res2.Rows[0]["name"])

	// 3. 测试错误 SQL
	errSQL, _ := json.Marshal(SQLRequest{
		SQL: "SELECT * FROM non_existent_table_999;",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/__yao/db/api/sql/execute", bytes.NewBuffer(errSQL))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res3 SQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &res3)
	assert.NoError(t, err)
	assert.NotEmpty(t, res3.Error)
}

func TestDBAdmin_StaticPage(t *testing.T) {
	router := setupTestDB(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__yao/db", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Yao DB Admin")
}

func TestDBAdmin_BatchCommit(t *testing.T) {
	router := setupTestDB(t)

	// 测试批量原子提交 (新增 2 条，更新 1 条，删除 1 条)
	batchReq := BatchCommitRequest{
		Inserts: []map[string]interface{}{
			{"name": "Charlie", "email": "charlie@test.com", "age": 22},
			{"name": "David", "email": "david@test.com", "age": 28},
		},
		Updates: []RowUpdate{},
		Deletes: []map[string]interface{}{},
	}
	body, _ := json.Marshal(batchReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/__yao/db/api/tables/test_users/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res BatchCommitResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, 2, res.InsertedCount)

	// 验证更新和删除批量处理
	batchReq2 := BatchCommitRequest{
		Inserts: []map[string]interface{}{},
		Updates: []RowUpdate{
			{
				PrimaryKey: map[string]interface{}{"name": "Charlie"},
				Data:       map[string]interface{}{"age": 23},
			},
		},
		Deletes: []map[string]interface{}{
			{"name": "David"},
		},
	}
	body2, _ := json.Marshal(batchReq2)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/__yao/db/api/tables/test_users/batch", bytes.NewBuffer(body2))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res2 BatchCommitResponse
	err = json.Unmarshal(w.Body.Bytes(), &res2)
	assert.NoError(t, err)
	assert.Equal(t, 1, res2.UpdatedCount)
	assert.Equal(t, 1, res2.DeletedCount)
}

func TestDBAdmin_DDL(t *testing.T) {
	router := setupTestDB(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__yao/db/api/tables/test_users/ddl", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Contains(t, res["ddl"], "CREATE TABLE")
}

func TestDBAdmin_EmptyTableHasColumns(t *testing.T) {
	router := setupTestDB(t)

	// 1. 查询空数据表的分页数据，验证 total 为 0 但 columns 依然非空且包含所有字段
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__yao/db/api/tables/test_empty_table/data", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var pageData PaginatedData
	err := json.Unmarshal(w.Body.Bytes(), &pageData)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), pageData.Total)
	assert.Empty(t, pageData.Data)
	assert.NotEmpty(t, pageData.Columns, "即使表中没有数据，也必须返回完整的列名列表")
	assert.Contains(t, pageData.Columns, "title")
	assert.Contains(t, pageData.Columns, "description")
	assert.Contains(t, pageData.Columns, "status")

	// 2. 查询空数据表的 schema
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db/api/tables/test_empty_table/schema", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var schema TableSchema
	err = json.Unmarshal(w.Body.Bytes(), &schema)
	assert.NoError(t, err)
	assert.Equal(t, "test_empty_table", schema.Name)
	assert.NotEmpty(t, schema.Columns)

	// 3. 查询空数据表的 DDL
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db/api/tables/test_empty_table/ddl", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var ddlRes map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &ddlRes)
	assert.NoError(t, err)
	assert.Contains(t, ddlRes["ddl"], "CREATE TABLE")
	assert.Contains(t, ddlRes["ddl"], "test_empty_table")
}

func TestDBAdmin_SubpathMount(t *testing.T) {
	os.Setenv("YAO_DB_ADMIN_ROOT", "/syd")
	defer os.Unsetenv("YAO_DB_ADMIN_ROOT")

	router := gin.New()
	Mount(router, config.Config{Mode: "development"})

	// 1. 测试二级目录静态页面路由
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/syd/__yao/db", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	bodyStr := w.Body.String()
	assert.Contains(t, bodyStr, "window.__YAO_DB_ADMIN_ROOT__ = \"/syd\"")
	assert.Contains(t, bodyStr, "window.__YAO_DB_BASE__")

	// 2. 测试二级目录下 API
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/syd/__yao/db/api/status", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. 同时保持默认 /__yao/db 路由可用
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/__yao/db", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

