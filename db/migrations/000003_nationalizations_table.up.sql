CREATE TABLE IF NOT EXISTS nationalizations (
	people_id   SERIAL       NOT NULL REFERENCES peoples(people_id),
	country_id  VARCHAR(4)[] NOT NULL,
	probability NUMERIC[]    NOT NULL
);
