# Shortcut command to put the repo in development mode
.PHONY: local
local: clean
	air

.PHONY: clean
clean:
	rm static/tailwind.css

# Serve up the local binary
.PHONY: serve
serve:
	env $$(grep -v '^#' .env | xargs) ./bin/playlist-rotator serve

# Prepare a binary to serve locally. Depends on un-purged css
.PHONY: build
build: static/tailwind.css
	go build -o bin/playlist-rotator .

static/tailwind.css: tailwind.config.js postcss.config.js css/tailwind.css
	npm run build-local

# Prepare the repo for production
.PHONY: prod
prod:
	npm run build

# -------- Utility Commands --------

.PHONY: local-db-shell
local-db-shell:
	psql postgres://postgres:postgres@localhost:5432/playlist-rotator?sslmode=disable

.PHONY: heroku-db-shell
heroku-db-shell:
	heroku pg:psql

# https://stackoverflow.com/questions/19331497/set-environment-variables-from-file-of-key-value-pairs/19331521
.PHONY: run-build-cmd
run-build-cmd:
	env $$(grep -v '^#' .env | xargs) go run . build