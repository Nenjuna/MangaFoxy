package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
)

var (
	xmlTmplOnce  sync.Once
	xmlTemplates *template.Template
)

func loadXMLTemplates() {
	xmlTmplOnce.Do(func() {
		var err error
		xmlTemplates, err = template.New("").Funcs(funcMap).ParseFiles(
			"templates/xml/sitemap_index.xml",
			"templates/xml/sitemap_manga.xml",
			"templates/xml/sitemap_chapters.xml",
			"templates/xml/sitemap_genres.xml",
			"templates/xml/sitemap_pages.xml",
		)
		if err != nil {
			log.Fatalf("Failed to parse XML templates: %v", err)
		}
	})
}

// Simple in-memory cache for sitemaps
type sitemapCache struct {
	mu      sync.RWMutex
	entries map[string]cachedResponse
}

type cachedResponse struct {
	body      []byte
	expiresAt time.Time
}

var smCache = &sitemapCache{entries: map[string]cachedResponse{}}

func (sc *sitemapCache) get(key string) ([]byte, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	e, ok := sc.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.body, true
}

func (sc *sitemapCache) set(key string, body []byte, ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries[key] = cachedResponse{body: body, expiresAt: time.Now().Add(ttl)}
}

func renderXML(c *gin.Context, cacheKey string, tmplName string, data interface{}) {
	loadXMLTemplates()

	if cached, ok := smCache.get(cacheKey); ok {
		c.Data(http.StatusOK, "application/xml", cached)
		return
	}

	var buf []byte
	w := &bytesWriter{}
	if err := xmlTemplates.ExecuteTemplate(w, tmplName, data); err != nil {
		c.String(http.StatusInternalServerError, "sitemap error: %v", err)
		return
	}
	buf = w.b
	smCache.set(cacheKey, buf, time.Hour)
	c.Data(http.StatusOK, "application/xml", buf)
}

type bytesWriter struct{ b []byte }

func (bw *bytesWriter) Write(p []byte) (int, error) {
	bw.b = append(bw.b, p...)
	return len(p), nil
}

func SitemapIndex(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var total int64
		db.Model(&models.Chapter{}).Count(&total)
		const perPage = 1000
		totalPages := int((total + perPage - 1) / perPage)

		pages := make([]int, totalPages)
		for i := range pages {
			pages[i] = i + 1
		}

		renderXML(c, "sitemap_index", "sitemap_index.xml", gin.H{
			"chapterPages": pages,
		})
	}
}

func SitemapManga(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
			page = p
		}
		const perPage = 1000

		var mangas []models.Manga
		db.Order("updated_at DESC").
			Offset((page - 1) * perPage).
			Limit(perPage).
			Find(&mangas)

		renderXML(c, "sitemap_manga_"+strconv.Itoa(page), "sitemap_manga.xml", gin.H{
			"mangas": mangas,
		})
	}
}

func SitemapChapters(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		pageStr := strings.TrimSuffix(c.Param("page"), ".xml")
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		const perPage = 1000

		var total int64
		db.Model(&models.Chapter{}).Count(&total)
		totalPages := int((total + perPage - 1) / perPage)
		if page > totalPages {
			c.Status(http.StatusNotFound)
			return
		}

		var chapters []models.Chapter
		db.Preload("Manga").Order("created_at DESC").
			Offset((page - 1) * perPage).
			Limit(perPage).
			Find(&chapters)

		renderXML(c, "sitemap_chapters_"+strconv.Itoa(page), "sitemap_chapters.xml", gin.H{
			"chapters": chapters,
		})
	}
}

func SitemapGenres(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var allMangas []models.Manga
		db.Select("genre").Find(&allMangas)

		genreSet := map[string]bool{}
		for _, m := range allMangas {
			var gs []string
			json.Unmarshal(m.Genre, &gs)
			for _, g := range gs {
				if g != "" {
					genreSet[g] = true
				}
			}
		}
		genres := make([]string, 0, len(genreSet))
		for g := range genreSet {
			genres = append(genres, g)
		}

		renderXML(c, "sitemap_genres", "sitemap_genres.xml", gin.H{
			"genres": genres,
		})
	}
}

func SitemapPages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		renderXML(c, "sitemap_pages", "sitemap_pages.xml", nil)
	}
}
