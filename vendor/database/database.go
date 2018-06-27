package database

import (
	"github.com/jinzhu/gorm"
	"fmt"
	"log"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

var (
	SQL *gorm.DB
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

func Connect(d Info, from string) {
	var err error

	databases = d

	//(Creates) or (Deletes and Creates) new database as mentioned in present config file
	if (from == "main") {
		createMainDatabase(d)
	}

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

func createMainDatabase(d Info) {
	var err error

	SQL, err = gorm.Open("mysql", "root:password@tcp(127.0.0.1:3306)/")
	if err != nil {
		log.Fatal(err)
	}
	defer SQL.Close()
	rows, _ := SQL.Raw("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", d.MySQL.Name).Rows()
	if rows.Next() {
		newDb := SQL.Exec("DROP DATABASE " + d.MySQL.Name)

		newDb = SQL.Exec("CREATE DATABASE " + d.MySQL.Name)
		if newDb != nil {
			fmt.Println("Database dropped and then created !")
		} else {
			fmt.Println("Database dropped and Error occured in creation !")
		}

	} else {
		newDb := SQL.Exec("CREATE DATABASE " + d.MySQL.Name)
		if newDb != nil {
			fmt.Println("Database not found and thus created !")
		} else {
			fmt.Println("Database not found and Error occured in creation !")
		}
	}

	SQL.Close()
}