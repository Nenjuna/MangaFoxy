package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var tmplCache = map[string]*template.Template{}

var funcMap = template.FuncMap{
	"proxyImage": func(imageURL string) string {
		if imageURL == "" {
			return imageURL
		}
		if strings.HasPrefix(imageURL, "/") || strings.HasPrefix(imageURL, "data:") {
			return imageURL
		}
		return "/proxy/image/?url=" + url.QueryEscape(imageURL)
	},
	"slugify": func(s string) string {
		s = strings.ToLower(s)
		var sb strings.Builder
		prev := false
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				sb.WriteRune(r)
				prev = false
			} else if !prev {
				sb.WriteRune('-')
				prev = true
			}
		}
		return strings.Trim(sb.String(), "-")
	},
	"join": func(sep string, items []string) string {
		return strings.Join(items, sep)
	},
	"truncateWords": func(n int, s string) string {
		words := strings.Fields(s)
		if len(words) <= n {
			return s
		}
		return strings.Join(words[:n], " ") + "..."
	},
	"title": strings.Title,
	"year":  func() string { return fmt.Sprintf("%d", time.Now().Year()) },
	"formatDate": func(t time.Time, layout string) string {
		switch layout {
		case "F j, Y":
			return t.Format("January 2, 2006")
		case "c":
			return t.Format(time.RFC3339)
		}
		return t.Format(layout)
	},
	"linebreaks": func(s string) template.HTML {
		escaped := template.HTMLEscapeString(s)
		result := strings.ReplaceAll(escaped, "\n\n", "</p><p>")
		result = strings.ReplaceAll(result, "\n", "<br>")
		return template.HTML("<p>" + result + "</p>")
	},
	"decodeJSON": func(data []byte) interface{} {
		var v interface{}
		json.Unmarshal(data, &v)
		return v
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"seq": func(start, end int) []int {
		var s []int
		for i := start; i <= end; i++ {
			s = append(s, i)
		}
		return s
	},
	"urlEncode": url.QueryEscape,
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict requires even number of arguments")
		}
		d := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			d[key] = values[i+1]
		}
		return d, nil
	},
	"first": func(s []string) string {
		if len(s) == 0 {
			return ""
		}
		return s[0]
	},
}

// LoadTemplates parses all template combinations at startup.
func LoadTemplates() {
	base := "templates/base.html"
	card := "templates/components/manga_card.html"
	share := "templates/components/share_buttons.html"

	pages := map[string][]string{
		"index":          {base, card, "templates/index.html"},
		"manga_detail":   {base, card, share, "templates/manga_detail.html"},
		"chapter_detail": {base, share, "templates/chapter_detail.html"},
		"genre":          {base, card, "templates/genre.html"},
		"updates":        {base, "templates/updates.html"},
		"about":          {base, "templates/about.html"},
		"contact":        {base, "templates/contact.html"},
		"copyright":      {base, "templates/copyright.html"},
		"terms":          {base, "templates/terms.html"},
	}

	for name, files := range pages {
		t, err := template.New("").Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}
		tmplCache[name] = t
	}
}

// Render executes a named template and writes to the response.
func Render(c *gin.Context, status int, tmplName string, data interface{}) {
	t, ok := tmplCache[tmplName]
	if !ok {
		c.String(http.StatusInternalServerError, "template %s not found", tmplName)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		log.Printf("Template execution error for %s: %v", tmplName, err)
	}
}

// WithUser merges the authenticated user (if any) into template data.
func WithUser(c *gin.Context, data gin.H) gin.H {
	if user, exists := c.Get("currentUser"); exists {
		data["CurrentUser"] = user
	}
	return data
}

// BuildAbsoluteURL constructs the full request URL.
func BuildAbsoluteURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + c.Request.Host + c.Request.RequestURI
}
