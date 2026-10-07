package moapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/openai"
)

func init() {
	process.RegisterGroup("moapi", map[string]process.Handler{
		"images.generations": ImagesGenerations,
		"chat.completions":   ChatCompletions,
	})
}

// ImagesGenerations Generate images
func ImagesGenerations(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	model := process.ArgsString(0)
	prompt := process.ArgsString(1)
	option := process.ArgsMap(2, map[string]interface{}{})

	if model == "" {
		exception.New("ImagesGenerations error: model is required", 400).Throw()
	}

	if prompt == "" {
		exception.New("ImagesGenerations error: prompt is required", 400).Throw()
	}

	ai, err := openai.NewMoapi(model)
	if err != nil {
		exception.New("ImagesGenerations error: %s", 400, err).Throw()
	}

	option["model"] = model
	res, ex := ai.ImagesGenerations(prompt, option)
	if ex != nil {
		ex.Throw()
	}
	return res
}

// ChatCompletions chat completions
func ChatCompletions(process *process.Process) interface{} {

	return func(c *gin.Context) {

		option := map[string]interface{}{}
		query := c.Query("payload")
		err := jsoniter.UnmarshalFromString(query, &option)
		if err != nil {
			exception.New("ChatCompletions error: %s", 400, err).Throw()
		}

		// option := payload
		// model := "gpt-3.5-turbo"
		// messages := []map[string]interface{}{
		// 	{
		// 		"role":    "system",
		// 		"content": "You are a helpful assistant.",
		// 	},
		// 	{
		// 		"role":    "user",
		// 		"content": "Hello!",
		// 	},
		// // }

		// option["messages"] = messages
		// option["model"] = model

		delete(option, "context")
		model, ok := option["model"].(string)
		if !ok || model == "" {
			exception.New("ChatCompletions error: model is required", 400).Throw()
		}

		ai, err := openai.NewMoapi(model)
		if err != nil {
			exception.New("ChatCompletions error: %s", 400, err).Throw()
		}

		if v, ok := option["stream"].(bool); ok && v {

			chanStream := make(chan []byte, 32)
			chanError := make(chan error, 1)

			ctx, cancel := context.WithCancel(c.Request.Context())
			defer cancel()

			done := make(chan struct{})
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer func() {
					close(chanStream)
					close(chanError)
					close(done)
					wg.Done()
				}()

				err := ai.Stream(ctx, "/v1/chat/completions", option, func(data []byte) int {
					if (string(data)) == "\n" || string(data) == "" {
						return 1 // HandlerReturnOk
					}

					select {
					case chanStream <- data:
					case <-ctx.Done():
						return 0 // HandlerReturnBreak0
					}

					if strings.HasSuffix(string(data), "[DONE]") {
						return 0 // HandlerReturnBreak0
					}
					return 1 // HandlerReturnOk
				})
				if err != nil {
					select {
					case chanError <- errors.New(err.Message):
					case <-ctx.Done():
					}
				}
			}()

			c.Header("Content-Type", "text/event-stream")
			c.Stream(func(w io.Writer) bool {
				select {
				case err, ok := <-chanError:
					if !ok {
						return false
					}
					if err != nil {
						c.JSON(http.StatusInternalServerError, err.Error())
					}
					return false

				case msg, ok := <-chanStream:
					if !ok {
						return false
					}
					if string(msg) == "\n" {
						return true
					}

					message := strings.TrimLeft(string(msg), "data: ")
					c.SSEvent("message", message)
					return true

				case <-ctx.Done():
					return false
				}
			})

			// 读循环退出后，立即取消上游请求
			cancel()

			// 带超时等待，防止外部协程脱缰
			select {
			case <-done:
			case <-time.After(10 * time.Second):
			}
			return
		}

		return
	}
}
