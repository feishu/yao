package core

import (
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// Cache the cache
type Cache struct {
	Data          string
	Global        string
	Config        string
	Guard         string
	GuardRedirect string
	HTML          string
	Root          string
	CacheStore    string
	CacheTime     time.Duration
	DataCacheTime time.Duration
	Script        *Script
	Imports       map[string]string
}

// Caches the caches
var Caches = map[string]*Cache{}
var cachesLock sync.RWMutex

// SetCache set the cache (thread-safe)
func SetCache(file string, cache *Cache) {
	cachesLock.Lock()
	defer cachesLock.Unlock()
	Caches[file] = cache
}

// GetCache get the cache (thread-safe)
func GetCache(file string) *Cache {
	cachesLock.RLock()
	defer cachesLock.RUnlock()
	if cache, has := Caches[file]; has {
		return cache
	}
	return nil
}

// RemoveCache remove the cache (thread-safe)
func RemoveCache(file string) {
	cachesLock.Lock()
	delete(Caches, file)
	cachesLock.Unlock()
	RemoveScript(file)
}

// CleanCache clean the cache (thread-safe)
func CleanCache() {
	cachesLock.Lock()
	defer cachesLock.Unlock()
	Caches = map[string]*Cache{}
}

// GetHTML get the html
func (c *Cache) GetHTML(hash string) (string, bool) {

	stor, err := store.Get(c.CacheStore)
	if err != nil {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return "", false
	}

	v, has := stor.Get(hash)
	if !has {
		return "", false
	}

	return v.(string), true
}

// GetData get the data
func (c *Cache) GetData(hash string) (Data, bool) {
	stor, err := store.Get(c.CacheStore)
	if err != nil {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return Data{}, false
	}

	v, has := stor.Get(hash)
	if !has {
		return Data{}, false
	}

	data := Data{}
	err = jsoniter.Unmarshal(v.([]byte), &data)
	if err != nil {
		log.Error(`[SUI] The data is not a valid json: %s`, err.Error())
		return Data{}, false
	}

	return data, true
}

// SetData set the data
func (c *Cache) SetData(hash string, data Data, ttl time.Duration) {
	stor, err := store.Get(c.CacheStore)
	if err != nil {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}

	raw, err := jsoniter.Marshal(data)
	if err != nil {
		log.Error(`[SUI] The data is not a valid json: %s`, err.Error())
		return
	}

	stor.Set(hash, raw, ttl)
}

// SetHTML set the html
func (c *Cache) SetHTML(hash, html string, ttl time.Duration) {
	stor, err := store.Get(c.CacheStore)
	if err != nil {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}
	stor.Set(hash, html, ttl)
}

// DelHTML del the html
func (c *Cache) DelHTML(hash string) {
	stor, err := store.Get(c.CacheStore)
	if err != nil {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}
	stor.Del(hash)
}
