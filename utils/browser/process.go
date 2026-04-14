package browser

import (
	"encoding/json"
	"net/url"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// Init 注册浏览器渲染进程
func Init() {
	process.RegisterGroup("utils.browser", map[string]process.Handler{
		"pdf": ProcessPDF,
		"png": ProcessPNG,
	})
}

// ProcessPDF utils.browser.pdf
func ProcessPDF(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	html := process.ArgsString(0)
	options, err := parseOptions(process)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	res, err := RenderPDF(html, options)
	if err != nil {
		code := 500
		if isValidationError(err) {
			code = 400
		}
		exception.New(err.Error(), code).Throw()
	}

	return res
}

// ProcessPNG utils.browser.png
func ProcessPNG(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	html := process.ArgsString(0)
	options, err := parseOptions(process)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	res, err := RenderPNG(html, options)
	if err != nil {
		code := 500
		if isValidationError(err) {
			code = 400
		}
		exception.New(err.Error(), code).Throw()
	}

	return res
}

func parseOptions(process *process.Process) (Options, error) {
	if process.NumOfArgs() <= 1 {
		return Options{}, nil
	}

	data, err := optionsDataOf(process.Args[1])
	if err != nil {
		return Options{}, err
	}

	return loadOptions(data), nil
}

func optionsDataOf(value interface{}) (map[string]interface{}, error) {
	switch value := value.(type) {
	case nil:
		return map[string]interface{}{}, nil
	case string:
		if value == "" {
			return map[string]interface{}{}, nil
		}

		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			return nil, newValidationError("invalid options json: %s", err.Error())
		}
		return data, nil
	case map[string]interface{}:
		return value, nil
	case map[string]string:
		data := map[string]interface{}{}
		for key, item := range value {
			data[key] = item
		}
		return data, nil
	case maps.MapStrAny:
		return map[string]interface{}(value), nil
	case url.Values:
		data := map[string]interface{}{}
		for key, items := range value {
			if len(items) == 1 {
				data[key] = items[0]
				continue
			}
			data[key] = items
		}
		return data, nil
	default:
		return nil, newValidationError("options must be an object or JSON string")
	}
}

func loadOptions(data map[string]interface{}) Options {
	options := Options{}

	if value, ok := data["output"].(string); ok {
		options.Output = value
	}

	if value, ok := data["filename"].(string); ok {
		options.Filename = value
	}

	if value, ok := data["base_url"].(string); ok {
		options.BaseURL = value
	}

	if value, ok := data["timeout"].(float64); ok {
		options.Timeout = int(value)
	}

	if value, ok := data["wait"].(float64); ok {
		options.Wait = int(value)
	}

	if value, ok := data["width"].(float64); ok {
		options.Width = int(value)
	}

	if value, ok := data["height"].(float64); ok {
		options.Height = int(value)
	}

	if value, ok := data["scale"].(float64); ok {
		options.Scale = value
	}

	if value, ok := data["landscape"].(bool); ok {
		options.Landscape = value
	}

	if value, ok := data["print_background"].(bool); ok {
		options.PrintBackground = value
	}

	if value, ok := data["full_page"].(bool); ok {
		options.FullPage = value
	}

	return options
}
