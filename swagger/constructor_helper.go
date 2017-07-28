package swagger

import (
	"github.com/go-openapi/spec"
	"github.com/Soontao/go-mysql-api/mysql"
	"fmt"
)

func NewRefSchema(refDefinationName, reftype string) (s spec.Schema) {
	s = spec.Schema{
		spec.VendorExtensible{},
		spec.SchemaProps{
			Type: spec.StringOrArray{reftype},
			Items: &spec.SchemaOrArray{
				&spec.Schema{
					spec.VendorExtensible{},
					spec.SchemaProps{
						Ref: getTableSwaggerRef(refDefinationName),
					},
					spec.SwaggerSchemaProps{},
					nil,
				},
				nil,
			},
		},
		spec.SwaggerSchemaProps{},
		nil,
	}
	return
}

func NewField(sName, sType string, iExample interface{}) (s spec.Schema) {
	s = spec.Schema{
		spec.VendorExtensible{},
		spec.SchemaProps{
			Type:  spec.StringOrArray{sType},
			Title: sName,
		},
		spec.SwaggerSchemaProps{
			Example: iExample,
		},
		nil,
	}
	return
}

func NewCUDOperationReturnMessage() (s spec.Schema) {
	s = spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"object"},
			Properties: map[string]spec.Schema{
				"lastInsertID":  NewField("lastInsertID", "integer", 0),
				"rowesAffected": NewField("rowesAffected", "integer", 1),
			},
		},
	}
	return
}

func NewCUDOperationReturnArrayMessage() (s spec.Schema) {
	s = spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"array"},
			Items: &spec.SchemaOrArray{
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Properties: map[string]spec.Schema{
							"lastInsertID":  NewField("lastInsertID", "integer", 0),
							"rowesAffected": NewField("rowesAffected", "integer", 1),
						},
					},
				},
			},

		},
	}
	return
}

func NewDefinitionMessageWrap(definitionName string, data spec.Schema) (sWrap *spec.Schema) {

	sWrap = &spec.Schema{
		spec.VendorExtensible{},
		spec.SchemaProps{
			Type: spec.StringOrArray{"object"},
			Properties: map[string]spec.Schema{
				"status":  NewField("status", "integer", 200),
				"message": NewField("message", "string", nil),
				"data":    data,
			},
		},
		spec.SwaggerSchemaProps{},
		nil,
	}
	return
}

func NewSwaggerInfo(title, version string) (info *spec.Info) {
	info = &spec.Info{spec.VendorExtensible{}, spec.InfoProps{
		Title:   title,
		Version: version,
	}}
	return
}

func GetParametersFromDbMetadata(meta *mysql.DataBaseMetadata) (params map[string]spec.Parameter) {
	params = make(map[string]spec.Parameter)
	for _, t := range meta.Tables {
		for _, col := range t.Columns {
			params[col.ColumnName] = spec.Parameter{
				ParamProps: spec.ParamProps{
					In:          "body",
					Description: col.Comment,
					Name:        col.ColumnName,
					Required:    col.NullAble == "true",
				},
			}
		}
	}
	return
}

func NewQueryParametersForMySQLAPI() (ps []spec.Parameter) {
	ps = []spec.Parameter{
		NewQueryParameter("_field", "include the field", "string", false),
		NewQueryParameter("_limit", "limit max records num", "integer", false),
		NewQueryParameter("_skip", "skip first some records", "integer", false),
		NewQueryParameter("_where", "filter with field value", "string", false),
		NewQueryParameter("_link", "auto join a table", "string", false),
	}
	return
}

func NewQueryParameter(paramName, paramDescription, paramType string, required bool) (p spec.Parameter) {
	p = spec.Parameter{
		SimpleSchema: spec.SimpleSchema{
			Type: paramType,
		},
		ParamProps: spec.ParamProps{
			In:          "query",
			Name:        paramName,
			Required:    required,
			Description: paramDescription,
		},
	}
	return
}

func NewPathIDParameter(tMeta *mysql.TableMetadata) (p spec.Parameter) {
	p = spec.Parameter{
		SimpleSchema: spec.SimpleSchema{
			Type: "string",
		},
		ParamProps: spec.ParamProps{
			In:          "path",
			Name:        "id",
			Required:    true,
			Description: fmt.Sprintf("%s %s", tMeta.TableName, tMeta.GetPrimaryColumn().ColumnName),
		},
	}
	return
}

func NewParamForArrayDefinition(tName string) (p spec.Parameter) {
	s := NewRefSchema(tName, "array")
	p = spec.Parameter{
		ParamProps: spec.ParamProps{
			In:     "body",
			Name:   tName,
			Schema: &s,
		},
	}
	return
}

func NewParamForDefinition(tName string) (p spec.Parameter) {
	p = spec.Parameter{
		ParamProps: spec.ParamProps{
			In:     "body",
			Name:   tName,
			Schema: getTableSwaggerRefSchema(tName),
		},
	}
	return
}

func NewOperation(tName, opDescribetion string, params []spec.Parameter, respSchemaProps spec.SchemaProps) (op *spec.Operation) {
	op = &spec.Operation{
		spec.VendorExtensible{}, spec.OperationProps{
			Description: opDescribetion,
			Tags:        []string{tName},
			Parameters:  params,
			Responses: &spec.Responses{
				spec.VendorExtensible{},
				spec.ResponsesProps{
					nil,
					map[int]spec.Response{
						200: spec.Response{
							spec.Refable{},
							spec.ResponseProps{
								Description: "success",
								Schema: &spec.Schema{
									spec.VendorExtensible{},
									respSchemaProps,
									spec.SwaggerSchemaProps{},
									nil,
								},
							},
							spec.VendorExtensible{},
						},
					},
				},
			},
		},
	}
	return
}

func NewTag(t string) (tag spec.Tag) {
	tag = spec.Tag{TagProps: spec.TagProps{Name: t}}
	return
}

func NewTagsForOne(t string) (tags []spec.Tag) {
	tags = []spec.Tag{NewTag(t)}
	return
}