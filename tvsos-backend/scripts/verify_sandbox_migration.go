package main

import (
	"fmt"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
)

func main() {
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	var tables []string
	database.DB.Raw("SHOW TABLES").Scan(&tables)
	fmt.Println("tables:", tables)
	var cols []string
	database.DB.Raw("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND COLUMN_NAME = 'experiment_id'").Scan(&cols)
	fmt.Println("experiment_id in:", cols)
	var users, sbs, schemes int64
	database.DB.Table("user").Count(&users)
	database.DB.Table("sandbox").Count(&sbs)
	database.DB.Table("experiment_scheme").Count(&schemes)
	fmt.Printf("user=%d sandbox=%d scheme=%d\n", users, sbs, schemes)
}
