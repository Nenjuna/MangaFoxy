package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
	"mangafoxy/internal/scraper"
	"mangafoxy/internal/viewlog"
)

// MangaDetail renders the manga detail page.
func MangaDetail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var manga models.Manga
		if err := db.Where("slug = ?", slug).First(&manga).Error; err != nil {
			c.Status(http.StatusNotFound)
			c.String(http.StatusNotFound, "Manga not found")
			return
		}

		// Count existing chapters
		var chapCount int64
		db.Model(&models.Chapter{}).Where("manga_id = ?", manga.ID).Count(&chapCount)

		if chapCount == 0 {
			scraped, err := scraper.ScrapeChapters(slug)
			if err == nil {
				saveChapters(db, &manga, scraped)
			}
		}

		var chapters []models.Chapter
		db.Where("manga_id = ?", manga.ID).Find(&chapters)

		// Sort by chapter number numerically descending
		sort.Slice(chapters, func(i, j int) bool {
			ni, _ := strconv.ParseFloat(chapters[i].ChapterNumber, 64)
			nj, _ := strconv.ParseFloat(chapters[j].ChapterNumber, 64)
			return ni > nj
		})

		var firstChapter, latestChapter *models.Chapter
		if len(chapters) > 0 {
			last := chapters[len(chapters)-1]
			firstChapter = &last
			first := chapters[0]
			latestChapter = &first
		}

		mv := decodeMangaGenres(manga)

		viewlog.LogView(db, c, "manga", manga.ID)

		var genres []string
		json.Unmarshal(manga.Genre, &genres)
		metaKeywords := strings.Join(genres, ", ")

		summary := manga.Summary
		words := strings.Fields(summary)
		if len(words) > 30 {
			summary = strings.Join(words[:30], " ") + "..."
		}

		Render(c, http.StatusOK, "manga_detail", WithUser(c, gin.H{
			"manga":           mv,
			"chapters":        chapters,
			"firstChapter":    firstChapter,
			"latestChapter":   latestChapter,
			"requestURL":      BuildAbsoluteURL(c),
			"metaDescription": fmt.Sprintf("Read %s online, updated daily.", manga.Title),
			"metaKeywords":    metaKeywords,
			"genreJSON":       string(manga.Genre),
		}))
	}
}

func saveChapters(db *gorm.DB, manga *models.Manga, entries []scraper.ChapterEntry) {
	for _, e := range entries {
		chapterNumber := models.ExtractChapterNumber(e.Title)
		slug := models.Slugify(e.Title)
		var ch models.Chapter
		result := db.Where("manga_id = ? AND chapter_number = ?", manga.ID, chapterNumber).First(&ch)
		if result.Error != nil {
			db.Create(&models.Chapter{
				MangaID:       manga.ID,
				Title:         e.Title,
				ChapterNumber: chapterNumber,
				Slug:          slug,
				MangaowlURL:   e.URL,
			})
		}
	}
}

// --- REST API handlers ---

func MangaList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var mangas []models.Manga
		db.Find(&mangas)
		c.JSON(http.StatusOK, mangas)
	}
}

func MangaCreate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var manga models.Manga
		if err := c.ShouldBindJSON(&manga); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if manga.Slug == "" {
			manga.Slug = models.Slugify(manga.Title)
		}
		if err := db.Create(&manga).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, manga)
	}
}

func MangaGet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		var manga models.Manga
		if err := db.Where("slug = ?", slug).First(&manga).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, manga)
	}
}

func MangaUpdate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		var manga models.Manga
		if err := db.Where("slug = ?", slug).First(&manga).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := c.ShouldBindJSON(&manga); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Save(&manga)
		c.JSON(http.StatusOK, manga)
	}
}

func MangaDelete(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		var manga models.Manga
		if err := db.Where("slug = ?", slug).First(&manga).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		db.Delete(&manga)
		c.Status(http.StatusNoContent)
	}
}
