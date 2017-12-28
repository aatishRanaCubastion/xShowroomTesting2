package generator

import (
	"github.com/jinzhu/gorm"
	"strings"
	. "github.com/dave/jennifer/jen"
	u "utils"
)

func createSchema(schemaFile *File, allEntities []Entity, db *gorm.DB) {

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

	//write mutation schema
	u.SAppend(&sS, "# The mutation type, represents all updates we can make to our data\n")
	u.SAppend(&sS, "type Mutation {\n")
	u.SAppend(&sS, "# Create\n")
	for _, val := range allEntities {
		entityNameLower := strings.ToLower(val.DisplayName)
		entityNameCaps := snakeCaseToCamelCase(val.DisplayName)
		u.SAppend(&sS, "\tupsert" + entityNameCaps + "(" + entityNameLower + ": " + entityNameCaps + "Input!) :" + entityNameCaps + "\n")
	}
	u.SAppend(&sS, "}\n\n")
	var relationParent = []Relation{}
	var relationChild = []Relation{}

	for _, val := range allEntities {
			db.Preload("InterEntity").
				Preload("ChildEntity").
				Preload("ChildColumn").
				Preload("ParentColumn").
				Where("parent_entity_id=?", val.ID).
				Find(&relationParent)
			db.Preload("InterEntity").
				Preload("ParentEntity").
				Preload("ChildColumn").
				Preload("ParentColumn").
				Where("child_entity_id=?", val.ID).
				Find(&relationChild)

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
		for _, child := range relationParent {

			fieldType := child.ChildEntity.DisplayName
			fieldTypeLower := strings.ToLower(child.ChildEntity.DisplayName)

			if child.RelationTypeID == 1 || child.RelationTypeID == 4 {
				u.SAppend(&sS, "\t" + fieldTypeLower + ": " + fieldType + "!\n")
			} else {
				u.SAppend(&sS, "\t" + fieldTypeLower + "s" + ":" + "[" + fieldType + "!]" + "!\n")
			}
		}

		for _, child := range relationChild {

			fieldType := child.ParentEntity.DisplayName
			fieldTypeLower := strings.ToLower(child.ParentEntity.DisplayName)

			if child.RelationTypeID == 3 || child.RelationTypeID == 6 {
				u.SAppend(&sS, "\t" + fieldTypeLower + "s" + ": " + "[" + fieldType + "!]" + "!\n")
			} else {

				u.SAppend(&sS, "\t" + fieldTypeLower + ": " + fieldType + "!\n")
			}
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
			if strings.HasSuffix(col.Name, "_id") {
				u.SAppend(&sS, "\t" + col.Name + ": " + fieldType + "\n")

			} else if strings.HasSuffix(col.Name, "_type") {
				u.SAppend(&sS, "\t" + col.Name + ": " + fieldType + "\n")

			} else {
				u.SAppend(&sS, "\t" + col.Name + ": " + fieldType + "!\n")
			}

		}
		for _, child := range relationParent {

			fieldType := child.ChildEntity.DisplayName
			fieldTypeLower := strings.ToLower(child.ChildEntity.DisplayName)

			if child.RelationTypeID == 1 || child.RelationTypeID == 4 {
				u.SAppend(&sS, "\t" + fieldTypeLower + ": " + fieldType + "Input" + "\n")

			} else {
				u.SAppend(&sS, "\t" + fieldTypeLower + "s" + ": " + "[" + fieldType + "Input!]" + "\n")

			}
		}
		u.SAppend(&sS, "}\n\n")
	}

	schemaFile.Var().Id("Schema").Op("=").Id("`" + sS + "`")
}

