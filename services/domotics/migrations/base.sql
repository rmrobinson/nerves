CREATE TABLE IF NOT EXISTS building(
		id TEXT NOT NULL PRIMARY KEY,
		name TEXT,
		description TEXT,
		address TEXT
		);
CREATE TABLE IF NOT EXISTS floor(
		id TEXT NOT NULL PRIMARY KEY,
		building_id TEXT,
		name TEXT,
		description TEXT,
		level INT,
		FOREIGN KEY(building_id) REFERENCES building(id)
		);
CREATE TABLE IF NOT EXISTS room(
		id TEXT NOT NULL PRIMARY KEY,
		floor_id TEXT,
		name TEXT,
		description TEXT,
		FOREIGN KEY(floor_id) REFERENCES floor(id)
		);