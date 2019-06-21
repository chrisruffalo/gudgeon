-- add service time (ms) to buffer
ALTER TABLE buffer ADD COLUMN ServiceTime INTEGER DEFAULT 0;
UPDATE buffer SET ServiceTime = 0 WHERE ServiceTime = null;

-- add service time (ms) to qlog
ALTER TABLE qlog ADD COLUMN ServiceTime INTEGER DEFAULT 0;
UPDATE qlog SET ServiceTime = 0 WHERE ServiceTime = null;