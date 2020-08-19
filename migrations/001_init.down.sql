DROP TRIGGER update_time_users ON users;
DROP TRIGGER update_time_playlists ON playlists;
DROP FUNCTION update_modified_column();
DROP TABLE playlists;
DROP TABLE users;
DROP EXTENSION "uuid-ossp";