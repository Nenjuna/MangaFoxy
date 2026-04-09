package viewlog

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mangafoxy/internal/models"
)

// ---------------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------------

func GetClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	return c.ClientIP()
}

func GetOrCreateSessionID(c *gin.Context) string {
	const cookieName = "sessionid"
	if val, err := c.Cookie(cookieName); err == nil && val != "" {
		return val
	}
	raw := GetClientIP(c) + "|" + c.Request.UserAgent()
	var sum uint32
	for _, ch := range raw {
		sum = sum*31 + uint32(ch)
	}
	sid := fmt.Sprintf("%08x", sum)
	c.SetCookie(cookieName, sid, 60*60*24*30, "/", "", false, true)
	return sid
}

// LogView inserts a raw view event into go_viewlog.
// The ViewProcessor will aggregate it into view_count on its next flush cycle.
func LogView(db *gorm.DB, c *gin.Context, contentType string, objectID uint) {
	db.Create(&models.ViewLog{
		IPAddress:   GetClientIP(c),
		SessionID:   GetOrCreateSessionID(c),
		ContentType: contentType,
		ObjectID:    objectID,
	})
}

// ProxyImageURL converts an external image URL to our proxy endpoint URL.
func ProxyImageURL(imageURL string) string {
	if imageURL == "" || strings.HasPrefix(imageURL, "/") || strings.HasPrefix(imageURL, "data:") {
		return imageURL
	}
	return "/proxy/image/?url=" + url.QueryEscape(imageURL)
}

// ---------------------------------------------------------------------------
// ViewProcessor — background rollup + cleanup
// ---------------------------------------------------------------------------

// ViewProcessor aggregates raw go_viewlog rows into the view_count columns
// on manga_manga and manga_chapter, then purges old raw rows.
type ViewProcessor struct {
	db            *gorm.DB
	flushInterval time.Duration
	cleanInterval time.Duration
	retention     time.Duration
}

func NewProcessor(db *gorm.DB) *ViewProcessor {
	return &ViewProcessor{
		db:            db,
		flushInterval: 5 * time.Minute,
		cleanInterval: 24 * time.Hour,
		retention:     7 * 24 * time.Hour,
	}
}

// Start launches the flush and cleanup goroutines.
// It returns immediately; pass a cancellable context to stop them.
func (vp *ViewProcessor) Start(ctx context.Context) {
	// Run an immediate flush at startup so view counts are fresh after a restart.
	if n, err := vp.Flush(); err != nil {
		log.Printf("viewlog: initial flush error: %v", err)
	} else if n > 0 {
		log.Printf("viewlog: initial flush processed %d events", n)
	}

	go vp.loop(ctx, vp.flushInterval, func() {
		if n, err := vp.Flush(); err != nil {
			log.Printf("viewlog: flush error: %v", err)
		} else if n > 0 {
			log.Printf("viewlog: flushed %d events into view_count", n)
		}
	})

	go vp.loop(ctx, vp.cleanInterval, func() {
		if n, err := vp.Cleanup(); err != nil {
			log.Printf("viewlog: cleanup error: %v", err)
		} else if n > 0 {
			log.Printf("viewlog: purged %d raw entries older than %v", n, vp.retention)
		}
	})

	log.Printf("viewlog: processor started (flush every %v, retention %v)",
		vp.flushInterval, vp.retention)
}

func (vp *ViewProcessor) loop(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

// ---------------------------------------------------------------------------
// Flush — roll up unprocessed rows into view_count
// ---------------------------------------------------------------------------

type viewCount struct {
	ObjectID uint
	Total    int64
}

// Flush aggregates all unprocessed go_viewlog rows and increments view_count
// on the corresponding manga_manga / manga_chapter rows, then marks them processed.
// Returns the total number of raw rows consumed.
func (vp *ViewProcessor) Flush() (int64, error) {
	db := vp.db

	// Collect unprocessed row IDs first so we can mark exactly those processed,
	// even if new rows arrive during the flush.
	var ids []uint
	if err := db.Model(&models.ViewLog{}).
		Where("processed = false").
		Pluck("id", &ids).Error; err != nil {
		return 0, fmt.Errorf("fetch ids: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	// Aggregate by (content_type, object_id) within the snapshot.
	type row struct {
		ContentType string
		ObjectID    uint
		Total       int64
	}
	var rows []row
	if err := db.Model(&models.ViewLog{}).
		Select("content_type, object_id, COUNT(*) AS total").
		Where("id IN ?", ids).
		Group("content_type, object_id").
		Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("aggregate: %w", err)
	}

	// Apply increments inside a transaction.
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			switch r.ContentType {
			case "manga":
				if err := tx.Model(&models.Manga{}).
					Where("id = ?", r.ObjectID).
					UpdateColumn("view_count", gorm.Expr("view_count + ?", r.Total)).
					Error; err != nil {
					return err
				}
			case "chapter":
				if err := tx.Model(&models.Chapter{}).
					Where("id = ?", r.ObjectID).
					UpdateColumn("view_count", gorm.Expr("view_count + ?", r.Total)).
					Error; err != nil {
					return err
				}
			}
		}
		// Mark exactly the snapshot rows as processed.
		return tx.Model(&models.ViewLog{}).
			Where("id IN ?", ids).
			Update("processed", true).Error
	}); err != nil {
		return 0, fmt.Errorf("transaction: %w", err)
	}

	return int64(len(ids)), nil
}

// ---------------------------------------------------------------------------
// Cleanup — delete processed rows older than retention period
// ---------------------------------------------------------------------------

// Cleanup deletes processed go_viewlog rows older than the retention window.
// Unprocessed rows (not yet flushed) are preserved regardless of age.
func (vp *ViewProcessor) Cleanup() (int64, error) {
	cutoff := time.Now().UTC().Add(-vp.retention)
	result := vp.db.
		Where("processed = true AND timestamp < ?", cutoff).
		Delete(&models.ViewLog{})
	return result.RowsAffected, result.Error
}

// ---------------------------------------------------------------------------
// Stats — for diagnostics / admin
// ---------------------------------------------------------------------------

type Stats struct {
	PendingEvents   int64
	ProcessedEvents int64
	OldestPending   *time.Time
}

func GetStats(db *gorm.DB) Stats {
	var s Stats
	db.Model(&models.ViewLog{}).Where("processed = false").Count(&s.PendingEvents)
	db.Model(&models.ViewLog{}).Where("processed = true").Count(&s.ProcessedEvents)
	db.Model(&models.ViewLog{}).Where("processed = false").
		Order("timestamp ASC").Limit(1).Pluck("timestamp", &s.OldestPending)
	return s
}
