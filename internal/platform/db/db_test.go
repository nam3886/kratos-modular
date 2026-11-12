package db_test

import (
	"testing"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

func TestGetUnknownDatabase(t *testing.T) {
	dbs, cleanup, err := db.Open(&conf.Data{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := dbs.Get("nope"); err == nil {
		t.Fatal("Get of an unconfigured database returned no error")
	}
}
