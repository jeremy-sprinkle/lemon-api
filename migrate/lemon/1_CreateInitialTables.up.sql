CREATE TABLE feedback (
    id SERIAL NOT NULL,
    rating INT NOT NULL,
    description VARCHAR,
    type VARCHAR NOT NULL,
    submitted TIMESTAMP,
    read BOOLEAN DEFAULT false
);