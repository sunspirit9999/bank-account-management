package db

import (
	"account-management/model"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	username := "hppoc"
	password := "password"
	host := "127.0.0.1"
	port := "5432"
	dbname := "account-management"

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, username, password, dbname, port)
	fmt.Printf("PgService.NewPgService: dsn = %s\n", dsn)
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Dừng chương trình nếu quá trình kết nối tới database xảy ra lỗi
	if err != nil {
		panic("Failed to connect database")
	}

	db.AutoMigrate(&model.Account{})
	db.AutoMigrate(&model.Transaction{})

	return db

}
