package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
)

// MangaPage holds a paginated slice of ranked manga views.
type MangaPage struct {
	Items      []MangaView
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// MangaView wraps Manga with decoded/computed fields for templates.
type MangaView struct {
	models.Manga
	Genres       []string
	Rank         int    // 1-based global rank on this page
	ChapterCount int64  // total chapters for this manga
	LatestChapter string // chapter number of newest chapter, e.g. "1079"
}

func decodeMangaGenres(m models.Manga) MangaView {
	mv := MangaView{Manga: m}
	json.Unmarshal(m.Genre, &mv.Genres)
	return mv
}

// buildRankedViews decodes genres and assigns global rank numbers starting at offset+1.
func buildRankedViews(mangas []models.Manga, offset int) []MangaView {
	views := make([]MangaView, len(mangas))
	for i, m := range mangas {
		views[i] = decodeMangaGenres(m)
		views[i].Rank = offset + i + 1
	}
	return views
}

// decodeMangaViewList is kept for handlers that don't need ranking.
func decodeMangaViewList(mangas []models.Manga) []MangaView {
	return buildRankedViews(mangas, 0)
}

func Home(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
			page = p
		}
		const perPage = 20 // 4 columns × 5 rows

		q := c.Query("q")

		query := db.Model(&models.Manga{})
		if q != "" {
			query = query.Where("title ILIKE ?", "%"+q+"%")
		}

		var total int64
		query.Count(&total)

		var mangas []models.Manga
		query.Order("view_count DESC, title ASC").
			Offset((page - 1) * perPage).
			Limit(perPage).
			Find(&mangas)

		totalPages := int((total + perPage - 1) / perPage)
		if totalPages == 0 {
			totalPages = 1
		}

		offset := (page - 1) * perPage
		items := buildRankedViews(mangas, offset)

		// Attach chapter counts and latest chapter number
		for i := range items {
			var count int64
			db.Model(&models.Chapter{}).Where("manga_id = ?", items[i].ID).Count(&count)
			items[i].ChapterCount = count

			if count > 0 {
				var latest models.Chapter
				db.Where("manga_id = ?", items[i].ID).
					Order("CAST(chapter_number AS FLOAT) DESC").
					Limit(1).
					First(&latest)
				items[i].LatestChapter = latest.ChapterNumber
			}
		}

		mp := MangaPage{
			Items:      items,
			Page:       page,
			TotalPages: totalPages,
			HasPrev:    page > 1,
			HasNext:    page < totalPages,
			PrevPage:   page - 1,
			NextPage:   page + 1,
		}

		// Featured manga: top result on page 1 (no search)
		var featured *MangaView
		if page == 1 && q == "" && len(items) > 0 {
			f := items[0]
			featured = &f
		}

		Render(c, http.StatusOK, "index", WithUser(c, gin.H{
			"mangas":      mp,
			"featured":    featured,
			"searchQuery": q,
			"requestURL":  BuildAbsoluteURL(c),
		}))
	}
}
