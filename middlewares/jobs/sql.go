package jobs

// TODO @droman: add parent ID so we can have a tree of jobs for the saga pattern
var jobsTable = `

CREATE TABLE IF NOT EXISTS jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'enqueued',
	created_at INTEGER,
	updated_at INTEGER NULL,
	payload JSON,
	error TEXT NULL
);

CREATE TABLE IF NOT EXISTS workflows (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	uuid TEXT NOT NULL,
	job_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS activities (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	uuid TEXT NOT NULL,
	job_id INTEGER NOT NULL,
	workflow_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE,
	FOREIGN KEY (workflow_id) REFERENCES workflows (id) ON DELETE CASCADE
);

`
