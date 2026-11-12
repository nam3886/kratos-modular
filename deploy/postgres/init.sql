-- Scratch database that `make migration` replays the existing migrations on.
CREATE DATABASE app_dev;

-- Disposable database for TEST_DATABASE_URL; tests wipe it.
CREATE DATABASE app_test;
