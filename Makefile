local:
	go build -o bin/playlist-rotator .
	heroku local

db-shell:
	# sudo -u postgres psql postgres://postgres:postgres@localhost:5432/playlist-rotator
	psql postgres://postgres:postgres@localhost:5432/playlist-rotator?sslmode=disable

# Useful commands
# migrate -database 'postgres://postgres:postgres@localhost:5432/playlist-rotator' -path migrations up
