CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	login TEXT UNIQUE NOT NULL,
	name TEXT NOT NULL,
	password BYTEA NOT NULL,
	created_at TIMESTAMP DEFAULT NOW(),
	updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE jobs (
	id TEXT PRIMARY KEY,
	status TEXT CHECK (status IN ('processing', 'completed', 'failed')),
	user_id SERIAL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMP DEFAULT NOW(),
	updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE archives (
	id SERIAL PRIMARY KEY,
	job_id TEXT UNIQUE NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	archive BYTEA,
	gen_count INTEGER CHECK (gen_count >= 0),
	gen_errors JSONB,
	fatal_gen_error TEXT,
	created_at TIMESTAMP DEFAULT NOW(),
	updated_at TIMESTAMP DEFAULT NOW()
);

