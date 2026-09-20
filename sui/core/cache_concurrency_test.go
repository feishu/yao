package core

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheConcurrency(t *testing.T) {
	CleanCache()
	defer CleanCache()

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	// 并发写与读
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("cache_key_%d_%d", workerID, j)
				item := &Cache{
					HTML: fmt.Sprintf("<div>%s</div>", key),
				}
				SetCache(key, item)

				cached := GetCache(key)
				if cached != nil {
					assert.Equal(t, item.HTML, cached.HTML)
				}

				if j%2 == 0 {
					RemoveCache(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestScriptConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				file := fmt.Sprintf("script_%d_%d.js", workerID, j)
				script := &Script{}

				SetScript(file, script)

				got := GetScript(file)
				if got != nil {
					assert.Equal(t, script, got)
				}

				if j%2 == 0 {
					RemoveScript(file)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestComponentConcurrency(t *testing.T) {
	ClearComponentCache()
	defer ClearComponentCache()

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				route := fmt.Sprintf("comp_%d_%d", workerID, j)
				comp := &JitComponent{
					route: route,
					html:  "<span>component</span>",
				}

				componentsLock.Lock()
				Components[route] = comp
				componentsLock.Unlock()

				componentsLock.RLock()
				got, exists := Components[route]
				componentsLock.RUnlock()

				if exists {
					assert.Equal(t, comp.html, got.html)
				}

				if j%2 == 0 {
					RemoveComponentCache(route)
				}
			}
		}(i)
	}

	wg.Wait()
}
