CREATE TABLE WebhookLog (
    Id SERIAL PRIMARY KEY,
    ReceivedAt TIMESTAMP NOT NULL,
    Method VARCHAR(10) NOT NULL,
    Uri VARCHAR(255) NOT NULL,
    Headers TEXT,
    Body TEXT
);
