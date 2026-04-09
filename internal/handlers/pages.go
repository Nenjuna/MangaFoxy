package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
)

func AboutPage(c *gin.Context) {
	Render(c, http.StatusOK, "about", WithUser(c, gin.H{
		"requestURL": BuildAbsoluteURL(c),
	}))
}

func ContactPage(c *gin.Context) {
	if c.Request.Method == http.MethodPost {
		// Contact form submission — currently a no-op like Django version
		Render(c, http.StatusOK, "contact", WithUser(c, gin.H{
			"requestURL": BuildAbsoluteURL(c),
			"success":    true,
		}))
		return
	}
	Render(c, http.StatusOK, "contact", WithUser(c, gin.H{
		"requestURL": BuildAbsoluteURL(c),
		"success":    false,
	}))
}

func CopyrightPage(c *gin.Context) {
	Render(c, http.StatusOK, "copyright", WithUser(c, gin.H{
		"requestURL": BuildAbsoluteURL(c),
	}))
}

func TermsPage(c *gin.Context) {
	Render(c, http.StatusOK, "terms", WithUser(c, gin.H{
		"requestURL": BuildAbsoluteURL(c),
	}))
}

func UpdatesPage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
			page = p
		}
		const perPage = 20

		var total int64
		db.Model(&models.Update{}).Count(&total)

		var updates []models.Update
		db.Preload("Manga").Preload("Chapter").Preload("Chapter.Manga").
			Order("created_at DESC").
			Offset((page - 1) * perPage).
			Limit(perPage).
			Find(&updates)

		totalPages := int((total + perPage - 1) / perPage)
		if totalPages == 0 {
			totalPages = 1
		}

		type UpdatePage struct {
			Items      []models.Update
			Page       int
			TotalPages int
			HasPrev    bool
			HasNext    bool
			PrevPage   int
			NextPage   int
		}

		Render(c, http.StatusOK, "updates", WithUser(c, gin.H{
			"updates": UpdatePage{
				Items:      updates,
				Page:       page,
				TotalPages: totalPages,
				HasPrev:    page > 1,
				HasNext:    page < totalPages,
				PrevPage:   page - 1,
				NextPage:   page + 1,
			},
			"requestURL":      BuildAbsoluteURL(c),
			"metaDescription": "Latest updates on new manga releases, chapter updates, and site news.",
			"metaKeywords":    "manga updates, new chapters, new manga, site news",
		}))
	}
}
