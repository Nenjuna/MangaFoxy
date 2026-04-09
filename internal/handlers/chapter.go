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

// ChapterView wraps Chapter with decoded image URLs.
type ChapterView struct {
	models.Chapter
	ImageURLList []string
}

func decodeChapterImages(ch models.Chapter) ChapterView {
	cv := ChapterView{Chapter: ch}
	json.Unmarshal(ch.ImageURLs, &cv.ImageURLList)
	return cv
}

func ChapterDetail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		mangaSlug := c.Param("slug")
		chapterSlug := c.Param("chapter_number")

		var manga models.Manga
		if err := db.Where("slug = ?", mangaSlug).First(&manga).Error; err != nil {
			c.String(http.StatusNotFound, "Manga not found")
			return
		}

		var chapter models.Chapter
		if err := db.Where("manga_id = ? AND slug = ?", manga.ID, chapterSlug).First(&chapter).Error; err != nil {
			c.String(http.StatusNotFound, "Chapter not found")
			return
		}

		// Scrape images if missing
		var imgList []string
		json.Unmarshal(chapter.ImageURLs, &imgList)
		if len(imgList) == 0 && chapter.MangaowlURL != "" {
			imgs, err := scraper.ScrapeImages(chapter.MangaowlURL)
			if err == nil && len(imgs) > 0 {
				imgList = imgs
				raw, _ := json.Marshal(imgs)
				chapter.ImageURLs = raw
				db.Model(&chapter).Update("image_urls", raw)
			}
		}

		// Load all chapters for prev/next navigation
		var allChapters []models.Chapter
		db.Where("manga_id = ?", manga.ID).Find(&allChapters)
		sort.Slice(allChapters, func(i, j int) bool {
			ni, _ := strconv.ParseFloat(allChapters[i].ChapterNumber, 64)
			nj, _ := strconv.ParseFloat(allChapters[j].ChapterNumber, 64)
			return ni < nj
		})

		curNum, _ := strconv.ParseFloat(chapter.ChapterNumber, 64)
		var prevChapter, nextChapter *models.Chapter
		for i, ch := range allChapters {
			n, _ := strconv.ParseFloat(ch.ChapterNumber, 64)
			if n < curNum {
				tmp := allChapters[i]
				prevChapter = &tmp
			}
			if n > curNum && nextChapter == nil {
				tmp := allChapters[i]
				nextChapter = &tmp
			}
		}

		viewlog.LogView(db, c, "chapter", chapter.ID)

		mv := decodeMangaGenres(manga)
		cv := decodeChapterImages(chapter)

		var genres []string
		json.Unmarshal(manga.Genre, &genres)
		keywords := append(genres, manga.Title, chapter.Title)

		summary := manga.Summary
		words := strings.Fields(summary)
		if len(words) > 30 {
			summary = strings.Join(words[:30], " ") + "..."
		}

		Render(c, http.StatusOK, "chapter_detail", WithUser(c, gin.H{
			"manga":           mv,
			"chapter":         cv,
			"prevChapter":     prevChapter,
			"nextChapter":     nextChapter,
			"requestURL":      BuildAbsoluteURL(c),
			"metaDescription": fmt.Sprintf("Read %s %s online. Updated daily on MangaFoxy.", manga.Title, chapter.Title),
			"metaKeywords":    strings.Join(keywords, ", "),
			"summaryShort":    summary,
		}))
	}
}

// --- REST API handlers ---

func ChapterList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var chapters []models.Chapter
		db.Find(&chapters)
		c.JSON(http.StatusOK, chapters)
	}
}

func ChapterCreate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var chapter models.Chapter
		if err := c.ShouldBindJSON(&chapter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if chapter.ChapterNumber == "" {
			chapter.ChapterNumber = models.ExtractChapterNumber(chapter.Title)
		}
		if chapter.Slug == "" {
			chapter.Slug = models.Slugify(chapter.Title)
		}
		if err := db.Create(&chapter).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, chapter)
	}
}

func ChapterGet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("pk")
		var chapter models.Chapter
		if err := db.First(&chapter, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, chapter)
	}
}

func ChapterUpdate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("pk")
		var chapter models.Chapter
		if err := db.First(&chapter, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := c.ShouldBindJSON(&chapter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Save(&chapter)
		c.JSON(http.StatusOK, chapter)
	}
}

func ChapterDelete(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("pk")
		var chapter models.Chapter
		if err := db.First(&chapter, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		db.Delete(&chapter)
		c.Status(http.StatusNoContent)
	}
}
