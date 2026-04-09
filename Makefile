.PHONY: css build run dev clean

# ── CSS ──────────────────────────────────────────────────────────────────────
# Recompile Tailwind CSS from source templates.
# Run this after editing any .html template or internal/web/static/css/input.css
css:
	./internal/web/static/css/tailwind \
		-i ./internal/web/static/css/input.css \
		-o ./internal/web/static/css/output.css

# Watch mode — recompiles on every template save (useful during development)
css-watch:
	./internal/web/static/css/tailwind \
		-i ./internal/web/static/css/input.css \
		-o ./internal/web/static/css/output.css \
		--watch

# ── Build ─────────────────────────────────────────────────────────────────────
build: css
	go build -o server ./cmd/server

# ── Run ───────────────────────────────────────────────────────────────────────
run: build
	DB_NAME=mangabase DB_USER=mangabase_user DB_PASS=localpassword DB_HOST=127.0.0.1 ./server

# Dev mode: skip CSS rebuild for faster iteration
dev:
	DB_NAME=mangabase DB_USER=mangabase_user DB_PASS=localpassword DB_HOST=127.0.0.1 \
	go run ./cmd/server

# ── Clean ─────────────────────────────────────────────────────────────────────
clean:
	rm -f server
