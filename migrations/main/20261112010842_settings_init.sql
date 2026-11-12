-- +goose Up
-- create "settings" table
CREATE TABLE "settings" ("currency" character varying NOT NULL, "max_amount" bigint NOT NULL, PRIMARY KEY ("currency"));

-- +goose Down
-- reverse: create "settings" table
DROP TABLE "settings";
