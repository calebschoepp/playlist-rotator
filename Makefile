.PHONY: local
local: build
	heroku local

.PHONY: build
build: static/tailwind.css
	go build -o bin/playlist-rotator .

static/tailwind.css: tailwind.config.js postcss.config.js css/tailwind.css
	npm run build

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

# TODO figure out this whole asset pipeline thing