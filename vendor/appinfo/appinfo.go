package appinfo

type AppInfo struct {
	Name          string
	FieldTypes    []FieldType
	Entities      []Entity
	RelationTypes []RelationType
	Relations     []Relation
}

type FieldType struct {
	Id   int
	Name string
}

type Field struct {
	Name        string
	DisplayName string
	Type        int
	Size        int
}

type Entity struct {
	Name        string
	DisplayName string
	Fields      []Field
}

type RelationType struct {
	Id   int
	Name string
}

type Relation struct {
	ParentEntity      string
	ParentEntityField string
	ChildEntity       string
	ChildEntityField  string
	Pivot             string
	Type              int
}
