package models

import (
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
)

type Manga struct {
	ID                 uint           `gorm:"primaryKey"`
	Title              string         `gorm:"size:255;not null"`
	Alternative        string         `gorm:"size:255"`
	Slug               string         `gorm:"uniqueIndex;size:255"`
	Status             string         `gorm:"size:10"`
	Genre              datatypes.JSON `gorm:"type:jsonb"`
	Rating             *float64
	Rank               *uint
	Year               *uint
	ImageURL           string         `gorm:"column:image_url"`
	LastUpdatedChapter string         `gorm:"size:100"`
	Summary            string
	Chapter            datatypes.JSON `gorm:"type:jsonb"`
	LastScraped        *time.Time
	ViewCount          uint      `gorm:"default:0"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
	Chapters           []Chapter `gorm:"foreignKey:MangaID"`
}

func (Manga) TableName() string   { return "manga_manga" }
func (Chapter) TableName() string { return "manga_chapter" }
func (ViewLog) TableName() string { return "go_viewlog" } // new table; Django's manga_viewlog uses a different schema
func (Update) TableName() string  { return "manga_update" }

func (m Manga) ThumbnailURL() string {
	if m.ImageURL != "" {
		return m.ImageURL
	}
	return "/static/images/no-thumbnail.png"
}

type Chapter struct {
	ID            uint           `gorm:"primaryKey"`
	MangaID       uint           `gorm:"not null;index"`
	Manga         Manga          `gorm:"foreignKey:MangaID"`
	Title         string         `gorm:"size:255;not null"`
	Subtitle      string         `gorm:"size:255"`
	ChapterNumber string         `gorm:"size:50;not null"`
	Slug          string         `gorm:"size:255"`
	ImageURLs     datatypes.JSON `gorm:"type:jsonb;column:image_urls"`
	MangaowlURL   string         `gorm:"column:mangaowl_url;size:500"`
	ViewCount     uint           `gorm:"default:0"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
}

type ViewLog struct {
	ID          uint      `gorm:"primaryKey"`
	IPAddress   string    `gorm:"size:45"`
	SessionID   string    `gorm:"size:100"`
	Timestamp   time.Time `gorm:"autoCreateTime"`
	Processed   bool      `gorm:"default:false"`
	ContentType string    `gorm:"size:20"` // "manga" or "chapter"
	ObjectID    uint
}

// Composite indexes are created by database.ensureViewLogIndexes:
//   idx_viewlog_flush   (processed, content_type, object_id)
//   idx_viewlog_cleanup (processed, timestamp)

type Update struct {
	ID         uint    `gorm:"primaryKey"`
	Title      string  `gorm:"size:200;not null"`
	Content    string
	UpdateType string   `gorm:"size:20"`
	MangaID    *uint
	Manga      *Manga   `gorm:"foreignKey:MangaID"`
	ChapterID  *uint
	Chapter    *Chapter `gorm:"foreignKey:ChapterID"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (u Update) UpdateTypeDisplay() string {
	switch u.UpdateType {
	case "new_manga":
		return "New Manga"
	case "new_chapter":
		return "New Chapter"
	case "site_update":
		return "Site Update"
	}
	return u.UpdateType
}

// ExtractChapterNumber extracts the chapter number from a title string.
func ExtractChapterNumber(title string) string {
	re1 := regexp.MustCompile(`(?i)chapter\s*([\d.]+)`)
	if m := re1.FindStringSubmatch(title); m != nil {
		return m[1]
	}
	re2 := regexp.MustCompile(`[\d.]+`)
	if m := re2.FindString(title); m != "" {
		return m
	}
	return "0"
}

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
