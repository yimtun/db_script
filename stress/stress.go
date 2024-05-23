package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	mysqlIP string
	user    string
	passwd  string
	dbname  string
)

func initDB() *gorm.DB {
	//	mysql.Open(cfg.DSN)
	db, err := gorm.Open(mysql.Open(user + ":" + passwd + "@tcp(" + mysqlIP + ")/" + dbname))
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&UserT{})
	return db
}

type UserT struct {
	Id       int64  `gorm:"primaryKey,autoIncrement"`
	Nickname string `gorm:"type=varchar(128)"`
}

func main() {

	cfile := pflag.String("config",
		"config/config.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigType("yaml")
	viper.SetConfigFile(*cfile)
	// 读取配置
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	mysqlIP = viper.GetString("srress.mysql")
	fmt.Println("mysqlIP:", mysqlIP)
	user = viper.GetString("srress.user")
	passwd = viper.GetString("srress.password")
	dbname = viper.GetString("srress.dbname")

	db := initDB()

	// 定时器，每5秒插入一条数据
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		randomNickname := generateRandomNickname()
		user := UserT{
			Nickname: randomNickname,
		}
		err := db.Create(&user).Error
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(db.Error)
}

func generateRandomNickname() string {
	uuid := uuid.New()
	return uuid.String()
}
