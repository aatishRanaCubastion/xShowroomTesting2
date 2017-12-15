package database

import (
	"github.com/jinzhu/gorm"
	"fmt"
	"log"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

var (
	SQL       *gorm.DB
	databases Info
)

type Type string

const (
	TypeMySQL Type = "MySQL"
)

type Info struct {
	Type  Type
	MySQL MySQLInfo
}

type MySQLInfo struct {
	Username  string
	Password  string
	Name      string
	Hostname  string
	Port      int
	Parameter string
}

func Connect(d Info) {
	var err error

	databases = d

	switch d.Type {
	case TypeMySQL:
		// Connect to MySQL
		SQL, err = gorm.Open("mysql", DSN(d.MySQL))
		if err != nil {
			log.Println("SQL Driver Error", err)
		}

		if err = SQL.DB().Ping(); err != nil {
			log.Println("Database Error", err)
		}
	default:
		log.Println("No registered database in config")
	}
}

func DSN(ci MySQLInfo) string {

	return ci.Username +
		":" +
		ci.Password +
		"@tcp(" +
		ci.Hostname +
		":" +
		fmt.Sprintf("%d", ci.Port) +
		")/" +
		ci.Name + ci.Parameter
}
