package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"mangafoxy/internal/config"
	"mangafoxy/internal/database"
	"mangafoxy/internal/handlers"
	"mangafoxy/internal/viewlog"
	"mangafoxy/internal/web"
)

func main() {
	cfg := config.Load()

	database.Connect(cfg)
	db := database.DB

	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	viewlog.NewProcessor(db).Start(ctx)

	handlers.LoadTemplates()
	auth := handlers.InitAuth(cfg)

	r := gin.Default()
	r.Use(handlers.AuthMiddleware(auth))

	// ── Static assets — served from embedded FS ──────────────────────────────
	r.StaticFS("/static", web.StaticFS())

	// Root-level files (robots.txt, manifest.json) served from the same FS
	for _, name := range []string{"robots.txt", "manifest.json"} {
		name := name // capture for closure
		r.GET("/"+name, func(c *gin.Context) {
			data, err := web.StaticFile(name)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			ct := "text/plain"
			if name == "manifest.json" {
				ct = "application/json"
			}
			c.Data(http.StatusOK, ct, data)
		})
	}

	// ── Page routes ───────────────────────────────────────────────────────────
	r.GET("/", handlers.Home(db))
	r.GET("/about/", handlers.AboutPage)
	r.GET("/contact/", handlers.ContactPage)
	r.POST("/contact/", handlers.ContactPage)
	r.GET("/copyright/", handlers.CopyrightPage)
	r.GET("/terms/", handlers.TermsPage)
	r.GET("/updates/", handlers.UpdatesPage(db))

	// ── Genre routes ──────────────────────────────────────────────────────────
	r.GET("/genre/", handlers.GenreList(db))
	r.GET("/genre/:genre_slug/", handlers.GenreDetail(db))

	// ── Image proxy ───────────────────────────────────────────────────────────
	r.GET("/proxy/image/", handlers.ImageProxy)

	// ── Sitemaps ──────────────────────────────────────────────────────────────
	r.GET("/sitemap.xml", handlers.SitemapIndex(db))
	r.GET("/sitemap-manga.xml", handlers.SitemapManga(db))
	r.GET("/sitemap-chapters-:page.xml", handlers.SitemapChapters(db))
	r.GET("/sitemap-genres.xml", handlers.SitemapGenres(db))
	r.GET("/sitemap-pages.xml", handlers.SitemapPages(db))

	// ── Auth routes ───────────────────────────────────────────────────────────
	r.GET("/login", handlers.Login(auth))
	r.GET("/authorization-code/callback", handlers.Callback(auth))
	r.GET("/logout", handlers.Logout(auth))

	// ── REST API ──────────────────────────────────────────────────────────────
	r.GET("/api/mangas/", handlers.MangaList(db))
	r.GET("/api/mangas/:slug/", handlers.MangaGet(db))
	r.GET("/api/chapters/", handlers.ChapterList(db))
	r.GET("/api/chapters/:pk/", handlers.ChapterGet(db))

	apiProtected := r.Group("/api", handlers.RequireAuth(auth))
	apiProtected.POST("/mangas/", handlers.MangaCreate(db))
	apiProtected.PUT("/mangas/:slug/", handlers.MangaUpdate(db))
	apiProtected.DELETE("/mangas/:slug/", handlers.MangaDelete(db))
	apiProtected.POST("/chapters/", handlers.ChapterCreate(db))
	apiProtected.PUT("/chapters/:pk/", handlers.ChapterUpdate(db))
	apiProtected.DELETE("/chapters/:pk/", handlers.ChapterDelete(db))

	// ── Manga / chapter detail (catch-all — must be last) ─────────────────────
	r.GET("/:slug/", handlers.MangaDetail(db))
	r.GET("/:slug/:chapter_number/", handlers.ChapterDetail(db))

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
