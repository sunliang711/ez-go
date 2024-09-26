package main

import (
	"github.com/sunliang711/ez-go/ezdb"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name string `gorm:"type:varchar(100);not null;"`
}

func main() {
	configs := []ezdb.DatabaseConfig{{
		Name:   "mysql",
		Dsn:    "root:root@tcp(10.1.9.120:3306)/go_admin2?charset=utf8&parseTime=True&loc=Local&timeout=1000ms",
		Driver: "mysql",
		Tables: []ezdb.Table{{Name: "users", Definition: &User{}}},
		Log:    ezdb.DBLOG_SILENT,
	}}
	db, err := ezdb.NewDatabase(configs, true)
	if err != nil {
		panic(err)
	}
	_ = db

	defer db.Close()

	err = db.GetDB("mysql").Create(&User{Name: "sunliang"}).Error
	if err != nil {
		panic(err)
	}

}
