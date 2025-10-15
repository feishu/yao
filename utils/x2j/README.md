# X2J - XML to JSON Conversion Utility

## 概述

`x2j` 模块提供 XML 和 JSON 之间的转换功能，以及对 XML 数据的查询和操作功能。该模块基于 `github.com/clbanning/mxj/v2` 库实现，提供了高性能的 XML 处理能力。

## 功能特性

- **XML 转 JSON**：将 XML 字符串转换为 JSON 格式
- **XML 转 Map**：将 XML 解析为 Go 的 map 结构
- **Map 转 XML**：将 map 结构转换回 XML
- **标签值提取**：根据标签名提取 XML 中的值
- **路径查询**：查找 XML 中特定标签的所有路径
- **值更新**：更新 XML 中指定路径的值

## Process 接口

### 1. utils.x2j.XmlToJson

将 XML 字符串转换为 JSON 格式。

**参数：**
- `args[0]` (string | []byte): XML 字符串或字节数组
- `args[1]` (bool, 可选): 安全编码标志，默认为 false

**返回：** `[]byte` - JSON 字节数组

**示例：**

```javascript
// 基本用法
var json = Process("utils.x2j.XmlToJson", "<root><name>test</name><value>123</value></root>");

// 使用安全编码
var json = Process("utils.x2j.XmlToJson", xmlString, true);
```

**YAO DSL 示例：**

```json
{
  "name": "Convert XML to JSON",
  "process": "utils.x2j.XmlToJson",
  "args": ["<root><user><name>张三</name><age>30</age></user></root>"]
}
```

### 2. utils.x2j.XmlToMap

将 XML 字符串解析为 map 结构，便于在 Go 代码中操作。

**参数：**
- `args[0]` (string | []byte): XML 字符串或字节数组

**返回：** `map[string]interface{}` - XML 的 map 表示

**示例：**

```javascript
var data = Process("utils.x2j.XmlToMap", "<root><name>test</name></root>");
console.log(data.root.name); // 输出: "test"
```

**YAO DSL 示例：**

```json
{
  "name": "Parse XML",
  "process": "utils.x2j.XmlToMap",
  "args": ["<config><host>localhost</host><port>3306</port></config>"]
}
```

### 3. utils.x2j.MapToXml

将 map 结构转换为 XML 字符串。

**参数：**
- `args[0]` (map[string]interface{}): 要转换的 map

**返回：** `[]byte` - XML 字节数组

**示例：**

```javascript
var xmlData = Process("utils.x2j.MapToXml", {
  "root": {
    "user": {
      "name": "张三",
      "age": 30
    }
  }
});
```

**YAO DSL 示例：**

```json
{
  "name": "Create XML",
  "process": "utils.x2j.MapToXml",
  "args": [
    {
      "config": {
        "database": {
          "host": "localhost",
          "port": 3306
        }
      }
    }
  ]
}
```

### 4. utils.x2j.XmlValuesForTag

提取 XML 中所有匹配指定标签名的值。

**参数：**
- `args[0]` (string | []byte): XML 字符串或字节数组
- `args[1]` (string): 标签名
- `args[2...]` (string, 可选): 属性过滤器，格式为 `-attributeName:value`

**返回：** `[]interface{}` - 匹配的值数组

**示例：**

```javascript
var xml = `
<root>
  <person id="1">
    <name>张三</name>
    <age>30</age>
  </person>
  <person id="2">
    <name>李四</name>
    <age>25</age>
  </person>
</root>
`;

// 获取所有 name 值
var names = Process("utils.x2j.XmlValuesForTag", xml, "name");
// 结果: ["张三", "李四"]

// 获取具有特定属性的 person
var person = Process("utils.x2j.XmlValuesForTag", xml, "person", "-id:1");
```

**YAO DSL 示例：**

```json
{
  "name": "Extract Names",
  "process": "utils.x2j.XmlValuesForTag",
  "args": ["{{$xml}}", "name"]
}
```

### 5. utils.x2j.XmlPathsForTag

查找 XML 中所有匹配指定标签的路径。

**参数：**
- `args[0]` (string | []byte): XML 字符串或字节数组
- `args[1]` (string): 标签名

**返回：** `[]string` - 路径数组，使用点分隔符

**示例：**

```javascript
var xml = `
<root>
  <user>
    <name>张三</name>
    <details>
      <name>详细名称</name>
    </details>
  </user>
</root>
`;

var paths = Process("utils.x2j.XmlPathsForTag", xml, "name");
// 结果: ["root.user.name", "root.user.details.name"]
```

**YAO DSL 示例：**

```json
{
  "name": "Find Tag Paths",
  "process": "utils.x2j.XmlPathsForTag",
  "args": ["{{$xml}}", "email"]
}
```

### 6. utils.x2j.XmlUpdateValsForPath

更新 XML 中指定路径的值。

**参数：**
- `args[0]` (string | []byte): XML 字符串或字节数组
- `args[1]` (interface{}): 新值
- `args[2]` (string): 点分隔的路径
- `args[3...]` (string, 可选): 子键过滤器

**返回：** `[]byte` - 更新后的 XML 字节数组

**示例：**

```javascript
var xml = "<root><name>oldValue</name></root>";
var updated = Process("utils.x2j.XmlUpdateValsForPath", xml, "newValue", "root.name");
// 结果: <root><name>newValue</name></root>
```

**YAO DSL 示例：**

```json
{
  "name": "Update Config",
  "process": "utils.x2j.XmlUpdateValsForPath",
  "args": ["{{$xml}}", "newhost.com", "config.host"]
}
```

## 使用场景

### 场景 1: API 数据转换

处理返回 XML 格式的第三方 API：

```javascript
// 调用返回 XML 的 API
var response = http.Get("https://api.example.com/data.xml");

// 转换为 JSON 供前端使用
var jsonData = Process("utils.x2j.XmlToJson", response);

return jsonData;
```

### 场景 2: 配置文件处理

读取和修改 XML 配置文件：

```javascript
// 读取配置文件
var configXml = Process("fs.ReadFile", "config.xml");

// 解析为 map
var config = Process("utils.x2j.XmlToMap", configXml);

// 修改配置
config.database.host = "newhost.com";

// 转换回 XML
var updatedXml = Process("utils.x2j.MapToXml", config);

// 保存
Process("fs.WriteFile", "config.xml", updatedXml);
```

### 场景 3: 数据迁移

从 XML 格式迁移数据到数据库：

```javascript
// 读取 XML 数据
var xmlData = Process("fs.ReadFile", "users.xml");

// 提取所有用户
var users = Process("utils.x2j.XmlValuesForTag", xmlData, "user");

// 批量插入数据库
users.forEach(function(user) {
  Process("models.user.Insert", {
    name: user.name,
    email: user.email,
    age: user.age
  });
});
```

### 场景 4: RSS/Atom Feed 处理

解析 RSS feed：

```javascript
// 获取 RSS feed
var feed = http.Get("https://blog.example.com/rss");

// 转换为 map
var data = Process("utils.x2j.XmlToMap", feed);

// 提取文章标题
var titles = Process("utils.x2j.XmlValuesForTag", feed, "title");

return titles;
```

## 注意事项

1. **XML 格式**: 输入的 XML 必须是格式正确的，否则会抛出异常
2. **大文件处理**: 对于大型 XML 文件，考虑使用流式处理或分块处理
3. **字符编码**: 默认使用 UTF-8 编码，对于其他编码需要先转换
4. **属性处理**: XML 属性在转换为 map 时会以 `-attributeName` 格式表示
5. **命名空间**: 支持 XML 命名空间，但在路径中需要包含命名空间前缀

## 性能优化建议

1. **缓存结果**: 对于频繁访问的 XML 数据，考虑缓存转换结果
2. **批量操作**: 使用 `XmlUpdateValsForPath` 批量更新多个值
3. **选择合适的方法**: 
   - 如果只需要数据访问，使用 `XmlToMap`
   - 如果需要传输或存储，使用 `XmlToJson`
   - 如果需要特定值，直接使用 `XmlValuesForTag`

## 错误处理

所有 process 方法在遇到错误时会抛出异常，包括：

- **400**: 无效的输入参数
- **500**: XML 解析错误或转换失败

建议在调用时进行适当的错误处理：

```javascript
try {
  var data = Process("utils.x2j.XmlToMap", xmlString);
  // 处理数据
} catch (e) {
  log.Error("XML parsing failed: " + e.message);
  // 错误处理逻辑
}
```

## 相关资源

- [mxj 库文档](https://pkg.go.dev/github.com/clbanning/mxj/v2)
- [XML 规范](https://www.w3.org/TR/xml/)
- [JSON 规范](https://www.json.org/)

## 更新日志

### v0.10.4
- 初始版本
- 实现基本的 XML/JSON 转换功能
- 添加 XML 查询和操作功能
- 完整的单元测试覆盖
