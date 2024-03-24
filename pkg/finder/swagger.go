package finder

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi"
	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
	"github.com/pb33f/libopenapi/orderedmap"
)

type API struct {
	Method  string
	URL     string
	Headers map[string]string
	Content string
}

type Param struct {
	name  string
	value string
	in    string
}

func FindLinksFromSwagger(source []byte) ([]*API, error) {
	doc, err := libopenapi.NewDocument(source)

	if err != nil {
		return nil, fmt.Errorf("cannot create new document: %e", err)
	}

	var errs []error
	var v2Model *libopenapi.DocumentModel[v2.Swagger]

	v2Model, errs = doc.BuildV2Model()

	if len(errs) > 0 {
		for i := range errs {
			fmt.Printf("error: %e\n", errs[i])
		}
		return nil, fmt.Errorf("cannot create v2 model from document: %d errors reported", len(errs))
	}

	if v2Model.Model.Paths == nil {
		return nil, errors.New("there is no paths in document")
	}

	basePath := v2Model.Model.BasePath
	paths := v2Model.Model.Paths.PathItems
	var (
		defMap   map[string]string
		paramMap map[string]*Param
	)
	if v2Model.Model.Definitions != nil {
		defMap = parseDefinitions(*v2Model.Model.Definitions)
	}
	if v2Model.Model.Parameters != nil {
		paramMap = parseParameters(*v2Model.Model.Parameters)
	}

	var result []*API
	for pathPair := paths.First(); pathPair != nil; pathPair = pathPair.Next() {
		api := &API{
			URL:     filepath.Join(basePath, pathPair.Key()),
			Headers: make(map[string]string),
		}

		for op := pathPair.Value().GetOperations().First(); op != nil; op = op.Next() {
			api.Method = op.Key()
			consumes := op.Value().Consumes
			if len(consumes) > 0 {
				api.Headers["Content-Type"] = consumes[0]
			}

			var value string
			for i := 0; i < len(op.Value().Parameters); i++ {
				param := op.Value().Parameters[i]

				// TODO: Parameter Definitions
				if param.Name == "$ref" {
					fmt.Println("This parameter is a reference:", param)
					fmt.Println(paramMap)
				}

				if param.Default != nil {
					value = param.Default.Value
				} else {
					if param.Schema.IsReference() {
						ref := param.Schema.GetReference()
						value = defMap[getRefName(ref)]
					} else if items := param.Items; items != nil {
						if param.GoLow().Items.IsReference() {
							ref := param.GoLow().Items.GetReference()
							value = "[" + defMap[getRefName(ref)] + "]"
						} else if items.Format != "" {
							value = getValueByFormat(items.Format, false)
						} else {
							value = getValueByFormat(items.Type, false)
						}
					} else if param.Format != "" {
						value = getValueByFormat(param.Format, false)
					} else if param.Type != "" {
						value = getValueByType(param.Type, false)
					}
				}

				switch param.In {
				case "path":
					api.URL = strings.ReplaceAll(api.URL, fmt.Sprintf("{%s}", param.Name), value)
				case "query":
					if !strings.Contains(api.URL, "?") {
						api.URL += fmt.Sprintf("?%s=%s", param.Name, value)
					} else {
						api.URL += fmt.Sprintf("&%s=%s", param.Name, value)
					}
				case "formData":
					if api.Content == "" {
						api.Content += fmt.Sprintf("%s=%s", param.Name, value)
					} else {
						api.Content += fmt.Sprintf("&%s=%s", param.Name, value)
					}
				case "header":
					api.Headers[param.Name] = value
				default:
					api.Content = value
				}
			}

			result = append(result, api)
		}
	}

	return result, nil
}

func parseParameters(model v2.ParameterDefinitions) map[string]*Param {
	result := make(map[string]*Param)

	for pair := model.Definitions.First(); pair != nil; pair = pair.Next() {
		p := pair.Value()
		var value string

		if p.Default != nil {
			value = p.Default.Value
		} else {
			value = getValueByType(p.Type, false)
		}

		result[p.Name] = &Param{
			name:  p.Name,
			value: value,
			in:    p.In,
		}
	}

	return result
}

func parseDefinitions(model v2.Definitions) map[string]string {
	result := make(map[string]string)
	schemas := model.Definitions

	for sPair := schemas.First(); sPair != nil; sPair = sPair.Next() {
		schema := sPair.Key()
		schemaVal := sPair.Value()
		result[schema] = parseProperties(schemaVal.Schema().Properties)
	}
	return result
}

// parseProperties 根据 properties 生成 json 字符串
func parseProperties(properties *orderedmap.Map[string, *base.SchemaProxy]) string {
	var jsonStr string

	for pair := properties.First(); pair != nil; pair = pair.Next() {
		propName := pair.Key()
		prop := pair.Value().Schema()

		if prop.Example != nil {
			jsonStr += fmt.Sprintf("\"%s\":\"%s\",", propName, prop.Example.Value)
		} else if prop.Properties.First() != nil {
			// 如果当前 property 存在 Properties 属性表示通过 $ref 引用了其他 schema
			jsonStr += fmt.Sprintf("\"%s\":%s,", propName, parseProperties(prop.Properties))
		} else if prop.Items != nil {
			// TODO: parse additionalProperties
			if items := prop.Items.A.Schema(); items.Example != nil {
				jsonStr += fmt.Sprintf("\"%s\":[\"%s\"],", propName, items.Example.Value)
			} else if items.Properties != nil {
				// 当前 property 为其他 schema 的列表
				jsonStr += fmt.Sprintf("\"%s\":[%s],", propName, parseProperties(items.Properties))
			} else if items.Format != "" {
				jsonStr += fmt.Sprintf("\"%s\":[%s],", propName, getValueByFormat(items.Format, true))
			} else {
				jsonStr += fmt.Sprintf("\"%s\":[%s],", propName, getValueByType(items.Type[0], true))
			}
		} else if prop.Format != "" {
			jsonStr += fmt.Sprintf("\"%s\":%s,", propName, getValueByFormat(prop.Format, true))
		} else if len(prop.Type) > 0 {
			jsonStr += fmt.Sprintf("\"%s\":%s,", propName, getValueByType(prop.Type[0], true))
		}
	}

	return "{" + strings.TrimRight(jsonStr, ",") + "}"
}

func getValueByType(dataType string, quote bool) string {
	switch dataType {
	case "integer":
		return "1"
	case "number":
		return "1.0"
	case "string":
		if quote {
			return `"test"`
		} else {
			return "test"
		}
	case "boolean":
		return "true"
	case "array":
		return "[]"
	case "object":
		return "{}"
	case "file":
		if quote {
			return `"TODO"`
		} else {
			return "TODO"
		}
	default:
		return ""
	}
}

// see: https://graphql-faas.github.io/OpenAPI-Specification/versions/2.0.html#data-types
func getValueByFormat(format string, quote bool) string {
	switch format {
	case "int32", "int64":
		return "1"
	case "float", "double":
		return "1.0"
	case "byte": // base64 encoded characters
		if quote {
			return `"dGVzdA=="`
		} else {
			return "dGVzdA=="
		}
	case "binary": // any sequence of octets
		if quote {
			return `"TODO"`
		} else {
			return "TODO"
		}
	case "date", "date-time": // RFC3339
		if quote {
			return `"1985-04-12T23:20:50.52Z"`
		} else {
			return "1985-04-12T23:20:50.52Z"
		}
	case "password": // Used to hint UIs the input needs to be obscured
		if quote {
			return `"TODO"`
		} else {
			return "TODO"
		}
	case "email":
		if quote {
			return `"gatherer@1234.com"`
		} else {
			return "gatherer@1234.com"
		}
	default:
		// format 的要求比较宽松，可能存在将 type 填在 format 字段的情况
		return getValueByType(format, quote)
	}
}

func getRefName(ref string) string {
	return strings.Replace(string(ref), "#/definitions/", "", -1)
}
