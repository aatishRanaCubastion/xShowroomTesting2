package generator

import (
	"github.com/jinzhu/gorm"
	"os"
	"strings"
	"database"
	"fmt"
	"log"
	. "github.com/dave/jennifer/jen"

	"strconv"
	"bytes"
)

func createEntities(entity Entity, db *gorm.DB) string {

	// create entity name from table
	entityName := snakeCaseToCamelCase(entity.DisplayName)

	//entity relations stored to generate routes and their methods for each sub entities ((parent to child) and (child to parent))
	entityRelationsForEachEndpoint := []EntityRelation{}

	//entity relations stored to generate one route to access all sub entities depending on query params(parent to child only)
	entityRelationsForAllEndpoint := []EntityRelation{}
        var childOfEntity []string
	//create entity file in models sub directory
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

		//write composite fields while looking at parent
		for _, relation := range relationsParent {

			interName:=relation.InterEntity.Name
			interDispName:=relation.InterEntity.DisplayName

			interEntity:=InterEntity{TableName:interName,StructName:interDispName}

			//fmt.Println("parent ", relation.InterEntity.Name)
			name := snakeCaseToCamelCase(relation.ChildEntity.DisplayName)
			childName := string(relation.ChildColumn.Name)
			parentName := string(relation.ParentColumn.Name)

			d := " "
			relType := "_normal"
			if entityName == name {
				d = "*" //if name and entityName are same, its a self join, so add *
				relType = "_self"
			}
			//comment
			switch relation.RelationTypeID {
			case 1: //one to one
				relationName := name
				finalId := relationName + " " + d + name + " `gorm:\"ForeignKey:" + childName + ";AssociationForeignKey:" + parentName + "\" json:\"" + relation.ChildEntity.DisplayName + ",omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"OneToOne" + relType, name, childName,InterEntity{}})
				//entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToOne" + relType, relationName, childName, InterEntity{}})
				childOfEntity=append(childOfEntity,name)
				g.Id(finalId)
			case 2: //one to many
				relationName := name + "s"
				finalId := relationName + " []" + name + " `gorm:\"ForeignKey:" + childName + ";AssociationForeignKey:" + parentName + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"OneToMany", name, childName,InterEntity{}})
				//entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", relationName, childName})
				childOfEntity=append(childOfEntity,name+"s")

				g.Id(finalId)
			case 3: //many to many
				relationName := name + "s"
				//finalId := relationName + " []" + name + " `gorm:\"many2many:" + relation.InterEntity.Name + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				finalId := relationName + " []" + name + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				g.Id(finalId)
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"ManyToMany", name, childName,InterEntity{}})
			        entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", relationName, childName,interEntity})


			case 4: // Polymorphic OnetoOne
				relationName := name
				finalId := relationName + " " + d + name + " `gorm:\":" + childName + ";AssociationForeignKey:" + parentName + "\" json:\"" + relation.ChildEntity.DisplayName + ",omitempty\"`"
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"OneToOne" + relType, name, childName,InterEntity{}})
				//entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToOne" + relType, relationName, childName})
				childOfEntity=append(childOfEntity,name)

				g.Id(finalId)

			case 5:        // Polymorphic OnetoMany
				relationName := name + "s"
				//finalId := relationName + " []" + name + " `gorm:\"many2many:" + relation.InterEntity.Name + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				finalId := relationName + " []" + name + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				g.Id(finalId)
				childOfEntity=append(childOfEntity,name+"s")

				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"ManyToMany", name, childName,InterEntity{}})

			case 6:        // Polymorphic ManytoMany
				relationName := name + "s"
				//finalId := relationName + " []" + name + " `gorm:\"many2many:" + relation.InterEntity.Name + "\" json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				finalId := relationName + " []" + name + " `json:\"" + relation.ChildEntity.DisplayName + "s,omitempty\"`"
				g.Id(finalId)
				entityRelationsForEachEndpoint = append(entityRelationsForEachEndpoint, EntityRelation{"ManyToMany", name, childName,InterEntity{}})
				entityRelationsForAllEndpoint = append(entityRelationsForAllEndpoint, EntityRelation{"OneToMany", relationName, childName,interEntity})

			}

		}


		//write composite fields while looking at child
		for _, relation := range relationsChild {
			interName := relation.InterEntity.Name
			interDispName := relation.InterEntity.DisplayName

			interEntity := InterEntity{TableName:interName, StructName:interDispName}

			name := snakeCaseToCamelCase(relation.ParentEntity.DisplayName)
			childName := string(relation.ChildColumn.Name)
			//     fmt.Println("entity child name :",entityName,"inter :",interEntity)
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
	createEntitiesResolver(resolverFile, entityName, entity, database.SQL,entityRelationsForAllEndpoint)

	createEntitiesChildSlice(modelFile, entityName, entityRelationsForAllEndpoint,childOfEntity)

	createEntitiesGetAllMethod(modelFile, entityName, getAllMethodName, controllerFile)

	createEntitiesGetMethod(modelFile, entityName, getByIdMethodName, controllerFile)

	createEntitiesPostMethod(modelFile, entityName, postMethodName, entityFields, controllerFile)

	createEntitiesPutMethod(modelFile, entityName, putMethodName, controllerFile)

	createEntitiesDeleteMethod(modelFile, entityName, deleteMethodName, controllerFile)

	createResolverFieldFunctions(modelFile, entity, database.SQL)


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

func createEntitiesChildSlice(modelFile *File, entityName string, entityRelationsForAllEndpoint []EntityRelation,childOfEntity []string) {
	allChildren := []string{}
	for _, value := range childOfEntity {
		if value!= "" {
			allChildren = append(allChildren, value)
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
			//modelFile.Qual(const_GeneratorPath, "InterEntity").Block(
			modelFile.Lit(value.InterEntity).Id(",")
			//).Id(",")
		}
		allInterRelation = append(allInterRelation, value.InterEntity.StructName)

		//if value.InterEntity.StructName != "" {
		//     for _, val := range allInterRelation {
		//            if val == value.InterEntity {
		//                   flag = 1
		//            }
		//     }
		//     if flag != 1 {
		//            allInterRelation = append(allInterRelation, value.InterEntity)
		//     }
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
		//Id("data").Op(":=").Id(entityName).Op("{").Id("Id").Op(":").Id("ID").Op("}"),
		Var().Id("data").Id(entityName),
		Var().Id("del").Bool(),
		Qual(const_DatabasePath, "SQL.Where").Call(Lit(parent.Name + ".id=(?)"), Id("ID")).Dot("First").Call(Id("&").Id("data")),
		If(Id("data.Id").Op("!=").Id("0")).Block(
			Qual(const_DatabasePath, "SQL.Delete").Call(Id("&").Id("data")),

			Id("del").Op("=").True(),
		),
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

func createResolverFieldFunctions(modelFile *File, entity Entity, db *gorm.DB) {
	entityName := snakeCaseToCamelCase(entity.DisplayName)
	//entityNameLower := strings.ToLower(entity.DisplayName)

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

	for _, val := range relationsParent {
		childNameLower := strings.ToLower(val.ChildEntity.DisplayName)
		childNameCaps := snakeCaseToCamelCase(val.ChildEntity.DisplayName)

		if val.RelationTypeID == 1 || val.RelationTypeID == 2 {
			modelFile.Func().Id("Get" + entityName + "Of" + childNameCaps).Params(Id(childNameLower).Id(childNameCaps)).Id(entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id(entityName).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(childNameLower).Dot(val.ChildColumn.DisplayName)).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data"))
			})
		}

		if val.RelationTypeID == 4 || val.RelationTypeID == 5 {
			modelFile.Func().Id("Get" + entityName + "Of" + childNameCaps).Params(Id(childNameLower).Id(childNameCaps)).Id(entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id(entityName).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(childNameLower).Dot(val.ChildColumn.DisplayName)).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data"))
			})
		}

		if val.RelationTypeID == 3 || val.RelationTypeID == 6 {
			// todo many to many and polymorphic many to many

			modelFile.Func().Id("Get" + entityName + "sOf" + childNameCaps).Params(Id(childNameLower+"id").Uint()).Id("[]"+entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id("[]"+entityName).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(childNameLower+"id")).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data"))
			})
		}

	}

	for _, val := range relationsChild {
		parentNameLower := strings.ToLower(val.ParentEntity.DisplayName)
		parentNameCaps := snakeCaseToCamelCase(val.ParentEntity.DisplayName)

		if val.RelationTypeID == 1 || val.RelationTypeID ==4{
			modelFile.Func().Id("Get" + entityName + "Of" + parentNameCaps).Params(Id(parentNameLower + "id").Uint()).Id(entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id(parentNameCaps).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").Id("Preload").Params(Lit(entityName)).Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(parentNameLower + "id")).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data").Op(".").Id(entityName))
			})
		}

		if val.RelationTypeID == 2 || val.RelationTypeID == 5{
			modelFile.Func().Id("Get" + entityName + "sOf" + parentNameCaps).Params(Id(parentNameLower + "id").Uint()).Id("[]" + entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id(parentNameCaps).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").Id("Preload").Params(Lit(entityName)).Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(parentNameLower + "id")).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data").Op(".").Id(entityName + "s"))
			})
		}

		if val.RelationTypeID == 3 || val.RelationTypeID == 6 {
			//todo many to many and polymorphic many to many
			modelFile.Func().Id("Get" + entityName + "sOf" + parentNameCaps).Params(Id(parentNameLower + "id").Uint()).Id("[]" + entityName).BlockFunc(func(g *Group) {
				g.Id("data").Op(":=").Id(parentNameCaps).Block()
				g.Qual(const_DatabasePath, "SQL").Op(".").Id("Debug").Params().Op(".").Id("Preload").Params(Lit(entityName)).Op(".").
					Id("Where").Params(Lit("id = ?").Op(",").Id(parentNameLower + "id")).Op(".").Id("Find").Params(Id("&data"))
				g.Return(Id("data").Op(".").Id(entityName + "s"))
			})
		}
	}

}