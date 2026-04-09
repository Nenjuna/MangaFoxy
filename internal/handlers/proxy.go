package handlers

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type imageCache struct {
	mu      sync.RWMutex
	entries map[string]imageCacheEntry
}

type imageCacheEntry struct {
	data        []byte
	contentType string
	etag        string
	expiresAt   time.Time
}

var imgCache = &imageCache{entries: map[string]imageCacheEntry{}}

func (ic *imageCache) get(key string) (imageCacheEntry, bool) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	e, ok := ic.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return imageCacheEntry{}, false
	}
	return e, true
}

func (ic *imageCache) set(key string, e imageCacheEntry) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.entries[key] = e
}

const proxyUserAgent = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36"

func ImageProxy(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		c.String(http.StatusBadRequest, "Missing url parameter")
		return
	}

	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		c.String(http.StatusBadRequest, "Invalid URL")
		return
	}

	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(imageURL)))

	if cached, ok := imgCache.get(cacheKey); ok {
		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("ETag", cached.etag)
		c.Header("Content-Length", fmt.Sprintf("%d", len(cached.data)))
		c.Data(http.StatusOK, cached.contentType, cached.data)
		return
	}

	// Build referer from origin URL
	referer := ""
	if parsed, err := url.Parse(imageURL); err == nil {
		referer = parsed.Scheme + "://" + parsed.Host
	}

	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		c.String(http.StatusBadGateway, "Error fetching image")
		return
	}
	req.Header.Set("User-Agent", proxyUserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
		}
		c.String(http.StatusBadGateway, "Error loading image: %v", err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.String(http.StatusBadGateway, "Error reading image")
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	etag := fmt.Sprintf("%x", md5.Sum(data))

	imgCache.set(cacheKey, imageCacheEntry{
		data:        data,
		contentType: contentType,
		etag:        etag,
		expiresAt:   time.Now().Add(24 * time.Hour),
	})

	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("ETag", etag)
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, contentType, data)
}
