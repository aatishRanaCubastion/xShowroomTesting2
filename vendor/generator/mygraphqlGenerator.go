package generator

import (
	"github.com/jinzhu/gorm"
	"strings"
	. "github.com/dave/jennifer/jen"

	"fmt"
)

func createEntitiesResolver(resolverFile *File, entityName string, entity Entity, db *gorm.DB,entityRelationsForAllEndpoint []EntityRelation) {


	var childOfEntity = []Relation{}
	db.Preload("InterEntity").
		Preload("ChildEntity").
		Preload("ChildColumn").
		Preload("ParentColumn").
		Where("parent_entity_id=?", entity.ID).
		Find(&childOfEntity)
	var parOfEntity = []Relation{}
	db.Preload("InterEntity").
		Preload("ParentEntity").
		Preload("ChildColumn").
		Preload("ParentColumn").
		Where("child_entity_id=?", entity.ID).
		Find(&parOfEntity)

	entityNameLower := strings.ToLower(entityName)
	resolverFile.Comment("Struct for graphql")
	resolverFile.Type().Id(entityNameLower).StructFunc(func(g *Group) {
		//write primitive fields
		for _, column := range entity.Columns {
			mapColumnTypesResolver(column, g, false)
		}
		for _, child := range childOfEntity {

			fieldTypeLower := strings.ToLower(child.ChildEntity.DisplayName)

			if child.RelationTypeID == 1 || child.RelationTypeID == 4 {
				g.Id(fieldTypeLower).Id("*" + fieldTypeLower)

			} else {
				g.Id(fieldTypeLower + "s").Op("[]*").Id(fieldTypeLower)

			}
		}

		for _, child := range parOfEntity {

			fieldTypeLower := strings.ToLower(child.ParentEntity.DisplayName)

			if child.RelationTypeID == 3 || child.RelationTypeID == 6 {
				g.Id(fieldTypeLower + "s").Op("[]*").Id(fieldTypeLower)
			} else {
				g.Id(fieldTypeLower).Id("*" + fieldTypeLower)

			}
		}

	})
	resolverFile.Empty()
	resolverFile.Comment("Struct for upserting")
	resolverFile.Type().Id(entityNameLower + "Input").StructFunc(func(g *Group) {
		//write primitive fields
		for _, column := range entity.Columns {
			mapColumnTypesResolver(column, g, true)
		}

		for _, child := range childOfEntity {
			fieldType := child.ChildEntity.DisplayName
			fieldTypeLower := strings.ToLower(child.ChildEntity.DisplayName)

			if child.RelationTypeID == 1 || child.RelationTypeID == 4 {
				g.Id(fieldType).Id("*" + fieldTypeLower + "Input")

			} else {
				g.Id(fieldType + "s").Op("*[]").Id(fieldTypeLower + "Input")

			}
		}
	})
	resolverFile.Empty()
	resolverFile.Comment("Struct for response")
	resolverFile.Type().Id(entityNameLower + "Resolver").StructFunc(func(g *Group) {
		g.Id(entityNameLower).Id(" *").Id(entityNameLower)
	})
	resolverFile.Empty()
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
	resolverFile.Empty()

	entitiesUpsertResolver(resolverFile,entityName, entity, db)

	resolverFile.Empty()
	resolverFile.Empty()

	entitiesdeleteResolver(resolverFile,entityName,entity,entityRelationsForAllEndpoint)

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



	for _,value := range childOfEntity {
		childNameLower := strings.ToLower(value.ChildEntity.DisplayName)
		childNameCaps := snakeCaseToCamelCase(value.ChildEntity.DisplayName)

		if value.RelationTypeID==1 || value.RelationTypeID == 4{
			resolverFile.Func().Params(Id("r *" + entityNameLower + "Resolver")).Id(childNameCaps).Params().Id("*" + childNameLower + "Resolver").BlockFunc(func(g *Group) {
				g.If(Id("r").Op(".").Id(entityNameLower).Op("!=").Nil()).BlockFunc(func(h *Group) {
					h.Id(childNameLower).Op(":=").Qual(const_ModelsPath, "Get" + childNameCaps + "Of" + entityName).Call(
						Qual(const_UtilsPath, const_UtilsConvertId).Call(Id("r").Op(".").Id(entityNameLower).Op(".").Id("id")),
					)
					h.Return(Id("&" + childNameLower + "Resolver").Values(
						Qual("", "Map" + childNameCaps).Call(Id(childNameLower)),
					))
				})
				g.Return(Id("&" + childNameLower + "Resolver").Values(Id("r").Op(".").Id(entityNameLower).Op(".").Id(childNameLower)), )
			})
		}

		if value.RelationTypeID==2 || value.RelationTypeID==3 || value.RelationTypeID == 5 || value.RelationTypeID == 6{
			resolverFile.Func().Params(Id("r *" + entityNameLower + "Resolver")).Id(childNameCaps+"s").Params().Id("[]*" + childNameLower + "Resolver").BlockFunc(func(g *Group) {
				g.Var().Id(childNameLower+"s").Id("[]*"+childNameLower+"Resolver")
				g.If(Id("r").Op(".").Id(entityNameLower).Op("!=").Nil()).BlockFunc(func(h *Group) {
					h.Id(childNameLower).Op(":=").Qual(const_ModelsPath, "Get" + childNameCaps + "sOf" + entityName).Call(
						Qual(const_UtilsPath, const_UtilsConvertId).Call(Id("r").Op(".").Id(entityNameLower).Op(".").Id("id")),
					)
					h.For().Id("_").Op(",").Id("value").Op(":=").Range().Id(childNameLower).BlockFunc(func (j *Group) {
						j.Id(childNameLower+"s").Op("=").Append(Id(childNameLower+"s"),Id("&"+childNameLower+"Resolver").Values(Qual("","Map"+childNameCaps).Call(Id("value"))))
					})
					h.Return(Id(childNameLower + "s"))
				})
				g.For().Id("_").Op(",").Id("value").Op(":=").Range().Id("r").Op(".").Id(entityNameLower).Op(".").Id(childNameLower+"s").BlockFunc(func (h *Group){
					h.Id(childNameLower+"s").Op("=").Append(Id(childNameLower+"s"),Id("&"+childNameLower+"Resolver").Values(Id("value")))
				})
				g.Return(Id(childNameLower+"s"))
			})

		}
	}

	for _,value := range parOfEntity {
		parentNameLower := strings.ToLower(value.ParentEntity.DisplayName)
		parentNameCaps := snakeCaseToCamelCase(value.ParentEntity.DisplayName)

		if value.RelationTypeID==1 || value.RelationTypeID==2 || value.RelationTypeID == 4 || value.RelationTypeID == 5{
			resolverFile.Func().Params(Id("r *" + entityNameLower + "Resolver")).Id(parentNameCaps).Params().Id("*" + parentNameLower + "Resolver").BlockFunc(func(g *Group) {
				g.If(Id("r").Op(".").Id(entityNameLower).Op("!=").Nil()).BlockFunc(func(h *Group) {
					h.Id(parentNameLower).Op(":=").Qual(const_ModelsPath, "Get" + parentNameCaps + "Of" + entityName).Call(
						Qual(const_UtilsPath, const_UtilsConvertId).Call(Id("r").Op(".").Id(entityNameLower).Op(".").Id("id")),
					)
					h.Return(Id("&" + parentNameLower + "Resolver").Values(
						Qual("", "Map" + parentNameCaps).Call(Id(parentNameLower)),
					))
				})
				g.Return(Id("&" + parentNameLower + "Resolver").Values(Id("r").Op(".").Id(entityNameLower).Op(".").Id(parentNameLower)), )
			})
		}

		if value.RelationTypeID==3 || value.RelationTypeID == 6{
			resolverFile.Func().Params(Id("r *" + entityNameLower + "Resolver")).Id(parentNameCaps+"s").Params().Id("*" + parentNameLower + "Resolver").BlockFunc(func(g *Group) {
				g.Var().Id(parentNameLower+"s").Id("[]*"+parentNameLower+"Resolver")
				g.If(Id("r").Op(".").Id(entityNameLower).Op("!=").Nil()).BlockFunc(func(h *Group) {
					h.Id(parentNameLower).Op(":=").Qual(const_ModelsPath, "Get" + parentNameCaps + "sOf" + entityName).Call(
						Qual(const_UtilsPath, const_UtilsConvertId).Call(Id("r").Op(".").Id(entityNameLower).Op(".").Id("id")),
					)
					h.For().Id("_").Op(",").Id("value").Op(":=").Range().Id(parentNameLower).BlockFunc(func (j *Group) {
						j.Id(parentNameLower+"s").Op("=").Append(Id(parentNameLower+"s"),Id("&"+parentNameLower+"Resolver").Values(Qual("","Map"+parentNameCaps).Call(Id("value"))))
					})
					h.Return(Id(parentNameLower + "s"))
				})
				g.For().Id("_").Op(",").Id("value").Op(":=").Range().Id("r").Op(".").Id(entityNameLower).Op(".").Id(parentNameLower+"s").BlockFunc(func (h *Group){
					h.Id(parentNameLower+"s").Op("=").Append(Id(parentNameLower+"s"),Id("&"+parentNameLower+"Resolver").Values(Id("value")))
				})
				g.Return(Id(parentNameLower+"s"))
			})

		}

	}


	resolverFile.Empty()
	resolverFile.Comment("Mapper methods")
	resolverFile.Func().Id("Map" + entityName).Params(Id("model" + entityName).
		Qual(const_ModelsPath, entityName)).Params(Id("*" + entityNameLower)).BlockFunc(func(g *Group) {
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
	resolverFile.Empty()
	resolverFile.Comment("Reverse Mapper methods")
	resolverFile.Func().Id("ReverseMap" + entityName).Params(Id("mygraphql" + entityName).Id("*" + entityNameLower + "Input")).
		Params(Qual(const_ModelsPath, entityName)).BlockFunc(func(g *Group) {
		g.Empty()

		//g.If(Id("model" + entityName).Op("== (").Qual(const_ModelsPath, entityName).Op("{})")).BlockFunc(func(h *Group) {
		g.If(Qual("reflect", "DeepEqual").Call(Id("mygraphql" + entityName), Id(entityNameLower + "Input").Op("{}"))).BlockFunc(func(h *Group) {
			h.Return(Qual(const_ModelsPath, entityName).Values())
		})

		g.Empty()
		g.Comment("Create graphql " + entityNameLower + " from " + const_ModelsPath + " " + entityName)
		g.Id(entityNameLower + "Model").Op(":=").Qual(const_ModelsPath, entityName).Values(DictFunc(func(d Dict) {
			for _, column := range entity.Columns {

				fieldNameCaps := snakeCaseToCamelCase(column.Name)

				if column.Name == "id" {
					//graphql.ID(strconv.Itoa(modelUser.Id)),
					d[Id(fieldNameCaps)] = Qual("utils", "ConvertId").Params(Id("*mygraphql" + entityName).Op(".").Id(fieldNameCaps))
					continue
				}

				if column.ColumnType.Type == "int" {
					d[Id(fieldNameCaps)] = Qual(const_UtilsPath,const_UtilsInt32ToUint).Call(Id("mygraphql" + entityName).Op(".").Id(fieldNameCaps))
					continue
				}

				d[Id(fieldNameCaps)] = Id("mygraphql" + entityName).Op(".").Id(fieldNameCaps)

			}
		}))
		g.Return(Id(entityNameLower + "Model"))
	})

}

func entitiesUpsertResolver(resolverFile *File, entityName string, entity Entity, db *gorm.DB){

	var childOfEntity = []Relation{}
	db.Preload("InterEntity").
		Preload("ChildEntity").
		Preload("ChildColumn").
		Preload("ParentColumn").
		Where("parent_entity_id=?", entity.ID).
		Find(&childOfEntity)

	entityNameLower := strings.ToLower(entityName)




	resolverFile.Func().Id("ResolveCreate" + entityName).Params(Id("args").Id("*").StructFunc(func(g *Group) {

		g.Id(entityName + " *").Id(entityNameLower + "Input")

	})).Id("*" + entityNameLower + "Resolver").BlockFunc(func(h *Group) {

		h.Var().Id(entityNameLower).Op("=").Op("&").Id(entityNameLower).Op("{}")
		h.Empty()
		h.If(Id("args").Dot(entityName).Dot("Id").Op("==").Nil()).Block(

			Id(entityNameLower).Op("=").Id("Map" + entityName).Params(Id("models").Dot("Post" + entityName).
				Params(Id("ReverseMap" + entityName).Params(Id("args").Dot(entityName)))),

		).Else().Block(

			Id(entityNameLower).Op("=").Id("Map" + entityName).Call(Id("models").Dot("Put" + entityName).
				Call(Id("ReverseMap" + entityName).Params(Id("args").Dot(entityName)))),

		)

		for _, val := range childOfEntity {

			var childName = val.ChildEntity.DisplayName
			var childNameLower = strings.ToLower(val.ChildEntity.DisplayName)

			if val.RelationTypeID == 1 {

				h.If(Id(entityNameLower).Op("!=").Nil().Id("&&").Id("args").Dot(entityName).
					Dot(childName).Op("!=").Nil()).Block(

					If(Id("args").Dot(entityName).Dot(childName).Dot("Id").Op("==").Nil()).Block(
						Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("args").
							Dot(entityName).Dot(childName)),

						If(Id(childNameLower).Dot(entityName + "Id").Op("!=0 && ").
							Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot(entityName + "Id")).Block(
							Comment("todo throw error"),
							Return(Id("&"+entityNameLower+"Resolver{}")),
						),

						Id(childNameLower).Dot(entityName + "Id").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")),
						Id(entityNameLower).Dot(childNameLower).Op("=").
							Id("Map" + childName).Call(Id("models").Dot("Post" + childName).
							Params(Id(childNameLower))),


					).Else().Block(


						Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("args").
							Dot(entityName).Dot(childName)),

						If(Id(childNameLower).Dot(entityName + "Id").Op("!=0 && ").
							Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot(entityName + "Id")).Block(
							Comment("todo throw error"),
							Return(Id("&"+entityNameLower+"Resolver{}")),						),

						Id(childNameLower).Dot(entityName + "Id").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")),
						Id(entityNameLower).Dot(childNameLower).Op("=").
							Id("Map" + childName).Call(Id("models").Dot("Put" + childName).
							Params(Id(childNameLower))),


					),

				)

			} else if val.RelationTypeID == 4 {

				h.If(Id(entityNameLower).Op("!=").Nil().Id("&&").Id("args").Dot(entityName).
					Dot(childName).Op("!=").Nil()).Block(

					If(Id("args").Dot(entityName).Dot(childName).Dot("Id").Op("==").Nil()).Block(
						Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("args").
							Dot(entityName).Dot(childName)),

						If(Id(childNameLower).Dot("TypeId").Op("!=0 && ").
							Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot("TypeId")).Block(
							Comment("todo throw error"),
							Return(Id("&"+entityNameLower+"Resolver{}")),						),

						Id(childNameLower).Dot("TypeId").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")),
						Id(entityNameLower).Dot(childNameLower).Op("=").
							Id("Map" + childName).Call(Id("models").Dot("Post" + childName).
							Params(Id(childNameLower))),


					).Else().Block(


						Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("args").
							Dot(entityName).Dot(childName)),

						If(Id(childNameLower).Dot("TypeId").Op("!=0 && ").
							Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot("TypeId")).Block(
							Comment("todo throw error"),
							Return(Id("&"+entityNameLower+"Resolver{}")),						),

						Id(childNameLower).Dot("TypeId").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")),
						Id(entityNameLower).Dot(childNameLower).Op("=").
							Id("Map" + childName).Call(Id("models").Dot("Put" + childName).
							Params(Id(childNameLower))),


					),

				)

			} else if val.RelationTypeID == 2 {

				h.If(Id(entityNameLower).Op("!=").Nil().Id("&&").Id("args").Dot(entityName).
					Dot(childName + "s").Op("!=").Nil()).Block(

					For(Id("_ ,").Id("dev").Op(":=").Range().Id("*args").Dot(entityName).Dot(childName + "s")).Block(


						If(Id("dev").Dot("Id").Op("==").Nil()).Block(
							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),

							If(Id(childNameLower).Dot(entityName + "Id").Op("!=0 && ").
								Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).
								Op("!=").Id(childNameLower).Dot(entityName + "Id")).Block(
								Comment("todo throw error"),
								Return(Id("&"+entityNameLower+"Resolver{}")),							),

							Id(childNameLower).Dot(entityName + "Id").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).
								Call(Id(entityNameLower).Dot("id")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Post" + childName).
								Params(Id(childNameLower)))),


						).Else().Block(

							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),

							If(Id(childNameLower).Dot(entityName + "Id").Op("!=0 && ").
								Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot(entityName + "Id")).Block(
								Comment("todo throw error"),
								Return(Id("&"+entityNameLower+"Resolver{}")),							),

							Id(childNameLower).Dot(entityName + "Id").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).
								Call(Id(entityNameLower).Dot("id")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Put" + childName).
								Params(Id(childNameLower)))),
						),

					),
				)
			} else if val.RelationTypeID == 3 {

				h.If(Id(entityNameLower).Op("!=").Nil().Id("&&").Id("args").Dot(entityName).
					Dot(childName + "s").Op("!=").Nil()).Block(

					For(Id("_ ,").Id("dev").Op(":=").Range().Id("*args").Dot(entityName).Dot(childName + "s")).Block(


						If(Id("dev").Dot("Id").Op("==").Nil()).Block(
							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Post" + childName).
								Params(Id(childNameLower)))),


						).Else().Block(

							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Put" + childName).
								Params(Id(childNameLower)))),

						),

					),
				)
			} else if val.RelationTypeID == 5 || val.RelationTypeID == 6 {

				h.If(Id(entityNameLower).Op("!=").Nil().Id("&&").Id("args").Dot(entityName).
					Dot(childName + "s").Op("!=").Nil()).Block(

					For(Id("_ ,").Id("dev").Op(":=").Range().Id("*args").Dot(entityName).Dot(childName + "s")).Block(


						If(Id("dev").Dot("Id").Op("==").Nil()).Block(
							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),

							If(Id(childNameLower).Dot("TypeId").Op("!=0 && ").
								Qual(const_UtilsPath,const_UtilsConvertId).
								Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot("TypeId")).Block(
								Comment("todo throw error"),
								Return(Id("&"+entityNameLower+"Resolver{}")),							),

							Id(childNameLower).Dot("TypeId").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).
								Call(Id(entityNameLower).Dot("id")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Post" + childName).
								Params(Id(childNameLower)))),


						).Else().Block(

							Id(childNameLower).Op(":=").Id("ReverseMap" + childName).Params(Id("&dev")),

							If(Id(childNameLower).Dot("TypeId").Op("!=0 && ").
								Qual(const_UtilsPath,const_UtilsConvertId).Call(Id(entityNameLower).Dot("id")).Op("!=").Id(childNameLower).Dot("TypeId")).Block(
								Comment("todo throw error"),
								Return(Id("&"+entityNameLower+"Resolver{}")),							),

							Id(childNameLower).Dot("TypeId").Op("=").Qual(const_UtilsPath,const_UtilsConvertId).
								Call(Id(entityNameLower).Dot("id")),
							Id(entityNameLower).Dot(childNameLower + "s").Op("=").
								Append(Id(entityNameLower).Dot(childNameLower+"s"),Id("Map" + childName).Call(Id("models").Dot("Put" + childName).
								Params(Id(childNameLower)))),

						),

					),
				)
			}

		}

		h.Return(Id("&" + entityNameLower + "Resolver").Op("{").Id(entityNameLower).Op("}"))

	})

}

func createResolver(resolverFile *File, allModels []string) {

	resolverFile.Type().Id("Resolver").Struct()

	for _, val := range allModels {
		valLower := strings.ToLower(val)
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

		//writing root mutation resolvers
		resolverFile.Empty()
		resolverFile.Comment("create resolver for " + val)
		resolverFile.Func().Params(Id("r").Id(" *Resolver")).Id("Create" + val).Params(Id("args").Id("*").StructFunc(func(g *Group) {
			g.Id(val).Id("*" + valLower + "Input")
		})).Params(Id("*" + strings.ToLower(val) + "Resolver")).
			BlockFunc(func(g *Group) {
			g.Return(Qual("", "ResolveCreate" + val)).Call(Id("args"))
		})

		////writing root mutation resolvers
		//resolverFile.Empty()
		//resolverFile.Comment("delete resolver for " + val)
		//resolverFile.Func().Params(Id("r").Id(" *Resolver")).Id("Delete"+val).Params(Id("args").StructFunc(func(g *Group) {
		//	g.Id("ID").Qual(const_GraphQlPath, "ID")
		//})).Params(Id("*" + strings.ToLower(val) + "Resolver")).
		//	BlockFunc(func(g *Group) {
		//	g.Return(Qual("", "ResolveDelete"+val)).Call(Id("args"))
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

func entitiesdeleteResolver(resolverFile *File, entityName string, entity Entity, entityRelationsForAllEndpoint []EntityRelation){
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
	    fmt.Println("dsfsd :", allInterRelation)

	resolverFile.Comment("For Delete")
	resolverFile.Func().Id("ResolveDelete" + entityName).Params(Id("args").StructFunc(func(g *Group) {

		g.Id("ID").Qual(const_GraphQlPath, "ID")
		g.Id("cascadeDelete").Bool()
	})).Params(Id("response *").Int()).BlockFunc(func(g *Group) {


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
				Id("ResolveDeleteEntity").Op(":=").Lit("ResolveDelete").Id("+v"),
				Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),

				//Qual("fmt","Sprintf").Lit("data.").Id("+ v +").Lit(".id"),

				//Id("del").Op("=").Id("temp1").Call(                            //delete child from model
				//     Qual(const_UtilsPath, const_UtilsConvertId).Call(
				//            Id("delId"),
				//     ),
				//     //Id("args.cascadeDelete"),
				//),
				For(Id("_,v1:=").Range().Qual("fmt","Sprintf").Call(Lit("data.").Id("+ v +").Lit(".id"))).Block(

					If(Id("v1").Op("!=").Lit(0)).Block(
						Id("count++"),

						Id("args.ID").Op("=").Qual(const_UtilsPath,const_UtilsRuneToGraphId).Call(Id("v1")),
						//Id("ResolveDeleteEntity").Call(Id("args")),
						Qual("fmt","Sprint").Call(Id("ResolveDeleteEntity+").Lit("(args)")),
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
					Id("ResolveDeleteInterTable").Op(":=").Lit("ResolveDelete").Id("+v.StructName"),
					//     Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
					Qual(const_DatabasePath, "SQL.Model").Call(Lit("models.").Id("+v.StructName+").Lit("{}")).Dot("Joins").Call(Lit("inner join").Id("+data.TableName()+").Lit("on").Id("+data.TableName()+").Lit(".id=").Id("+v.TableName+").Lit(".").Id("+").Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id")).Dot("Where").Call(Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id").Id("+").Lit("=(?)"), Id("args.ID")).Dot("Find").Call(Id("&interData")),

					Id("delId").Op(":=").Id("interData"),
					For(Id("_,v1:=").Range().Id("delId")).Block(

						If(Id("v1.Id").Op("!=").Lit(0)).Block(
							Id("count++"),

							Id("args.ID").Op("=").Qual(const_UtilsPath,const_UtilsUintToGraphId).Call(Id("v1.Id")),
							//Id("ResolveDeleteInterTable").Call(Id("args")),
							Qual("fmt","Sprint").Call(Id("ResolveDeleteInterTable+").Lit("(args)")),

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

			h.Return(Id("response"))
			//),
			//     Else().Block(
			//     Id("del").Op("=").False(),
			//     Id("response").Op("=").Id("&del"),
			//     Return(Id("response")),
			//),

		})

		g.Var().Id("flag").Int()
		g.Var().Id("data").Id("models." + entityName)

		g.For(Id("_,v").Op(":=").Range().Id("models." + entityName + "Children")).Block(

			Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
			//Id("childEntity").Op(":=").Lit("data.").Id("+v"),

			If(Qual("fmt","Sprint").Call(Lit("data.").Id("+v+").Lit(".id")).Op("==").Lit("")).Block(
				Id("flag=1"),
			),

		)

		for _,v:=range allInterRelation {

			g.For(Id("_,v").Op(":=").Range().Id("models." + entityName + "InterRelation")).Block(

				If().Id("v.StructName").Op("==").Lit(v).Block(
					Var().Id("interData []models."+v),

					//Qual(const_DatabasePath, "SQL.Model").Call(Id("models." + entityName).Values()).Dot("Preload").Call(Id("v")).Dot("Find").Call(Id("&data")),
					Qual(const_DatabasePath, "SQL.Model").Call(Lit("models.").Id("+v.StructName+").Lit("{}")).Dot("Joins").Call(Lit("inner join").Id("+data.TableName()+").Lit("on").Id("+data.TableName()+").Lit(".id=").Id("+v.TableName+").Lit(".").Id("+").Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id")).Dot("Where").Call(Qual("strings", "TrimPrefix").Call(Id("data.TableName()"), Lit("x_")).Id("+").Lit("_id").Id("+").Lit("=(?)"), Id("args.ID")).Dot("Find").Call(Id("&interData")),

					//Id("temp").Op(":=").Lit("data.+v"),
					If(Id("len(interData)").Op("==").Lit(0)).Block(
						Id("flag=1"),
					),
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

}

/*
func graphqlStructMaker (lowerName string, capsName string, g *Group, relationTypeID int, isinput bool, isChild bool) {

	var id string



	if isinput == false {

		switch relationTypeID {
		case 1: id = lowerName + " *" + lowerName
		case 2: if isChild == true {
			id = lowerName + " *" + lowerName
		} else {id = lowerName + "s []*" + lowerName}
		case 3: id = lowerName + "s []*" + lowerName
		case 4: id = lowerName + " *" + lowerName
		case 5: if isChild == true {
			id = lowerName + " *" + lowerName
		} else {id = lowerName + "s []*" + lowerName}
		case 6: id = lowerName + "s []*" + lowerName
		}

		*/
/*if relationTypeID == 1 {
			id = lowerName + " *" + lowerName
		}
		if relationTypeID == 2 && isChild == true{
			id = lowerName + " *" + lowerName
		}
		if relationTypeID == 2 && isChild == false{
			id = lowerName + "s []*" + lowerName
		}
		if relationTypeID == 3 {
			id = lowerName + "s []*" + lowerName
		}*//*

		g.Id(id)
	}
	if isinput == true {

		if relationTypeID == 1 || relationTypeID == 4{
			id = capsName+" *"+lowerName+"Input"
		}
		if relationTypeID == 2 || relationTypeID == 3 || relationTypeID == 5 || relationTypeID == 6{
			id = capsName+"s *[]"+lowerName+"Input"
		}
		g.Id(id)
	}
	return
}
*/
