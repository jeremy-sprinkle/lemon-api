CREATE TABLE feedback (
    id SERIAL PRIMARY KEY,
    rating INT NOT NULL,
    description VARCHAR,
    type VARCHAR NOT NULL,
    submitted TIMESTAMP,
    read BOOLEAN DEFAULT false
);

CREATE INDEX feedback_index ON feedback (id);