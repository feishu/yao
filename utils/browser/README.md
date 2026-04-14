# utils.browser

`utils.browser` 提供基于浏览器的 HTML 渲染能力，首版包含两个进程：

- `utils.browser.pdf`
- `utils.browser.png`

## 设计边界

- 只支持 HTML 字符串输入
- 不支持 URL 渲染
- 不支持本地 HTML 文件路径输入
- 支持 `base64` 和 `file` 两种输出模式
- 默认以浅色模式渲染，未显式指定背景色时使用白色背景
- `file` 模式下：
  - 绝对路径写入本地文件系统
  - 非绝对路径写入 `data` 文件系统

## 运行时前提

- 运行环境需要可用的 Chrome / Chromium
- 浏览器查找顺序：
  1. `YAO_BROWSER_BIN`
  2. `GOOGLE_CHROME_BIN`
  3. `CHROME_BIN`
  4. `CHROMIUM_BIN`
  5. 系统已安装浏览器
  6. `rod` 默认启动器兜底

## PDF 示例

```javascript
const pdf = Process("utils.browser.pdf", `
  <html>
    <body>
      <h1>Hello Yao</h1>
      <p>Render to PDF</p>
    </body>
  </html>
`, {
  output: "base64",
  print_background: true,
  timeout: 10000
});
```

## PNG 示例

```javascript
const result = Process("utils.browser.png", `
  <html>
    <body style="margin:0;padding:40px;font-family:sans-serif;">
      <h1>Hello Yao</h1>
      <p>Render to PNG</p>
    </body>
  </html>
`, {
  output: "file",
  filename: "renders/hello.png",
  width: 1280,
  height: 720,
  full_page: true
});
```

返回值示例：

```json
{
  "filename": "renders/hello.png",
  "content_type": "image/png",
  "size": 12345
}
```

## 样式与资源

- 支持标准 HTML/CSS
- 支持在 `<head>` 中使用 `<style>` 定义自定义 class
- 支持 `<link rel="stylesheet">` 引用外部样式表
- 支持 Tailwind，但更推荐传入已编译好的 CSS，而不是依赖运行时脚本动态生成
- 图片 `src` 支持三种常见方式：
  - `data:image/png;base64,...`
  - 可访问的绝对 URL
  - 相对路径，配合 `base_url` 解析
