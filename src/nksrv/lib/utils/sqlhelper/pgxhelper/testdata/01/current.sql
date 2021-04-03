-- :set version 0
CREATE SCHEMA test;
CREATE TABLE test.test (
    t_id INTEGER  GENERATED ALWAYS AS IDENTITY,
    PRIMARY KEY (t_id)
);
