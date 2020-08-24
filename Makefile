.PHONY: local
local: build
	heroku local

.PHONY: build
build: static/tailwind.css
	go build -o bin/playlist-rotator .

static/tailwind.css: tailwind.config.js postcss.config.js css/tailwind.css
	npm run build

.PHONY: db-shell
db-shell:
	# sudo -u postgres psql postgres://postgres:postgres@localhost:5432/playlist-rotator
	psql postgres://postgres:postgres@localhost:5432/playlist-rotator?sslmode=disable

# Useful commands
# migrate -database 'postgres://postgres:postgres@localhost:5432/playlist-rotator' -path migrations up
