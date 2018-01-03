package generator

import (
	"os"
	"fmt"
	. "github.com/dave/jennifer/jen"
	"database"
	"log"
	"strings"
)
/*
func init(){
	error:=os.Mkdir("vendor/"+const_ModelsPath,0777)
	if error!=nil{
		log.Fatal(error)
	}
	error = os.Mkdir("vendor/"+const_MyGraphQlPath,0777)
	if error!=nil{
		log.Fatal(error)
	}
}*/


var const_ConfigPath = "config"
var const_JsonConfigPath = "jsonconfig"
var const_DatabasePath = "database"
var const_ModelsPath = "models"
var const_ControllersPath = "controllers"
var const_MyGraphQlPath = "mygraphql"
var const_ServerPath = "server"
var const_RoutePath = "route"
var const_RouterPath = "router"
var const_UtilsPath = "utils"
var const_GeneratorPath = "generator"
var const_GraphQlPath = "github.com/neelance/graphql-go"
var const_UtilsInt32ToUint = "Int32ToUint"
var const_UtilsStringToUInt = "StringToUInt"
var const_UtilsConvertId = "ConvertId"
var const_UtilsUintToGraphId = "UintToGraphId"
var const_UtilsRuneToGraphId = "RuneToGraphId"
var const_OneToOne = "OneToOne"
var const_OneToMany = "OneToMany"
var const_ManyToOne = "ManyToOne"
var const_ManyToMany = "ManyToMany"

var const_reverse = "_reverse"
var const_normal = "_normal"
var const_self = "_self"
var const_resolver = "_resolver"

type Entity struct {
	ID          int `sql:"AUTO_INCREMENT"`
	Name        string `sql:"type:varchar(30)"  gorm:"column:name;not null;unique"`
	DisplayName string `sql:"type:varchar(30)" gorm:"column:display_name"`
	Columns     []Column `gorm:"ForeignKey:entity_id;AssociationForeignKey:id"` // one to many, has many columns
}

type ColumnType struct {
	ID      int    `sql:"AUTO_INCREMENT"`
	Type    string `sql:"type:varchar(30)"`
	Columns []Column `gorm:"ForeignKey:type_id;AssociationForeignKey:id"` //one to many, has many columns
}

type Column struct {
	ID          int `sql:"AUTO_INCREMENT"`
	Name        string `sql:"type:varchar(30)" gorm:"unique_index:idx_name_entity_id"`
	DisplayName string `sql:"type:varchar(30)"`
	Size        int `sql:"type:int(30)"`
	TypeID      int `sql:"type:int(30)"`
	EntityID    int `sql:"type:int(100)" gorm:"unique_index:idx_name_entity_id"`
	IsNull	    int `sql:"type:int(30)"`
	ColumnType  ColumnType `gorm:"ForeignKey:TypeID"` //belong to (for reverse access)
}

type RelationType struct {
	ID   int `sql:"AUTO_INCREMENT"`
	Name string `sql:"type:varchar(30)"`
}

type Relation struct {
	ID                int `sql:"AUTO_INCREMENT"`
	ParentEntityID    int `sql:"type:int(100)" gorm:"unique_index:idx_all_relation"`
	ParentEntityColID int `sql:"type:int(100)" gorm:"unique_index:idx_all_relation"`
	ChildEntityID     int `sql:"type:int(100)" gorm:"unique_index:idx_all_relation"`
	ChildEntityColID  int `sql:"type:int(100)" gorm:"unique_index:idx_all_relation"`
	InterEntityID     int `sql:"type:int(100)" gorm:"unique_index:idx_all_relation"`
	RelationTypeID    int `sql:"type:int(10)" gorm:"unique_index:idx_all_relation"`

	ParentEntity      Entity `gorm:"ForeignKey:ParentEntityID"`       //belong to
	ChildEntity       Entity `gorm:"ForeignKey:ChildEntityID"`        //belong to
	InterEntity       Entity `gorm:"ForeignKey:InterEntityID"`        //belong to
	ParentColumn      Column `gorm:"ForeignKey:ParentEntityColID"`    //belong to
	ChildColumn       Column `gorm:"ForeignKey:ChildEntityColID"`     //belong to
	RelationType      RelationType `gorm:"ForeignKey:RelationTypeID"` //belong to
}

func (Entity) TableName() string {
	return "c_entity"
}

func (ColumnType) TableName() string {
	return "c_column_type"
}

func (Column) TableName() string {
	return "c_column"
}

func (RelationType) TableName() string {
	return "c_relation_type"
}

func (Relation) TableName() string {
	return "c_relation"
}

type EntityRelation struct {
	Type             string
	SubEntityName    string
	SubEntityColName string
	InterEntity      InterEntity
}

type InterEntity struct {
	TableName  string
	StructName string
}

type EntityRelationMethod struct {
	MethodName       string
	Type             string
	SubEntityName    string
	SubEntityColName string
}

type EntityField struct {
	FieldName string
	FieldType string
}

func GenerateCode(appName string) {

	//fetch all entities
	entities := []Entity{}

	database.SQL.Preload("Columns.ColumnType").
		Find(&entities)

	//print all entities
	//for _, entity := range entitie {
	//	fmt.Println("dsfsd :",entity.ChildEntity.Name , "    ", entity.ParentEntity.Name)
	//	//for _, col := range entity.Columns {
	//	//	fmt.Print("\t", col.Name, " ", col.ColumnType.Type, "(", col.Size, ")\n")
	//	//}
	//}

	allModels := make([]string, 0)
	//creating entity structures
	for _, entity := range entities {
		allModels = append(allModels, createEntities(entity, database.SQL))
	}

	//write root resolver
	//create resolver.go
	fileResolver, err := os.Create("vendor/" + const_MyGraphQlPath + "/resolver.go")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer fileResolver.Close()
	//created file
	appResolver := NewFile(const_MyGraphQlPath)
	createResolver(appResolver, allModels)

	//write root schema
	//create schema.go
	fileSchema, err := os.Create("vendor/" + const_MyGraphQlPath + "/schema.go")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer fileSchema.Close()
	//created file
	appSchema := NewFile(const_MyGraphQlPath)
	createSchema(appSchema, entities,database.SQL)

	//create appName.go
	fileMain, err := os.Create(appName + ".go")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer fileMain.Close()
	//created file
	appMain := NewFile("main")

	//write all code
	createAppMain(appMain, allModels)

	//flush xShowroom.go
	fmt.Fprintf(fileResolver, "%#v", appResolver)
	fmt.Fprintf(fileSchema, "%#v", appSchema)
	fmt.Fprintf(fileMain, "%#v", appMain)
	fmt.Println("=========================")
	fmt.Println(appName, "generated!!!")
}

//xShowroom generation methods
func createAppMain(appMain *File, allModels []string) {

	//create an instance of configuration
	appMain.Var().Id("conf").Op("= &").Qual("config", "Configuration{}")

	createAppMainInitMethod(appMain)

	createAppMainMainMethod(appMain, allModels)
}

func createAppMainInitMethod(appMain *File) {
	//add init method in appMain.go
	appMain.Func().Id("init").Params().Block(
		Comment(" Use all cpu cores"),
		Qual("runtime", "GOMAXPROCS").Call(Qual("runtime", "NumCPU").Call()),
	)
}

func createAppMainMainMethod(appMain *File, allModels []string) {

	//add main method in appMain.go
	appMain.Func().Id("main").Params().Block(

		Comment("Load the configuration file"),
		Qual(const_JsonConfigPath, "Load").Call(
			Lit(const_ConfigPath).
				Op("+").
				Id("string").
				Op("(").
				Qual("os", "PathSeparator").
				Op(")").
				Op("+").
				Lit("config.json"),
			Id("conf")),

		Empty(),

		Comment("Connect to database"),
		Qual(const_DatabasePath, "Connect").Call(
			Id("conf").Op(".").Id("Database"),
		),

		Empty(),

		Comment("Create schema"),
		Id("schema").Op(":=").Qual(const_GraphQlPath, "MustParseSchema").Call(Qual(const_MyGraphQlPath, "Schema"), Op("&").Qual(const_MyGraphQlPath, "Resolver{}")),

		Empty(),

		Comment("Load the controller routes"),
		Qual(const_ControllersPath, "Load").Call(Id("schema")),

		Empty(),

		Comment("Auto migrate all models"),
		Qual(const_DatabasePath, "SQL.AutoMigrate").CallFunc(func(g *Group) {
			for _, value := range allModels {
				g.Id("&").Qual(const_ModelsPath, value + "{}")
			}
		}),

		Empty(),

		Comment("Start the listener"),
		Qual(const_ServerPath, "Run").Call(
			Qual(const_RoutePath, "LoadHTTP").Call(),
			Qual(const_RoutePath, "LoadHTTPS").Call(),
			Id("conf").Op(".").Id("Server"),
		),
	)
}


//models generation methods

func mapColumnTypesGorm(col Column, g *Group) EntityField {

	entityField := EntityField{}
	entityField.FieldName = col.Name

	if col.ColumnType.Type == "int" {
		entityField.FieldType = "uint"
		finalId := snakeCaseToCamelCase(col.Name) + " uint" + " `gorm:\"column:" + col.Name + "\" json:\"" + col.Name + ",omitempty\"`"
		g.Id(finalId)
	} else if col.ColumnType.Type == "varchar" {
		entityField.FieldType = "string"
		finalId := snakeCaseToCamelCase(col.Name) + " string" + " `gorm:\"column:" + col.Name + "\" json:\"" + col.Name + ",omitempty\"`"
		g.Id(finalId)
	} else {
		entityField.FieldType = "string"
		g.Id(snakeCaseToCamelCase(col.Name)).String() //default string
	}
	return entityField
}

func mapColumnTypesResolver(col Column, g *Group, isInput bool) {

	var fieldName string
	fieldNameLower := strings.ToLower(col.Name)
	fieldNameCaps := snakeCaseToCamelCase(col.Name)
	isNull:=" "
	if isInput == true && col.IsNull==1{
		isNull=" *"
	}
	if isInput {
		fieldName = fieldNameCaps
	} else {
		fieldName = fieldNameLower
	}

	if fieldName == "id" || fieldName == "ID" || fieldName == "Id" {

		finalId := fieldName
		if isInput {
			finalId = fieldName+" *"
		}

		g.Id(finalId).Qual(const_GraphQlPath, "ID")
		return
	}

	if isInput == false {
		if col.ColumnType.Type == "int" {
			finalId := fieldName + " int32"
			g.Id(finalId)
		} else if col.ColumnType.Type == "varchar" {
			finalId := fieldName + " string"
			g.Id(finalId)
		} else {
			g.Id(fieldName).String() //default string
		}

	}else if isInput == true {
	if col.ColumnType.Type == "int" {
		finalId := fieldName +isNull+ "int32"
		g.Id(finalId)
	} else if col.ColumnType.Type == "varchar" && strings.HasSuffix(col.Name,"_type") {
		finalId := fieldName + " *string"
		g.Id(finalId)
	} else if col.ColumnType.Type == "varchar"{
		finalId := fieldName + isNull+"string"
		g.Id(finalId)
	} else {
		finalId := fieldName +isNull+"string"//default string
		g.Id(finalId)
	}
}
	return
}



//helper methods
func snakeCaseToCamelCase(inputUnderScoreStr string) (camelCase string) {
	//snake_case to camelCase

	isToUpper := false

	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if isToUpper {
				camelCase += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return

}

func handlerRequestParams() (Code, Code) {
	return Id("w").Qual("net/http", "ResponseWriter"), Id("req").Op("*").Qual("net/http", "Request")
}

func setJsonHeader() Code {
	return Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
}

//func sendResponse(statusCode uint, statusMsg string, data interface{}) Code {
//	return Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
//		Op("{").
//		Lit(statusCode).Op(",").
//		Lit(statusMsg).Op(",").
//		Lit(data).
//		Op("}"))
//}

func sendResponse(data interface{}) Code {
	return Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Lit(data))
}
