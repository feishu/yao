package dbadmin

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xun/capsule"
)

// cleanSQL 提取并清洗 SQL 语句的首个关键词（过滤注释与空白）
func cleanSQL(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	lines := strings.Split(trimmed, "\n")
	cleanLines := make([]string, 0, len(lines))
	for _, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "--") || strings.HasPrefix(lTrim, "#") {
			continue
		}
		cleanLines = append(cleanLines, l)
	}
	clean := strings.TrimSpace(strings.Join(cleanLines, "\n"))
	// 移除块级注释 /* ... */
	for strings.HasPrefix(clean, "/*") {
		endIdx := strings.Index(clean, "*/")
		if endIdx == -1 {
			break
		}
		clean = strings.TrimSpace(clean[endIdx+2:])
	}

	fields := strings.Fields(clean)
	firstWord := ""
	if len(fields) > 0 {
		firstWord = strings.ToUpper(fields[0])
	}
	return clean, firstWord
}

// sanitizeRow 处理行数据中的 []byte 为字符串，避免 JSON 变成 base64
func sanitizeRow(row map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(row))
	for k, v := range row {
		if b, ok := v.([]byte); ok {
			clean[k] = string(b)
		} else {
			clean[k] = v
		}
	}
	return clean
}

// sanitizeRows 批量处理数据行
func sanitizeRows(rows []map[string]interface{}) []map[string]interface{} {
	for i := range rows {
		rows[i] = sanitizeRow(rows[i])
	}
	return rows
}

// handleStatus 获取数据库连接状态
func handleStatus(c *gin.Context) {
	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未初始化"})
		return
	}

	primary, err := capsule.Global.Primary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"driver":    primary.Config.Driver,
		"name":      primary.Config.Name,
		"readonly":  primary.Config.ReadOnly,
	})
}

// handleGetTables 获取所有数据库表
func handleGetTables(c *gin.Context) {
	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未初始化"})
		return
	}

	schema := capsule.Schema()
	tableNames, err := schema.GetTables()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取表列表失败: %v", err)})
		return
	}

	result := make([]TableInfo, 0, len(tableNames))
	for _, name := range tableNames {
		result = append(result, TableInfo{
			Name: name,
			Type: "BASE TABLE",
		})
	}

	c.JSON(http.StatusOK, result)
}

// getTableColumnsFromDB 优先从 Schema 获取有序字段列表，若失败则通过零行查询直接读取底层驱动返回的物理字段列表
func getTableColumnsFromDB(tableName string) []string {
	columns := make([]string, 0)
	if capsule.Global == nil {
		return columns
	}

	// 1. 优先尝试从 Schema 获取（按字段定义顺序）
	schema := capsule.Schema()
	if schema != nil {
		if tbl, err := schema.GetTable(tableName); err == nil && tbl != nil {
			for _, colName := range tbl.GetColumnNames() {
				columns = append(columns, colName)
			}
			if len(columns) > 0 {
				return columns
			}
		}
	}

	// 2. 兜底策略：通过原生 DB 执行 SELECT * FROM <table> WHERE 1=0 获取物理列列表（0行查询，速度 <1ms）
	cols, _ := getColumnsFromZeroRowQuery(tableName)
	for _, col := range cols {
		columns = append(columns, col.Name)
	}

	return columns
}

// getColumnsFromZeroRowQuery 通过 SELECT * FROM <table> WHERE 1=0 获取列结构（字段名、数据类型、是否可为空）
func getColumnsFromZeroRowQuery(tableName string) ([]ColumnInfo, []string) {
	columns := make([]ColumnInfo, 0)
	primaryKeys := make([]string, 0)

	if capsule.Global == nil {
		return columns, primaryKeys
	}

	primary, err := capsule.Global.Primary()
	if err != nil || primary == nil {
		return columns, primaryKeys
	}

	driver := strings.ToLower(primary.Config.Driver)
	queries := []string{
		fmt.Sprintf("SELECT * FROM %s WHERE 1=0", tableName),
	}
	if strings.Contains(driver, "mysql") {
		queries = []string{
			fmt.Sprintf("SELECT * FROM `%s` WHERE 1=0", tableName),
			fmt.Sprintf("SELECT * FROM %s WHERE 1=0", tableName),
		}
	} else if strings.Contains(driver, "dm") || strings.Contains(driver, "oracle") {
		queries = []string{
			fmt.Sprintf("SELECT * FROM \"%s\" WHERE 1=0", tableName),
			fmt.Sprintf("SELECT * FROM \"%s\" WHERE 1=0", strings.ToUpper(tableName)),
			fmt.Sprintf("SELECT * FROM %s WHERE 1=0", tableName),
			fmt.Sprintf("SELECT * FROM %s WHERE 1=0", strings.ToUpper(tableName)),
		}
	}

	var rows *sql.Rows
	for _, q := range queries {
		r, err := primary.DB.Query(q)
		if err == nil && r != nil {
			rows = r
			break
		}
	}

	if rows == nil {
		return columns, primaryKeys
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err == nil && len(colTypes) > 0 {
		for _, ct := range colTypes {
			name := ct.Name()
			typeName := ct.DatabaseTypeName()
			if typeName == "" {
				typeName = "VARCHAR"
			}
			isNullable := true
			if nullable, ok := ct.Nullable(); ok {
				isNullable = nullable
			}

			isPK := strings.EqualFold(name, "id") || strings.EqualFold(name, "ID") || strings.EqualFold(name, tableName+"_id")
			if isPK {
				primaryKeys = append(primaryKeys, name)
			}

			columns = append(columns, ColumnInfo{
				Name:         name,
				Type:         typeName,
				IsNullable:   isNullable,
				IsPrimaryKey: isPK,
			})
		}
	} else {
		// 降级使用 Columns()
		colNames, err := rows.Columns()
		if err == nil {
			for _, name := range colNames {
				isPK := strings.EqualFold(name, "id") || strings.EqualFold(name, "ID")
				if isPK {
					primaryKeys = append(primaryKeys, name)
				}
				columns = append(columns, ColumnInfo{
					Name:         name,
					Type:         "VARCHAR",
					IsNullable:   true,
					IsPrimaryKey: isPK,
				})
			}
		}
	}

	if len(primaryKeys) == 0 && len(columns) > 0 {
		primaryKeys = append(primaryKeys, columns[0].Name)
	}

	return columns, primaryKeys
}

// handleGetTableSchema 获取指定表的字段与索引结构
func handleGetTableSchema(c *gin.Context) {
	tableName := c.Param("name")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表名不能为空"})
		return
	}

	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未初始化"})
		return
	}

	schema := capsule.Schema()
	has, err := schema.HasTable(tableName)
	if err != nil || !has {
		// 达梦或者特殊数据库大小写敏感时可能 HasTable 判定失败，尝试零行查询检测
		fallbackCols, fallbackPKs := getColumnsFromZeroRowQuery(tableName)
		if len(fallbackCols) > 0 {
			c.JSON(http.StatusOK, TableSchema{
				Name:        tableName,
				PrimaryKeys: fallbackPKs,
				Columns:     fallbackCols,
				Indexes:     []IndexInfo{},
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("表 %s 不存在", tableName)})
		return
	}

	dbalTable, err := schema.GetTable(tableName)
	if err != nil || dbalTable == nil {
		// 降级兜底：通过底层驱动获取 ColumnTypes 构建 Schema
		fallbackCols, fallbackPKs := getColumnsFromZeroRowQuery(tableName)
		if len(fallbackCols) > 0 {
			c.JSON(http.StatusOK, TableSchema{
				Name:        tableName,
				PrimaryKeys: fallbackPKs,
				Columns:     fallbackCols,
				Indexes:     []IndexInfo{},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取表结构失败: %v", err)})
		return
	}

	blueprint := dbalTable
	columns := make([]ColumnInfo, 0)
	primaryKeys := make([]string, 0)

	colNames := blueprint.GetColumnNames()
	for _, colName := range colNames {
		col := blueprint.GetColumn(colName)
		if col == nil || col.Column == nil {
			continue
		}

		isPK := col.Column.Primary
		if isPK {
			primaryKeys = append(primaryKeys, col.Column.Name)
		}

		var defVal *string
		if col.Column.DefaultRaw != "" {
			v := col.Column.DefaultRaw
			defVal = &v
		}

		comment := ""
		if col.Column.Comment != nil {
			comment = *col.Column.Comment
		}

		extra := ""
		if col.Column.Extra != nil {
			extra = *col.Column.Extra
		}

		columns = append(columns, ColumnInfo{
			Name:         col.Column.Name,
			Type:         col.Column.Type,
			IsNullable:   col.Column.Nullable,
			DefaultValue: defVal,
			IsPrimaryKey: isPK,
			Comment:      comment,
			Extra:        extra,
		})
	}

	// 备选主键检查
	if len(primaryKeys) == 0 {
		if primary := blueprint.GetPrimary(); primary != nil {
			for _, col := range primary.Columns {
				primaryKeys = append(primaryKeys, col.Name)
			}
		}
	}

	// 若从 Schema 没有解析出字段，降级使用零行查询
	if len(columns) == 0 {
		fallbackCols, fallbackPKs := getColumnsFromZeroRowQuery(tableName)
		if len(fallbackCols) > 0 {
			columns = fallbackCols
			if len(primaryKeys) == 0 {
				primaryKeys = fallbackPKs
			}
		}
	}

	indexes := make([]IndexInfo, 0)
	indexNames := blueprint.GetIndexNames()
	for _, idxName := range indexNames {
		idx := blueprint.GetIndex(idxName)
		if idx == nil {
			continue
		}
		colList := make([]string, 0)
		if idx.ColumnName != "" {
			colList = append(colList, idx.ColumnName)
		}
		indexes = append(indexes, IndexInfo{
			Name:     idx.Name,
			Columns:  colList,
			IsUnique: idx.Unique,
		})
	}

	c.JSON(http.StatusOK, TableSchema{
		Name:        tableName,
		PrimaryKeys: primaryKeys,
		Columns:     columns,
		Indexes:     indexes,
	})
}

// handleGetTableData 分页查询数据表内容
func handleGetTableData(c *gin.Context) {
	tableName := c.Param("name")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表名不能为空"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 50
	}

	sortField := c.Query("sort_field")
	sortOrder := strings.ToLower(c.Query("sort_order"))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	filterField := c.Query("filter_field")
	filterOp := c.Query("filter_op")
	filterValue := c.Query("filter_value")

	qb := capsule.Query().Table(tableName)

	if filterField != "" && filterValue != "" {
		switch filterOp {
		case "like":
			qb.Where(filterField, "like", "%"+filterValue+"%")
		case "!=", "<>", ">", ">=", "<", "<=":
			qb.Where(filterField, filterOp, filterValue)
		default:
			qb.Where(filterField, filterValue)
		}
	}

	total, err := qb.Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询总数失败: %v", err)})
		return
	}

	if sortField != "" {
		qb.OrderBy(sortField, sortOrder)
	}

	offset := (page - 1) * pageSize
	rows, err := qb.Offset(offset).Limit(pageSize).Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询数据失败: %v", err)})
		return
	}

	// 转换为通用的 []map[string]interface{}
	data := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		data = append(data, sanitizeRow(r))
	}

	// 无论是否有数据，必须获取按顺序排列的列名列表！
	columns := getTableColumnsFromDB(tableName)
	if len(columns) == 0 && len(data) > 0 {
		for k := range data[0] {
			columns = append(columns, k)
		}
	}

	c.JSON(http.StatusOK, PaginatedData{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Columns:  columns,
		Data:     data,
	})
}

// handleInsertTableData 插入单条表记录
func handleInsertTableData(c *gin.Context) {
	tableName := c.Param("name")
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据请求格式"})
		return
	}

	if len(payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插入数据不能为空"})
		return
	}

	id, err := capsule.Query().Table(tableName).InsertGetID(payload)
	if err != nil {
		// 回退尝试普通 Insert
		err = capsule.Query().Table(tableName).Insert(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("新增记录失败: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, MutationResponse{
		Success:  true,
		Message:  "插入成功",
		Affected: 1,
		ID:       id,
	})
}

// handleUpdateTableData 根据主键修改记录
func handleUpdateTableData(c *gin.Context) {
	tableName := c.Param("name")
	var payload struct {
		PrimaryKey map[string]interface{} `json:"primary_key"`
		Data       map[string]interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据请求格式"})
		return
	}

	if len(payload.PrimaryKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供主键条件，拒绝执行无主键更新"})
		return
	}

	if len(payload.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供更新内容"})
		return
	}

	qb := capsule.Query().Table(tableName)
	for k, v := range payload.PrimaryKey {
		qb.Where(k, v)
	}

	affected, err := qb.Update(payload.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新记录失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, MutationResponse{
		Success:  true,
		Message:  "更新成功",
		Affected: int64(affected),
	})
}

// handleDeleteTableData 根据主键删除记录
func handleDeleteTableData(c *gin.Context) {
	tableName := c.Param("name")
	var payload struct {
		PrimaryKey map[string]interface{} `json:"primary_key"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据请求格式"})
		return
	}

	if len(payload.PrimaryKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供主键条件，拒绝执行无主键删除"})
		return
	}

	qb := capsule.Query().Table(tableName)
	for k, v := range payload.PrimaryKey {
		qb.Where(k, v)
	}

	affected, err := qb.Delete()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除记录失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, MutationResponse{
		Success:  true,
		Message:  "删除成功",
		Affected: int64(affected),
	})
}

// handleExecuteSQL 核心：执行任意自定义 SQL 查询与命令
func handleExecuteSQL(c *gin.Context) {
	var req SQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体缺少有效 sql 语句"})
		return
	}

	cleanQuery, firstWord := cleanSQL(req.SQL)
	if cleanQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SQL 语句不能为空"})
		return
	}

	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未就绪"})
		return
	}

	primary, err := capsule.Global.Primary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取数据库连接失败: %v", err)})
		return
	}

	db := primary.DB
	start := time.Now()

	// 判断是否是只读查询语句
	isQuery := false
	switch firstWord {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "PRAGMA", "WITH":
		isQuery = true
	}

	if isQuery {
		rows, err := db.Queryx(cleanQuery)
		if err != nil {
			elapsed := time.Since(start).Milliseconds()
			c.JSON(http.StatusOK, SQLResponse{
				Error:         err.Error(),
				TimeElapsedMs: elapsed,
			})
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			elapsed := time.Since(start).Milliseconds()
			c.JSON(http.StatusOK, SQLResponse{
				Error:         fmt.Sprintf("解析结果集列名失败: %v", err),
				TimeElapsedMs: elapsed,
			})
			return
		}

		resultRows := make([]map[string]interface{}, 0)
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				c.JSON(http.StatusOK, SQLResponse{
					Error:         fmt.Sprintf("扫描数据行失败: %v", err),
					TimeElapsedMs: time.Since(start).Milliseconds(),
				})
				return
			}

			row := make(map[string]interface{}, len(columns))
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
				row[col] = val
			}
			resultRows = append(resultRows, row)
		}

		elapsed := time.Since(start).Milliseconds()
		c.JSON(http.StatusOK, SQLResponse{
			Columns:       columns,
			Rows:          resultRows,
			Total:         int64(len(resultRows)),
			TimeElapsedMs: elapsed,
		})
		return
	}

	// 否则为变更类语句 (INSERT, UPDATE, DELETE, DDL 等)
	res, err := db.Exec(cleanQuery)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, SQLResponse{
			Error:         err.Error(),
			TimeElapsedMs: elapsed,
		})
		return
	}

	affected, _ := res.RowsAffected()
	c.JSON(http.StatusOK, SQLResponse{
		Affected:      affected,
		TimeElapsedMs: elapsed,
	})
}

// handleBatchCommit 批量提交新增、修改与删除操作 (Navicat 核心提交交互)
func handleBatchCommit(c *gin.Context) {
	tableName := c.Param("name")
	var req BatchCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式不正确"})
		return
	}

	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未就绪"})
		return
	}

	start := time.Now()
	insertedCount := 0
	updatedCount := 0
	deletedCount := 0

	// 1. 处理 Inserts (新增行)
	for _, row := range req.Inserts {
		if len(row) == 0 {
			continue
		}
		err := capsule.Query().Table(tableName).Insert(row)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("新增记录失败: %v", err)})
			return
		}
		insertedCount++
	}

	// 2. 处理 Updates (修改行)
	for _, up := range req.Updates {
		if len(up.PrimaryKey) == 0 || len(up.Data) == 0 {
			continue
		}
		qb := capsule.Query().Table(tableName)
		for k, v := range up.PrimaryKey {
			qb.Where(k, v)
		}
		_, err := qb.Update(up.Data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新记录失败: %v", err)})
			return
		}
		updatedCount++
	}

	// 3. 处理 Deletes (标记删除的行)
	for _, delPK := range req.Deletes {
		if len(delPK) == 0 {
			continue
		}
		qb := capsule.Query().Table(tableName)
		for k, v := range delPK {
			qb.Where(k, v)
		}
		_, err := qb.Delete()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除记录失败: %v", err)})
			return
		}
		deletedCount++
	}

	elapsed := time.Since(start).Milliseconds()
	c.JSON(http.StatusOK, BatchCommitResponse{
		Success:       true,
		InsertedCount: insertedCount,
		UpdatedCount:  updatedCount,
		DeletedCount:  deletedCount,
		TimeElapsedMs: elapsed,
		Message:       fmt.Sprintf("提交成功: 新增 %d 条, 更新 %d 条, 删除 %d 条 (耗时 %d ms)", insertedCount, updatedCount, deletedCount, elapsed),
	})
}

// handleGetTableDDL 获取表的建表 DDL 语句 (支持 DM、MySQL、SQLite 等全数据库类型)
func handleGetTableDDL(c *gin.Context) {
	tableName := c.Param("name")
	if capsule.Global == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库连接未就绪"})
		return
	}

	primary, err := capsule.Global.Primary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	db := primary.DB
	driver := strings.ToLower(primary.Config.Driver)
	ddl := ""

	// 1. 尝试调用底层数据库的原生 DDL 查看指令
	switch {
	case strings.Contains(driver, "mysql"):
		var tbl, createSQL string
		err := db.QueryRowx(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)).Scan(&tbl, &createSQL)
		if err == nil {
			ddl = createSQL
		}
	case strings.Contains(driver, "sqlite"):
		var sqlText string
		err := db.QueryRowx("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&sqlText)
		if err == nil {
			ddl = sqlText
		}
	case strings.Contains(driver, "dm"): // 达梦数据库
		var createSQL string
		err := db.QueryRowx(fmt.Sprintf("SELECT DBMS_METADATA.GET_DDL('TABLE', '%s') FROM DUAL", strings.ToUpper(tableName))).Scan(&createSQL)
		if err == nil && createSQL != "" {
			ddl = createSQL
		} else {
			_ = db.QueryRowx(fmt.Sprintf("SELECT DBMS_METADATA.GET_DDL('TABLE', '%s') FROM DUAL", tableName)).Scan(&createSQL)
			if createSQL != "" {
				ddl = createSQL
			}
		}
	}

	// 2. 通用兜底：如果原生命令不可用，自动根据解析的字段与主键生成标准 DDL
	if ddl == "" {
		ddl = buildStandardDDL(tableName)
	}

	c.JSON(http.StatusOK, gin.H{
		"name": tableName,
		"ddl":  ddl,
	})
}

// buildStandardDDL 根据表结构元数据自动构建标准的建表 SQL
func buildStandardDDL(tableName string) string {
	schema := capsule.Schema()
	blueprint, err := schema.GetTable(tableName)
	if err != nil || blueprint == nil {
		// 降级使用零行查询构建标准 DDL
		cols, pks := getColumnsFromZeroRowQuery(tableName)
		if len(cols) > 0 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", tableName))
			for i, col := range cols {
				sb.WriteString(fmt.Sprintf("  `%s` %s", col.Name, col.Type))
				if !col.IsNullable {
					sb.WriteString(" NOT NULL")
				}
				if col.DefaultValue != nil {
					sb.WriteString(fmt.Sprintf(" DEFAULT '%s'", *col.DefaultValue))
				}
				if i < len(cols)-1 || len(pks) > 0 {
					sb.WriteString(",\n")
				} else {
					sb.WriteString("\n")
				}
			}
			if len(pks) > 0 {
				pkCols := make([]string, len(pks))
				for k, pk := range pks {
					pkCols[k] = fmt.Sprintf("`%s`", pk)
				}
				sb.WriteString(fmt.Sprintf("  PRIMARY KEY (%s)\n", strings.Join(pkCols, ", ")))
			}
			sb.WriteString(");")
			return sb.String()
		}
		return fmt.Sprintf("-- 无法提取表 %s 的元数据结构: %v", tableName, err)
	}

	colNames := blueprint.GetColumnNames()
	if len(colNames) == 0 {
		return fmt.Sprintf("-- 表 %s 未包含字段定义", tableName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", tableName))

	primaryKeys := make([]string, 0)
	for i, colName := range colNames {
		col := blueprint.GetColumn(colName)
		if col == nil || col.Column == nil {
			continue
		}

		c := col.Column
		if c.Primary {
			primaryKeys = append(primaryKeys, c.Name)
		}

		colType := strings.ToUpper(c.Type)
		if colType == "" {
			colType = "VARCHAR(255)"
		}

		sb.WriteString(fmt.Sprintf("  `%s` %s", c.Name, colType))
		if !c.Nullable {
			sb.WriteString(" NOT NULL")
		}
		if c.DefaultRaw != "" {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", c.DefaultRaw))
		}
		if c.Extra != nil && *c.Extra != "" {
			sb.WriteString(fmt.Sprintf(" %s", *c.Extra))
		}
		if c.Comment != nil && *c.Comment != "" {
			sb.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(*c.Comment, "'", "\\'")))
		}

		if i < len(colNames)-1 {
			sb.WriteString(",\n")
		} else {
			sb.WriteString("\n")
		}
	}

	if len(primaryKeys) == 0 {
		if primary := blueprint.GetPrimary(); primary != nil {
			for _, pCol := range primary.Columns {
				primaryKeys = append(primaryKeys, pCol.Name)
			}
		}
	}

	if len(primaryKeys) > 0 {
		pkCols := make([]string, len(primaryKeys))
		for k, pk := range primaryKeys {
			pkCols[k] = fmt.Sprintf("`%s`", pk)
		}
		sb.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)\n", strings.Join(pkCols, ", ")))
	}

	sb.WriteString(");")
	return sb.String()
}
