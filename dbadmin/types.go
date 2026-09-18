package dbadmin

// TableInfo 数据库表的基本信息
type TableInfo struct {
	Name    string `json:"name"`              // 表名
	Comment string `json:"comment,omitempty"` // 注释
	Rows    int64  `json:"rows,omitempty"`    // 估算行数
	Type    string `json:"type,omitempty"`    // BASE TABLE / VIEW
}

// ColumnInfo 列元数据信息
type ColumnInfo struct {
	Name         string  `json:"name"`                   // 列名
	Type         string  `json:"type"`                   // 数据类型 (如 varchar(255), int, bigint)
	IsNullable   bool    `json:"is_nullable"`            // 是否允许为 NULL
	DefaultValue *string `json:"default_value"`          // 默认值
	IsPrimaryKey bool    `json:"is_primary_key"`         // 是否为主键
	Comment      string  `json:"comment,omitempty"`      // 字段注释
	Extra        string  `json:"extra,omitempty"`        // 额外属性 (如 auto_increment)
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name     string   `json:"name"`      // 索引名称
	Columns  []string `json:"columns"`   // 包含的字段名
	IsUnique bool     `json:"is_unique"` // 是否为唯一索引
}

// TableSchema 表完整结构元数据
type TableSchema struct {
	Name        string       `json:"name"`         // 表名
	Comment     string       `json:"comment"`      // 表注释
	PrimaryKeys []string     `json:"primary_keys"` // 主键列名列表
	Columns     []ColumnInfo `json:"columns"`      // 字段列表
	Indexes     []IndexInfo  `json:"indexes"`      // 索引列表
	DDL         string       `json:"ddl,omitempty"`// 建表 DDL
}

// PaginatedData 分页数据响应
type PaginatedData struct {
	Page     int                      `json:"page"`      // 当前页码
	PageSize int                      `json:"page_size"` // 每页条数
	Total    int64                    `json:"total"`     // 总记录数
	Columns  []string                 `json:"columns"`   // 表头列名
	Data     []map[string]interface{} `json:"data"`      // 记录行列表
}

// SQLRequest SQL 执行请求
type SQLRequest struct {
	SQL   string `json:"sql" binding:"required"` // 需要执行的 SQL 语句
	Limit int    `json:"limit,omitempty"`        // 默认返回最大限制 (防卡死)
}

// SQLResponse SQL 执行响应
type SQLResponse struct {
	Columns       []string                 `json:"columns,omitempty"` // 查询返回的字段名列表
	Rows          []map[string]interface{} `json:"rows,omitempty"`    // 查询返回的数据行列表
	Total         int64                    `json:"total"`             // 查询返回的总行数
	Affected      int64                    `json:"affected"`          // 影响行数
	TimeElapsedMs int64                    `json:"time_elapsed_ms"`   // 执行耗时(毫秒)
	Error         string                   `json:"error,omitempty"`   // 错误信息
}

// BatchCommitRequest 批量事务提交请求 (Navicat 对勾确认核心载荷)
type BatchCommitRequest struct {
	Inserts []map[string]interface{} `json:"inserts"` // 新增行数据列表
	Updates []RowUpdate              `json:"updates"` // 更新行列表
	Deletes []map[string]interface{} `json:"deletes"` // 删除行主键列表
}

// RowUpdate 行更新载荷
type RowUpdate struct {
	PrimaryKey map[string]interface{} `json:"primary_key"` // 原始主键值
	Data       map[string]interface{} `json:"data"`        // 被修改的字段键值对
}

// BatchCommitResponse 批量提交响应
type BatchCommitResponse struct {
	Success       bool   `json:"success"`
	InsertedCount int    `json:"inserted_count"`
	UpdatedCount  int    `json:"updated_count"`
	DeletedCount  int    `json:"deleted_count"`
	TimeElapsedMs int64  `json:"time_elapsed_ms"`
	Message       string `json:"message,omitempty"`
}

// MutationResponse 数据变更统一响应
type MutationResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Affected int64  `json:"affected,omitempty"`
	ID       any    `json:"id,omitempty"`
}
