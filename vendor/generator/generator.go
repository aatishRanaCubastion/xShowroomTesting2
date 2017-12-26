package generator

import (
	"os"
	"fmt"
	. "github.com/dave/jennifer/jen"
	"database"
	"log"
	"github.com/jinzhu/gorm"
	"strings"
	"bytes"
	"strconv"
	u "utils"
)

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

var const_UtilsStringToUInt = "StringToUInt"
var const_UtilsConvertId = "ConvertId"
var const_UtilsUintToGraphId = "UintToGraphId"

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
	//InterEntityDispName  string
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
	createSchema(appSchema, entities)

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

func createResolver(resolverFile *File, allModels []string) {

	resolverFile.Type().Id("Resolver").Struct()

	for _, val := range allModels {

		//writing root query resolvers
		resolverFile.Empty()
		resolverFile.Comment("query resolver for " + val)
		resolverFile.Func().Params(Id("r").Id(" *Resolver")).Id(val).Params(Id("args").StructFunc(func(g *Group) {
			g.Id("ID").Qual(const_GraphQlPath, "ID")
		})).Params(Id("[] *" + strings.ToLower(val) + "Resolver")).
			BlockFunc(func(g *Group) {
			g.Return(Qual("", "Resolve" + val)).Call(Id("args"))
		})

		// uncomment when create and delete resolvers are done

		//writing Create/Update root mutation resolvers
		//resolverFile.Empty()
		//resolverFile.Comment("create resolver for " + val)
		//resolverFile.Func().Params(Id("r").Id(" *Resolver")).Id("Create"+val).Params(Id("args").StructFunc(func(g *Group) {
		//	g.Id("ID").Qual(const_GraphQlPath, "ID")
		//})).Params(Id("*" + strings.ToLower(val) + "Resolver")).
		//	BlockFunc(func(g *Group) {
		//	g.Return(Qual("", "ResolveCreate"+val)).Call(Id("args"))
		//})

		//writing Delete root mutation resolvers
		resolverFile.Empty()
		resolverFile.Comment("delete resolver for " + val)
		resolverFile.Func().Params(Id("r").Id(" *Resolver")).Id("Delete" + val).Params(Id("args").StructFunc(func(g *Group) {
			g.Id("ID").Qual(const_GraphQlPath, "ID")
			//})).Params(Id("*" + strings.ToLower(val) + "Resolver")).
		})).Params(Id("*int")).
			BlockFunc(func(g *Group) {
			g.Return(Qual("", "ResolveDelete" + val)).Call(Id("args"))
		})

	}
}

func createSchema(schemaFile *File, allEntities []Entity) {
 
	sS := ""
	//write root schema
	u.SAppend(&sS, "\n")
	u.SAppend(&sS, "schema {\n")
	u.SAppend(&sS, "\tquery: Query\n")
	//u.SAppend(&sS, "\tmutation: Mutation\n")
	u.SAppend(&sS, "}\n\n")

	//write query schema
	u.SAppend(&sS, "# The query type, represents all of the entry points into our object graph\n")
	u.SAppend(&sS, "type Query {\n")
	for _, val := range allEntities {
		entityNameLower := strings.ToLower(val.DisplayName)
		entityNameCaps := snakeCaseToCamelCase(val.DisplayName)
		u.SAppend(&sS, "\t" + entityNameLower + "(id: ID!) : [" + entityNameCaps + "]!\n")
	}
	u.SAppend(&sS, "}\n\n")

	//uncomment when mutation resolvers are done

	////write mutation schema
	u.SAppend(&sS, "# The mutation type, represents all updates we can make to our data\n")
	u.SAppend(&sS, "type Mutation {\n")
	u.SAppend(&sS, "# Create\n")
	for _, val := range allEntities {
		entityNameLower := strings.ToLower(val.DisplayName)
		entityNameCaps := snakeCaseToCamelCase(val.DisplayName)
		u.SAppend(&sS, "\tcreate" + entityNameCaps + "(" + entityNameLower + ": " + entityNameCaps + "Input!) : [" + entityNameCaps + "]!\n")
	}
	u.SAppend(&sS, "# Delete\n")
	for _, val := range allEntities {
		entityNameCaps := snakeCaseToCamelCase(val.DisplayName)
		u.SAppend(&sS, "\t" + "delete" + entityNameCaps + "(id: ID!,cascadeDelete: Boolean) : Int \n")
	}
	u.SAppend(&sS, "}\n\n")

	for _, val := range allEntities {
		//entityNameLower := strings.ToLower(val.DisplayName)
		entityNameCaps := snakeCaseToCamelCase(val.DisplayName)

		u.SAppend(&sS, "type " + entityNameCaps + " {\n")
		for _, col := range val.Columns {
			fieldType := "String"
			if col.ColumnType.Type == "int" {
				fieldType = "Int"
			}
			if col.Name == "id" {
				fieldType = "ID"
			}

			u.SAppend(&sS, "\t" + col.Name + ": " + fieldType + "!\n")
		}
		u.SAppend(&sS, "}\n")

		u.SAppend(&sS, "input " + entityNameCaps + "Input {\n")
		for _, col := range val.Columns {

			fieldType := "String"
			if col.ColumnType.Type == "int" {
				fieldType = "Int"
			}
			if col.Name == "id" {
				fieldType = "ID"
			}

			u.SAppend(&sS, "\t" + col.Name + ": " + fieldType + "!\n")
		}
		u.SAppend(&sS, "}\n\n")
	} 

	schemaFile.Var().Id("Schema").Op("=").Id("`" + sS + "`")
}

//models generation methods
func createEntities(entity Entity, db *gorm.DB) string {

	// create entity name from table
	entityName := snakeCaseToCamelCase(entity.DisplayName)

	//childEntity := []Relation{}
	//database.SQL.Preload("ChildEntity").Preload("InterEntity").
	//	Preload("ParentEntity").Where("parent_entity_id=(?)",entity.ID).
	//	Find(&childEntity)

	//entity relations stored to generate routes and their methods for each sub entities ((parent to child) and (child to parent))
	entityRelationsForEachEndpoint := []EntityRelation{}

	//entity relations stored to generate one route to access all sub entities depending on query params(parent to child only)
	entityRelationsForAllEndpoint := []EntityRelation{}

	//create entity file in models sub directory
	//fmt.Print("dfsfs ::","vendor/" + const_ModelsPath + "/" + strings.ToLower(entityName) + ".go")
	fileModel, err := os.Create("vendor/" + const_ModelsPath + "/" + strings.ToLower(entityName) + ".go")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer fileModel.Close()

	//create controller entity file in controller sub directory
	fileController, err2 := os.Create("vendor/" + const_ControllersPath + "/" + strings.ToLower(entityName) + ".go")
	if err2 != nil {
		log.Fatal("Cannot create file", err2)
	}
	defer fileController.Close()

	//create resolver entity file in controller sub directory
	fileResolver, err3 := os.Create("vendor/" + const_MyGraphQlPath + "/" + strings.ToLower(entityName) + const_resolver + ".go")
	if err3 != nil {
		log.Fatal("Cannot create file", err3)
	}
	defer fileResolver.Close()

	//set package as "models"
	modelFile := NewFile(const_ModelsPath)

	//set package as "models"
	controllerFile := NewFile(const_ControllersPath)

	//set package as "models"
	resolverFile := NewFile(const_MyGraphQlPath)

	//fetch relations of this entity matching parent
	relationsParent := []Relation{}
	db.Preload("InterEntity").
		Preload("ChildEntity").
		Preload("ChildColumn").
		Preload("ParentColumn").
		Where("parent_entity_id=?", entity.ID).
		Find(&relationsParent)

	//fetch relations of this entity matching child
	relationsChild := []Relation{}
	db.Preload("InterEntity").
		Preload("ParentEntity").
		Preload("ChildColumn").
		Preload("ParentColumn").
		Where("child_entity_id=?", entity.ID).
		Find(&relationsChild)

	entityFields := []EntityField{}

	//write structure for entity
	modelFile.Type().Id(entityName).StructFunc(func(g *Group) {

		//write primitive fields
		for _, column := range entity.Columns {
			entityFields = append(entityFields, mapColumnTypesGorm(column, g))
		}

		//fmt.Println(entityFields)
		//write composite fields while looking at parent
		for _, relation := range relationsParent {
			interName := relation.InterEntity.Name
			interDispName := relation.InterEntity.DisplayName

			interEntity := InterEntity{TableName:interName, StructName:interDispName}

			//fmt.Println("parent ", relation.InterEntity.Name)
			name := snakeCaseToCamelCase(relation.ChildEntity.DisplayName)

			childName := string(relation.ChildColumn.Name)
			parentName := string(relation.ParentColumn.Name)
                   //    fmt.Println("entity name :",entityName,"inter :",interEntity)
			d := " "
			relType := "_normal"
			if entityName == name {
				d = "*" //if name and entityName are same, its a self join, so add *
				relType = "_self"
			}  

			switch relation.RelationTypeID {
			case 1: //one to one
				relationName := name
				finalId := relationName + " " + d + name + " `gorm:\"ForeignKey:" + childName + ";AssociationForeignKey:" + parentName + "\" json:\"" + relation.ChildEntity.DisplayName + ",omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"OneToOne" + relType, name, childName, InterEntity{}})
				//entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToOne" + relType, relationName, childName, InterEntity{}})
				g.Id(finalId)
			case 2: //one to many
				relationName := name
				//entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", relationName, childName, InterEntity{}})
				relationName = name + "s"
				finalId := relationName + " []" + name + " `gorm:\"ForeignKey:" + childName + ";AssociationForeignKey:" + parentName + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"OneToMany", name, childName, InterEntity{}})
				g.Id(finalId)
			case 3: //many to many
				relationName := name
				entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", "", childName, interEntity})
				relationName = name + "s"
				//finalId := relationName + " []" + name + " `gorm:\"many2many:" + relation.InterEntity.Name + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				finalId := relationName + " []" + name + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				g.Id(finalId)
				//finalInterId := interName + " []" + interName + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				//g.Id(finalInterId)

				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"ManyToMany", name, childName, InterEntity{}})
			}
		}

		//write composite fields while looking at child
		for _, relation := range relationsChild {
			interName := relation.InterEntity.Name
			interDispName := relation.InterEntity.DisplayName

			interEntity := InterEntity{TableName:interName, StructName:interDispName}

			name := snakeCaseToCamelCase(relation.ParentEntity.DisplayName)
			childName := string(relation.ChildColumn.Name)
		//	fmt.Println("entity child name :",entityName,"inter :",interEntity)
			switch relation.RelationTypeID {
			case 1: //ont to one
				// means current entity's one item belongs to
				if name != entityName {
					// if check to exclude self join
					entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{const_OneToOne + const_reverse, name, childName, InterEntity{}})
				}
			case 2: //one to many
				// means current entity's many items belongs to
				finalId := name + " " + name + " `gorm:\"ForeignKey:" + snakeCaseToCamelCase(childName) + "\" json:\"" + name + ",omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{const_ManyToOne, name, childName, InterEntity{}})
				g.Id(finalId)
			case 3: //many to many
				// add two record in relation table to create many to many or uncomment this and add relation here
				relationName := name
				entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", "", childName, interEntity})
				relationName = name + "s"
				//finalId := relationName + " []" + name + " `gorm:\"many2many:" + relation.InterEntity.Name + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				finalId := relationName + " []" + name + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				g.Id(finalId)
				//finalInterId := interName + " []" + interName + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				//g.Id(finalInterId)
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"ManyToMany", name, childName, InterEntity{}})
			//fmt.Println("\t\t many to many " + relation.InterEntity.DisplayName + " for " + entityName + " from child")
			}
		}
	})

	//write table name method for our struct
	modelFile.Func().Params(Id(snakeCaseToCamelCase(entity.DisplayName))).Id("TableName").Params().String().Block(
		Return(Lit(entity.Name)),
	)

	getAllMethodName := "GetAll" + entityName + "s"
	getByIdMethodName := "Get" + entityName
	postMethodName := "Post" + entityName
	putMethodName := "Put" + entityName
	deleteMethodName := "Delete" + entityName

	allMethodName := "GetAll" + entityName + "sSubEntities"
	allMethodExist := false

	specialMethods := []EntityRelationMethod{}

	//write routes in init method
	controllerFile.Comment("Routes related to " + entityName)
	controllerFile.Func().Id("init").Params().BlockFunc(func(g *Group) {

		g.Empty()
		g.Comment("Standard routes")
		g.Qual(const_RouterPath, "Get").Call(Lit("/" + strings.ToLower(entityName)), Id(getAllMethodName))
		g.Qual(const_RouterPath, "Get").Call(Lit("/" + strings.ToLower(entityName) + "/:id"), Id(getByIdMethodName))
		g.Qual(const_RouterPath, "Post").Call(Lit("/" + strings.ToLower(entityName)), Id(postMethodName))
		g.Qual(const_RouterPath, "Put").Call(Lit("/" + strings.ToLower(entityName) + "/:id"), Id(putMethodName))
		g.Qual(const_RouterPath, "Delete").Call(Lit("/" + strings.ToLower(entityName) + "/:id"), Id(deleteMethodName))

		//if len(entityRelationsForEachEndpoint) > 0 {
		//	g.Empty()
		//	g.Comment("Sub entities routes")
		//	for _, entRel := range entityRelationsForEachEndpoint {
		//
		//		if entRel.Type == const_OneToMany {
		//			methodName := "Get" + entityName + entRel.SubEntityName + "s"
		//			specialMethods = append(specialMethods, EntityRelationMethod{methodName, entRel.Type, entRel.SubEntityName, entRel.SubEntityColName})
		//			g.Empty()
		//			g.Comment("has many")
		//			g.Qual(const_RouterPath, "Get").Call(Lit("/"+strings.ToLower(entityName)+"/:id/"+strings.ToLower(entRel.SubEntityName+"s")), Id(methodName))
		//		} else if entRel.Type == const_OneToOne+const_normal || entRel.Type == const_OneToOne+const_self || entRel.Type == const_OneToOne+const_reverse {
		//			methodName := "Get" + entityName + entRel.SubEntityName
		//			specialMethods = append(specialMethods, EntityRelationMethod{methodName, entRel.Type, entRel.SubEntityName, entRel.SubEntityColName})
		//			g.Empty()
		//			g.Comment("has one")
		//			g.Qual(const_RouterPath, "Get").Call(Lit("/"+strings.ToLower(entityName)+"/:id/"+strings.ToLower(entRel.SubEntityName)), Id(methodName))
		//		} else if entRel.Type == const_ManyToOne {
		//			methodName := "Get" + entityName + entRel.SubEntityName + ""
		//			specialMethods = append(specialMethods, EntityRelationMethod{methodName, entRel.Type, entRel.SubEntityName, entRel.SubEntityColName})
		//			g.Empty()
		//			g.Comment("belongs to")
		//			g.Qual(const_RouterPath, "Get").Call(Lit("/"+strings.ToLower(entityName)+"/:id/"+strings.ToLower(entRel.SubEntityName)), Id(methodName))
		//		} else if entRel.Type == const_ManyToMany {
		//			methodName := "Get" + entityName + entRel.SubEntityName + "s"
		//			specialMethods = append(specialMethods, EntityRelationMethod{methodName, entRel.Type, entRel.SubEntityName, entRel.SubEntityColName})
		//			g.Empty()
		//			g.Comment("has many to many")
		//			g.Qual(const_RouterPath, "Get").Call(Lit("/"+strings.ToLower(entityName)+"/:id/"+strings.ToLower(entRel.SubEntityName)), Id(methodName))
		//		}
		//
		//	}
		//}

		//if len(entityRelationsForAllEndpoint) > 0 {
		//	allMethodExist = true
		//	g.Empty()
		//	g.Comment("extra route")
		//	g.Qual(const_RouterPath, "Get").Call(Lit("/"+strings.ToLower(entityName)+"/:id/all"), Id(allMethodName))
		//}
	})

	//write resolver
	createEntitiesResolver(resolverFile, entityName, entity, entityRelationsForAllEndpoint)

	createEntitiesChildSlice(modelFile, entityName, entityRelationsForAllEndpoint)

	createEntitiesGetAllMethod(modelFile, entityName, getAllMethodName, controllerFile)

	createEntitiesGetMethod(modelFile, entityName, getByIdMethodName, controllerFile)

	createEntitiesPostMethod(modelFile, entityName, postMethodName, entityFields, controllerFile)

	createEntitiesPutMethod(modelFile, entityName, putMethodName, controllerFile)

	createEntitiesDeleteMethod(modelFile, entityName, deleteMethodName, controllerFile)

	if len(specialMethods) > 0 {
		for _, method := range specialMethods {
			modelFile.Empty()
			modelFile.Func().Id(method.MethodName).Params(handlerRequestParams()).BlockFunc(func(g *Group) {
				g.Empty()
				g.Comment("Get the parameter id")
				g.Id("params").Op(":=").Qual(const_RouterPath, "Params").Call(Id("req"))
				g.Id("ID").Op(",").Id("_").Op(":=").Qual("strconv", "ParseUint").Call(
					Qual("", "params.ByName").Call(Lit("id")),
					Id("10"),
					Id("0"),
				)

				if method.Type == const_OneToMany || method.Type == const_OneToOne + const_normal {
					g.Id("data").Op(":= []").Id(method.SubEntityName).Id("{}")
					g.Qual(const_DatabasePath, "SQL.Find").Call(Id("&").Id("data"), Lit(" " + method.SubEntityColName + " = ?"), Id("ID"))
					g.Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
					g.Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
						Op("{").
						Id("2000").Op(",").
						Lit("Data fetched successfully").Op(",").
						Id("data").
						Op("}"))
				}

				if method.Type == const_ManyToOne || method.Type == const_OneToOne + const_reverse {
					g.Id(strings.ToLower(entityName)).Op(":=").Id(entityName).Op("{").Id("Id").Op(":").Id("uint(").Id("ID").Op(")}")

					g.Id("data").Op(":= ").Id(method.SubEntityName).Id("{}")
					g.Qual(const_DatabasePath, "SQL.Find").Call(
						Id("&").Id("data"), Lit(" id = (?)"),
						Qual(const_DatabasePath, "SQL.Select").Call(Lit(method.SubEntityColName)).Op(".").Id("First").Call(Id("&").Id(strings.ToLower(entityName))).Op(".").Id("QueryExpr").Call(),
					)
					g.Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
					g.Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
						Op("{").
						Id("2000").Op(",").
						Lit("Data fetched successfully").Op(",").
						Id("data").
						Op("}"))
				}

				if method.Type == const_OneToOne + const_self {
					g.Id("data").Op(":= ").Id(method.SubEntityName).Id("{}")
					g.Qual(const_DatabasePath, "SQL.Find").Call(Id("&").Id("data"), Lit(" " + method.SubEntityColName + " = ?"), Id("ID"))
					g.Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
					g.Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
						Op("{").
						Id("2000").Op(",").
						Lit("Data fetched successfully").Op(",").
						Id("data").
						Op("}"))
				}

				if method.Type == const_ManyToMany {

					relation := method.SubEntityName + "s"

					g.Id("data").Op(":=").Id(entityName).Id("{}")
					g.Qual(const_DatabasePath, "SQL.Find").Call(Id("&").Id("data"), Id("ID"))
					g.Qual(const_DatabasePath, "SQL.Model").Call(Id("&").Id("data")).Op(".").Id("Association").Call(Lit(relation)).
						Op(".").Id("Find").Call(Id("&").Id("data").Op(".").Id(relation))
					g.Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
					g.Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
						Op("{").
						Id("2000").Op(",").
						Lit("Data fetched successfully").Op(",").
						Id("data").
						Op("}"))
				}
			})
		}
	}

	if allMethodExist {
		createEntitiesAllChildMethod(modelFile, entityName, allMethodName, entityRelationsForAllEndpoint)
	}

	fmt.Fprintf(fileModel, "%#v", modelFile)
	fmt.Fprintf(fileController, "%#v", controllerFile)
	fmt.Fprintf(fileResolver, "%#v", resolverFile)

	fmt.Println(entityName + " generated")
	return entityName
}

/*func m2mdel(resolverFile *File, entityName string,entity Entity,entityRelationsForAllEndpoint []EntityRelation){
	var allInterRelation []string
	var flag int
	for _, entity := range entityRelationsForAllEndpoint {
		for _,v:=range allInterRelation{
			if entity.InterEntity.StructName==v{
				flag=1
			}

		}

		if flag!=1{
			allInterRelation=append(allInterRelation,entity.InterEntity.StructName)
		}

	}
	fmt.Println("dsfsd :",allInterRelation)

	if allInterRelation!=nil{

		For(Id("_,v:=").Range().Id("models." + entityName + "InterRelation")).Block(
				//Id("temp").Op(":=").Lit("Map").Id("+v"),
				//Var().Id("interData models."+allInterRelation),
				//Id("dataType").Op(":=").Lit("[]models.").Id("+v.StructName"),
                               // Var().Id("interData").Id("dataType"),
				Id("temp1").Op(":=").Lit("ResolveDelete").Id("+v.StructName"),
			//	Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
				Qual(const_DatabasePath, "SQL.Model").Call(Lit("models.").Id("+v.StructName+").Lit("{}")).Dot("Joins").Call(Lit("inner join").Id("+data.TableName()+").Lit("on").Id("+data.TableName()+").Lit(".id=").Id("+v.TableName+").Lit(".").Id("+").Qual("strings","TrimPrefix").Call(Id("data.TableName()"),Lit("x_")).Id("+").Lit("_id")).Dot("Where").Call(Qual("strings","TrimPrefix").Call(Id("data.TableName()"),Lit("x_")).Id("+").Lit("_id").Id("+").Lit("=(?)"),Id("args.ID")).Dot("Find").Call(Id("&interData")),

				Id("delId").Op(":=").Id("interData"),
				For(Id("_,v1:=").Range().Id("delId")).Block(

					If(Id("v1").Op("!=").Nil()).Block(
						Id("count++"),

						Id("args.ID").Op("=").Id("v1.Id"),
						Id("temp1").Call(Id("args")),
						//Id("response").Op("=").Id("count"),
					),
				),

			)

	}

}*/

func createEntitiesResolver(resolverFile *File, entityName string, entity Entity, entityRelationsForAllEndpoint []EntityRelation) {

	var allInterRelation []string
	//fmt.Println("test",len(allInterRelation))
	var flag int
	for _, entity := range entityRelationsForAllEndpoint {
		for _, v := range allInterRelation {
			if entity.InterEntity.StructName == v {
				flag = 1
			}

		}

		if flag != 1 {
			allInterRelation = append(allInterRelation, entity.InterEntity.StructName)
		}

	}
//	fmt.Println("dsfsd :", allInterRelation)

	entityNameLower := strings.ToLower(entityName)
	resolverFile.Comment("Struct for graphql")
	resolverFile.Type().Id(entityNameLower).StructFunc(func(g *Group) {
		//write primitive fields
		for _, column := range entity.Columns {
			mapColumnTypesResolver(column, g, false)
		}
	})
	resolverFile.Empty()
	resolverFile.Comment("Struct for upserting")
	resolverFile.Type().Id(entityNameLower + "Input").StructFunc(func(g *Group) {
		//write primitive fields
		for _, column := range entity.Columns {
			mapColumnTypesResolver(column, g, true)
		}
	})
	resolverFile.Empty()
	resolverFile.Comment("Struct for response")
	resolverFile.Type().Id(entityNameLower + "Resolver").StructFunc(func(g *Group) {
		g.Id(entityNameLower).Id(" *").Id(entityNameLower)
	})
	resolverFile.Empty()

	resolverFile.Comment("For Query")
	resolverFile.Func().Id("Resolve" + entityName).Params(Id("args").StructFunc(func(g *Group) {
		g.Id("ID").Qual(const_GraphQlPath, "ID")
	})).Params(Id("response []*").Id(entityNameLower + "Resolver")).BlockFunc(func(g *Group) {
		g.If(Id("args").Op(".").Id("ID").Op("!=").Lit("")).BlockFunc(func(h *Group) {
			h.Id("response").Op("=").Qual("", "append").Call(
				Id("response"),
				Op("&").Id(entityNameLower + "Resolver").Values(Dict{
					Id(entityNameLower): Qual("", "Map" + entityName).Call(
						Qual(const_ModelsPath, "Get" + entityName).Call(
							Qual(const_UtilsPath, const_UtilsConvertId).Call(
								Id("args.ID"),
							),
						),
					),
				}),
			)
			h.Return(Id("response"))
		})
		g.For(Id("_").Op(",").Id("val").Op(":=").Id("range").Qual(const_ModelsPath, "GetAll" + entityName + "s").Call()).BlockFunc(func(h *Group) {
			h.Id("response").Op("=").Qual("", "append").Call(
				Id("response"),
				Op("&").Id(entityNameLower + "Resolver").Values(Dict{
					Id(entityNameLower): Qual("", "Map" + entityName).Call(
						Id("val"),
					),
				}),
			)
		})
		g.Return(Id("response"))
	})
	resolverFile.Empty()

	resolverFile.Comment("For Delete")
	resolverFile.Func().Id("ResolveDelete" + entityName).Params(Id("args").StructFunc(func(g *Group) {

		g.Id("ID").Qual(const_GraphQlPath, "ID")
		g.Id("cascadeDelete").Bool()
	})).Params(Id("response *").Int()).BlockFunc(func(g *Group) {

		g.If(Id("args").Op(".").Id("cascadeDelete").Op("==").Lit("")).BlockFunc(func(h *Group) {
			h.Id("args").Op(".").Id("cascadeDelete").Op("=").False()
		})

		resolverFile.Empty()
		resolverFile.Empty()

		g.Var().Id("del").Bool()
		g.Var().Id("count").Int()

		g.If(Id("len").Call(Id("models." + entityName + "Children")).Op("==").Lit(0).Op("&&").Id("len").Call(Id("models." + entityName + "InterRelation")).Op("==").Lit(0)).Block(
			Id("del").Op("=").Qual(const_ModelsPath, "Delete" + entityName).Call(
				Qual(const_UtilsPath, const_UtilsConvertId).Call(
					Id("args.ID"),
				),
				//Id("args.cascadeDelete"),
			),
			If().Id("del").Op("==").True().Block(
				Id("count++"),
			),
			Id("response").Op("=").Id("&count"),

			Return(Id("response")),
		)
		g.If(Id("args.cascadeDelete").Op("==").True()).BlockFunc(func(h *Group) {
			h.Var().Id("data models." + entityName)

			h.For(Id("_,v:=").Range().Id("models." + entityName + "Children")).Block(
				//Id("temp").Op(":=").Lit("Map").Id("+v"),

				//Id("temp1").Op(":=").Lit("models.Delete").Id("+v"),
				Id("temp2").Op(":=").Lit("ResolveDelete").Id("+v"),
				Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),

				Id("delId").Op(":=").Lit("data.").Id("+ v +").Lit(".id"),

				//Id("del").Op("=").Id("temp1").Call(                            //delete child from model
				//	Qual(const_UtilsPath, const_UtilsConvertId).Call(
				//		Id("delId"),
				//	),
				//	//Id("args.cascadeDelete"),
				//),
				For(Id("_,v1:=").Range().Id("delId")).Block(

					If(Id("v1").Op("!=").Nil()).Block(
						Id("count++"),

						Id("args.ID").Op("=").Id("v1"),
						Id("temp2").Call(Id("args")),
						//Id("response").Op("=").Id("count"),
					),
				),
			)
			fmt.Println("len :",entityName,len(allInterRelation),"inter :",allInterRelation)

				for _,v:=range allInterRelation{
					h.For(Id("_,v:=").Range().Id("models." + entityName + "InterRelation")).Block(
						//Id("temp").Op(":=").Lit("Map").Id("+v"),
						Var().Id("interData []models."+v),
						//Id("dataType").Op(":=").Lit("[]models.").Id("+v.StructName"),
						// Var().Id("interData").Id("dataType"),
						Id("temp1").Op(":=").Lit("ResolveDelete").Id("+v.StructName"),
						//	Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
						Qual(const_DatabasePath, "SQL.Model").Call(Lit("models.").Id("+v.StructName+").Lit("{}")).Dot("Joins").Call(Lit("inner join").Id("+data.TableName()+").Lit("on").Id("+data.TableName()+").Lit(".id=").Id("+v.TableName+").Lit(".").Id("+").Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id")).Dot("Where").Call(Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id").Id("+").Lit("=(?)"), Id("args.ID")).Dot("Find").Call(Id("&interData")),

						Id("delId").Op(":=").Id("interData"),
						For(Id("_,v1:=").Range().Id("delId")).Block(

							If(Id("v1").Op("!=").Nil()).Block(
								Id("count++"),

								Id("args.ID").Op("=").Id("v1.Id"),
								Id("temp1").Call(Id("args")),
								//Id("response").Op("=").Id("count"),
							),
						),

					)
				}


			//If(Id("del").Op("==").True()).Block(

			h.Id("del").Op("=").Qual(const_ModelsPath, "Delete" + entityName).Call(
				Qual(const_UtilsPath, const_UtilsConvertId).Call(
					Id("args.ID"),
				),
				//Id("args.cascadeDelete"),
			)
			h.Id("count++")
			h.Id("response").Op("=").Id("&count")

			h.Return(Id("response").Op("+1"))
			//),
			//	Else().Block(
			//	Id("del").Op("=").False(),
			//	Id("response").Op("=").Id("&del"),
			//	Return(Id("response")),
			//),

		})

		g.Var().Id("flag").Int()
		g.Var().Id("data").Id("models." + entityName)

		g.For(Id("_,v").Op(":=").Range().Id("models." + entityName + "Children")).Block(
			Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
			Id("temp").Op(":=").Lit("data.").Id("+v"),
			If(Id("temp").Op("==").Lit("")).Block(
				Id("flag=1"),
			),

		)

		for _,v:=range allInterRelation {

			g.For(Id("_,v").Op(":=").Range().Id("models." + entityName + "InterRelation")).Block(
				Var().Id("interData []models."+v),

				//Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
				Qual(const_DatabasePath, "SQL.Model").Call(Lit("models.").Id("+v.StructName+").Lit("{}")).Dot("Joins").Call(Lit("inner join").Id("+data.TableName()+").Lit("on").Id("+data.TableName()+").Lit(".id=").Id("+v.TableName+").Lit(".").Id("+").Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id")).Dot("Where").Call(Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id").Id("+").Lit("=(?)"), Id("args.ID")).Dot("Find").Call(Id("&interData")),

				//Id("temp").Op(":=").Lit("data.+v"),
				If(Id("len(interData)").Op("==").Lit(0)).Block(
					Id("flag=1"),
				),

			)
		}
		g.If(Id("flag").Op("==").Lit(1)).Block(
			Id("del").Op("=").Qual(const_ModelsPath, "Delete" + entityName).Call(
				Qual(const_UtilsPath, const_UtilsConvertId).Call(
					Id("args.ID"),
				),
				//Id("args.cascadeDelete"),
			),
			Id("count++"),
			Id("response").Op("=").Id("&count"),


		).Else().Block(
			Comment("show error"),
			Id("del").Op("=").False(),
			Id("response").Op("=").Id("&count"),
		)

		g.Return(Id("response"))
	})

	resolverFile.Empty()
	resolverFile.Comment("Fields resolvers")
	//scalar types fields
	for _, column := range entity.Columns {

		fieldNameLower := strings.ToLower(column.Name)
		fieldNameCaps := snakeCaseToCamelCase(column.Name)

		if fieldNameLower == "id" {
			resolverFile.Func().Params(Id("r *").Id(entityNameLower + "Resolver")).Id(fieldNameCaps).Params().Params(Qual(const_GraphQlPath, "ID")).BlockFunc(func(g *Group) {
				g.Return(Id("r").Op(".").Id(entityNameLower).Op(".").Id(fieldNameLower))
			})
			continue
		}

		returnType := "string"
		if column.ColumnType.Type == "int" {
			returnType = "int32"
		}

		resolverFile.Func().Params(Id("r *").Id(entityNameLower + "Resolver")).Id(fieldNameCaps).Params().Params(Id(returnType)).BlockFunc(func(g *Group) {
			g.Return(Id("r").Op(".").Id(entityNameLower).Op(".").Id(fieldNameLower))
		})
	}

	resolverFile.Empty()
	resolverFile.Comment("Mapper methods")
	resolverFile.Func().Id("Map" + entityName).Params(Id("model" + entityName).Qual(const_ModelsPath, entityName)).Params(Id("*" + entityNameLower)).BlockFunc(func(g *Group) {
		g.Empty()

		//g.If(Id("model" + entityName).Op("== (").Qual(const_ModelsPath, entityName).Op("{})")).BlockFunc(func(h *Group) {
		g.If(Qual("reflect", "DeepEqual").Call(Id("model" + entityName), Qual(const_ModelsPath, entityName).Op("{}"))).BlockFunc(func(h *Group) {
			h.Return(Op("&").Id(entityNameLower).Values())
		})

		g.Empty()
		g.Comment("Create graphql " + entityNameLower + " from " + const_ModelsPath + " " + entityName)
		g.Id(entityNameLower).Op(":=").Id(entityNameLower).Values(DictFunc(func(d Dict) {
			for _, column := range entity.Columns {

				fieldNameCaps := snakeCaseToCamelCase(column.Name)

				if column.Name == "id" {
					//graphql.ID(strconv.Itoa(modelUser.Id)),
					d[Id(column.Name)] = Qual(const_UtilsPath, const_UtilsUintToGraphId).Call(Id("model" + entityName).Op(".").Id(fieldNameCaps))
					continue
				}

				if column.ColumnType.Type == "int" {
					d[Id(column.Name)] = Qual("", "int32").Call(Id("model" + entityName).Op(".").Id(fieldNameCaps))
					continue
				}

				d[Id(column.Name)] = Id("model" + entityName).Op(".").Id(fieldNameCaps)

			}
		}))
		g.Return(Id("&" + entityNameLower))
	})

}          //mygraphql

func createEntitiesChildSlice(modelFile *File, entityName string, entityRelationsForAllEndpoint []EntityRelation) {
	allChildren := []string{}
	for _, value := range entityRelationsForAllEndpoint {
		if value.SubEntityName != "" {
			allChildren = append(allChildren, value.SubEntityName)
		}
		//fmt.Println("sub :",value.SubEntityName)
	}

	modelFile.Empty()
	modelFile.Comment("Child entities")
	modelFile.Var().Id(entityName + "Children").Op("=").Lit(allChildren)

	allInterRelation := []string{}

	var flag int

	modelFile.Empty()
	modelFile.Comment("Inter entities")
	modelFile.Var().Id(entityName + "InterRelation").Op("= []").Qual(const_GeneratorPath, "InterEntity").Op("{")
	for _, value := range entityRelationsForAllEndpoint {

		for _, v := range allInterRelation {
			if value.InterEntity.StructName == v {
				flag = 1
			}

		}

		if flag != 1 {
			modelFile.Qual(const_GeneratorPath, "InterEntity").Block(
				Lit(value.InterEntity).Id(","),
			).Id(",")
		}
		allInterRelation = append(allInterRelation, value.InterEntity.StructName)

		//if value.InterEntity.StructName != "" {
		//	for _, val := range allInterRelation {
		//		if val == value.InterEntity {
		//			flag = 1
		//		}
		//	}
		//	if flag != 1 {
		//		allInterRelation = append(allInterRelation, value.InterEntity)
		//	}
		//}
		//fmt.Println("sub :", value.InterEntity)
	}
	modelFile.Op("}")


	//modelFile.Var().Id(entityName + "InterRelation").Op("=").Lit(allInterRelation)

}

func createEntitiesGetAllMethod(modelFile *File, entityName string, methodName string, controllerFile *File) {
	modelFile.Empty()
	//write getAll method
	modelFile.Comment("This method will return a list of all " + entityName + "s")
	modelFile.Func().Id(methodName).Params().Id("[]").Id(entityName).Block(
		Id("data").Op(":=").Op("[]").Id(entityName).Op("{}"),
		Qual(const_DatabasePath, "SQL.Find").Call(Id("&").Id("data")),
		Return(Id("data")),
	)

	controllerFile.Func().Id(methodName).Params(handlerRequestParams()).Block(
		Id("data").Op(":=").Qual(const_ModelsPath, methodName).Call(),
		setJsonHeader(),
		sendResponse(Id("data")),
	)
}

func createEntitiesGetMethod(modelFile *File, entityName string, methodName string, controllerFile *File) {
	modelFile.Empty()
	//write getOne method
	modelFile.Comment("This method will return one " + entityName + " based on id")
	modelFile.Func().Id(methodName).Params(Id("ID").Uint()).Id(entityName).Block(
		Id("data").Op(":=").Id(entityName).Op("{}"),
		Qual(const_DatabasePath, "SQL.First").Call(Id("&").Id("data"), Id("ID")),
		Return(Id("data")),
	)

	controllerFile.Empty()
	controllerFile.Func().Id(methodName).Params(handlerRequestParams()).Block(
		Id("params").Op(":=").Qual(const_RouterPath, "Params").Call(Id("req")),
		Id("ID").Op(":=").Qual("", "params.ByName").Call(Lit("id")),
		Id("data").Op(":=").Qual(const_ModelsPath, methodName).Call(Qual(const_UtilsPath, const_UtilsStringToUInt).Call(Id("ID"))),
		setJsonHeader(),
		sendResponse(Id("data")),
	)
}

func createEntitiesPostMethod(modelFile *File, entityName string, methodName string, entityFields []EntityField, controllerFile *File) {
	modelFile.Empty()
	//write insert method
	modelFile.Comment("This method will insert one " + entityName + " in db")
	modelFile.Func().Id(methodName).Params(Id("data").Id(entityName)).Id(entityName).Block(
		Qual(const_DatabasePath, "SQL.Create").Call(Id("&").Id("data")),
		Return(Id("data")),
	)

	// controller method
	controllerFile.Empty()
	controllerFile.Func().Id(methodName).Params(handlerRequestParams()).Block(
		Id("decoder").Op(":=").Qual("encoding/json", "NewDecoder").Call(Id("req").Op(".").Id("Body")),
		Var().Id("data").Qual(const_ModelsPath, entityName),
		Id("err").Op(":=").Qual("", "decoder.Decode").Call(Id("&").Id("data")),
		If(Id("err").Op("!=").Nil()).Block(
			setJsonHeader(),
			sendResponse("invalid data"),
			Return(),
		),
		Defer().Qual("", "req.Body.Close").Call(),
		Id("data").Op("=").Qual(const_ModelsPath, methodName).Call(Id("data")),
		setJsonHeader(),
		sendResponse(Id("data")),
	)
}

func createEntitiesPutMethod(modelFile *File, entityName string, methodName string, controllerFile *File) {
	modelFile.Empty()
	//write update method
	modelFile.Comment("This method will update " + entityName + " based on id")
	modelFile.Func().Id(methodName).Params(Id("newData").Id(entityName)).Id(entityName).Block(
		Id("oldData").Op(":=").Id(entityName).Id("{").Id("Id").Op(":").Id("newData").Op(".").Id("Id").Id("}"),
		Qual(const_DatabasePath, "SQL.Model").Call(Id("&oldData")).Op(".").Id("Updates").Call(Id("newData")),
		Return(Id("newData")),
	)

	//controller method
	controllerFile.Empty()
	controllerFile.Func().Id(methodName).Params(handlerRequestParams()).Block(

		Id("params").Op(":=").Qual(const_RouterPath, "Params").Call(Id("req")),
		Id("ID").Op(":=").Qual("", "params.ByName").Call(Lit("id")),

		Id("decoder").Op(":=").Qual("encoding/json", "NewDecoder").Call(Id("req").Op(".").Id("Body")),
		Var().Id("newData").Qual(const_ModelsPath, entityName),
		Id("err").Op(":=").Qual("", "decoder.Decode").Call(Id("&").Id("newData")),
		If(Id("err").Op("!=").Nil()).Block(
			setJsonHeader(),
			sendResponse("invalid data"),
			Return(),
		),
		Defer().Qual("", "req.Body.Close").Call(),

		Empty(),
		Id("newData.Id").Op("=").Qual(const_UtilsPath, const_UtilsStringToUInt).Call(Id("ID")),
		Id("data").Op(":=").Qual(const_ModelsPath, methodName).Call(Id("newData")),
		setJsonHeader(),
		sendResponse(Id("data")),

	)
}

func createEntitiesDeleteMethod(modelFile *File, entityName string, methodName string, controllerFile *File) {

	parent := Entity{}
	database.SQL.Where("c_entity.display_name = (?)", entityName).Find(&parent)   //current parent

	modelFile.Empty()
	//write delete method
	modelFile.Comment("This method will delete " + entityName + " based on id")
	modelFile.Func().Id(methodName).Params(Id("ID").Uint()).Bool().Block(
		Id("data").Op(":=").Id(entityName).Op("{").Id("Id").Op(":").Id("ID").Op("}"),
		Var().Id("del").Bool(),
		Qual(const_DatabasePath, "SQL.Where").Call(Lit(parent.Name + ".id=(?)"), Id("ID")).Dot("First").Call(Id("&").Id("data")),
		If(Id("data").Op("!=").Nil()).Block(
			Id("del").Op("=").True(),
		),
		Qual(const_DatabasePath, "SQL.Delete").Call(Id("&").Id("data")),
		Return(Id("del")),
	)

	//controller method
	controllerFile.Empty()
	controllerFile.Func().Id(methodName).Params(handlerRequestParams()).Block(

		Comment("Get the parameter id"),
		Id("params").Op(":=").Qual(const_RouterPath, "Params").Call(Id("req")),
		Id("ID").Op(":=").Qual("", "params.ByName").Call(Lit("id")),
		Id("data").Op(":=").Qual(const_ModelsPath, methodName).Call(Qual(const_UtilsPath, const_UtilsStringToUInt).Call(Id("ID"))),
		setJsonHeader(),
		sendResponse(Id("data")),
	)
}

func createEntitiesAllChildMethod(modelFile *File, entityName string, allMethodName string, entityRelationsForAllEndpoint []EntityRelation) {
	modelFile.Empty()
	modelFile.Func().Id(allMethodName).Params(handlerRequestParams()).BlockFunc(func(g *Group) {
		g.Empty()
		g.Comment("Get the parameter id")
		g.Id("params").Op(":=").Qual(const_RouterPath, "Params").Call(Id("req"))
		g.Id("ID").Op(",").Id("_").Op(":=").Qual("strconv", "ParseUint").Call(
			Qual("", "params.ByName").Call(Lit("id")),
			Id("10"),
			Id("0"),
		)
		g.Id("data").Op(":=").Id(entityName).Op("{").Id("Id").Op(":").Id("uint(ID)").Op("}")
		g.Empty()
		g.Var().Id("relations ").Op("[").Id(strconv.Itoa(len(entityRelationsForAllEndpoint))).Op("]").Id("string")
		g.Id("children").Op(":=").Qual("", "req.URL.Query().Get").Call(Lit("child"))
		g.If(Id("children").Op("!= \"\"")).
			Block(
			Var().Id("neededChildren ").Op("[]").Id("string"),

			For(Id("_,child").Op(":=").Id("range").Id(entityName + "Children")).
				Block(
				If(Qual("", "isValueInList").
					Call(
					Id("child"),
					Qual("strings", "Split").
						Call(
						Id("children"), Id("sep"),
					),
				).
					Block(
					Id("neededChildren").Op("=").Qual("", "append").Call(Id("neededChildren"), Id("child")),
				),
				), ),

			Empty(),

			For(Id("i").Op(":=").Id("range").Id("neededChildren")).
				Block(
				Id("relations").Op("[").Id("i").Op("]").Op("=").Id("neededChildren").Op("[").Id("i").Op("]"),
			),
		).Else().
			Block(
			For(Id("i").Op(":=").Id("range").Id(entityName + "Children")).
				Block(
				Id("relations").Op("[").Id("i").Op("]").Op("=").Id(entityName + "Children").Op("[").Id("i").Op("]"),
			),
		)
		g.If(Qual("", "len").Call(Id("relations")).Op(">0")).BlockFunc(func(g *Group) {

			var buffer bytes.Buffer
			buffer.WriteString("SQL.")
			for i := range entityRelationsForAllEndpoint {
				buffer.WriteString("Preload(relations[" + strconv.Itoa(i) + "]).")
			}
			buffer.WriteString("First")
			g.Qual(const_DatabasePath, buffer.String()).Call(Op("&").Id("data"))
		})
		g.Qual("", "w.Header().Set").Call(Lit("Content-Type"), Lit("application/json"))
		g.Qual("encoding/json", "NewEncoder").Call(Id("w")).Op(".").Id("Encode").Call(Id("Response").
			Op("{").
			Id("2000").Op(",").
			Lit("Data fetched successfully").Op(",").
			Id("data").
			Op("}"))
	})
}

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

	if isInput {
		fieldName = fieldNameCaps
	} else {
		fieldName = fieldNameLower
	}

	if fieldName == "id" || fieldName == "ID" || fieldName == "Id" {

		finalId := fieldName
		if isInput {
			finalId = fieldName + " *"
		}

		g.Id(finalId).Qual(const_GraphQlPath, "ID")
		return
	}

	if col.ColumnType.Type == "int" {
		finalId := fieldName + " int32"
		g.Id(finalId)
	} else if col.ColumnType.Type == "varchar" {
		finalId := fieldName + " string"
		g.Id(finalId)
	} else {
		g.Id(fieldName).String() //default string
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
