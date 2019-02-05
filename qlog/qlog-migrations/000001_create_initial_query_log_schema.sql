-- +migrate Up

-- +migrate StatementBegin
CREATE TABLE query_log ( 
    consumer_ip TEXT, 
    consumer_name TEXT, 
    consumer TEXT, 
    question_type TEXT, 
    question TEXT, 
    first_response TEXT, 
    response_count INT, 
    whole_response BLOB, 
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP 
);
-- +migrate StatementEnd

-- +migrate Down
DROP TABLE query_log;