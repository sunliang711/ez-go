package main

import (
	"fmt"

	"github.com/sunliang711/ez-go/ezdb"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name string `gorm:"type:text;not null;"`
	Age  uint   `gorm:"type:tinyint;"`
}

func main() {
	configs := []*ezdb.DatabaseConfig{
		{
			Name:   "mysql",
			Dsn:    "root:comecome@tcp(10.1.9.155:3306)/eagle_db01?charset=utf8&parseTime=True&loc=Local&timeout=1000ms",
			Driver: ezdb.DRIVER_MYSQL,
			Tables: []ezdb.Table{
				{Name: "users", Definition: &User{}},
			},
			Log:               ezdb.LOG_LEVEL_INFO,
			ReadWriteSeparate: true,
			Replicas:          "",
			Policy:            ezdb.Random,
		},
	}
	db, err := ezdb.NewDatabase(configs, true)
	if err != nil {
		panic(err)
	}
	_ = db

	defer db.Close()

	user := &User{Name: "sunl"}
	err = db.GetDB("mysql").Create(user).Error
	if err != nil {
		panic(err)
	}
	fmt.Printf("User created: %+v\n", user)

}
