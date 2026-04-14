package browser

// Result 浏览器渲染结果
type Result struct {
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int    `json:"size,omitempty"`
}

// Options 浏览器渲染选项
type Options struct {
	Output string `json:"output,omitempty"`

	Filename string `json:"filename,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`

	Timeout int `json:"timeout,omitempty"`
	Wait    int `json:"wait,omitempty"`
	Width   int `json:"width,omitempty"`
	Height  int `json:"height,omitempty"`

	Scale           float64 `json:"scale,omitempty"`
	Landscape       bool    `json:"landscape,omitempty"`
	PrintBackground bool    `json:"print_background,omitempty"`
	FullPage        bool    `json:"full_page,omitempty"`
}
