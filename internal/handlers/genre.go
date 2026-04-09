package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
)

// GenreList renders the genre browsing page (no genre selected).
func GenreList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		renderGenrePage(db, c, "")
	}
}

// GenreDetail renders the genre browsing page for a specific genre.
func GenreDetail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		genreSlug := c.Param("genre_slug")
		renderGenrePage(db, c, genreSlug)
	}
}

func renderGenrePage(db *gorm.DB, c *gin.Context, genreSlug string) {
	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	const perPage = 20 // 4 columns × 5 rows

	// Collect all unique genres
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
	sort.Strings(genres)

	// Determine selected genre
	selectedGenre := ""
	if genreSlug != "" {
		selectedGenre = strings.ReplaceAll(genreSlug, "-", " ")
		// Title-case
		words := strings.Fields(selectedGenre)
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		selectedGenre = strings.Join(words, " ")
	}

	// Query mangas filtered by genre
	var mangas []models.Manga
	var total int64
	q := db.Model(&models.Manga{})
	if selectedGenre != "" {
		q = q.Where("genre::text ILIKE ?", "%"+selectedGenre+"%")
	}
	q.Count(&total)
	q.Offset((page - 1) * perPage).Limit(perPage).Find(&mangas)

	if selectedGenre != "" && len(mangas) == 0 && page == 1 {
		c.String(http.StatusNotFound, "Genre not found")
		return
	}

	totalPages := int((total + perPage - 1) / perPage)
	if totalPages == 0 {
		totalPages = 1
	}

	Render(c, http.StatusOK, "genre", WithUser(c, gin.H{
		"genres":        genres,
		"selectedGenre": selectedGenre,
		"mangas": MangaPage{
			Items:      decodeMangaViewList(mangas),
			Page:       page,
			TotalPages: totalPages,
			HasPrev:    page > 1,
			HasNext:    page < totalPages,
			PrevPage:   page - 1,
			NextPage:   page + 1,
		},
		"requestURL": BuildAbsoluteURL(c),
	}))
}
