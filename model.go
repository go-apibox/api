// Model管理器

package api

import (
	"reflect"
	"strings"

	"github.com/go-xorm/xorm"
)

type ModelManager struct {
	Models map[string]*ModelDefine
}

type ModelDefine struct {
	Type          reflect.Type
	MainModelName string
	tableName     string
	fields        []string // 所有字段名，包括参照表的
	mainFields    []string // 主表字段名
	refFields     []string // 参照表字段名
	fieldTags     map[string][]*ModelTag
	tagFields     map[string][]string
}

type ModelTag struct {
	Name   string
	Params []string
}

func NewModelManager() *ModelManager {
	mm := new(ModelManager)
	mm.Models = make(map[string]*ModelDefine)
	return mm
}

// Register add new model to this manager.
func (mm *ModelManager) Register(model interface{}) {
	var tableName string
	method := reflect.ValueOf(model).MethodByName("TableName")
	if method.Kind() == reflect.Func {
		v := method.Call([]reflect.Value{})
		if len(v) == 1 {
			tableName, _ = v[0].Interface().(string)
		}
	}

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = reflect.Indirect(reflect.ValueOf(model)).Type()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	pkFound := false
	hasAnonymous := false
	m := ModelDefine{
		t, "", tableName, []string{}, []string{}, []string{},
		make(map[string][]*ModelTag), make(map[string][]string),
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			hasAnonymous = true
			if m.MainModelName == "" {
				m.MainModelName = field.Name
			}
			// 匿名字段则展开
			tt := field.Type
			for j := 0; j < tt.NumField(); j++ {
				field := tt.Field(j)
				tags := getTags(field, pkFound)
				m.fieldTags[field.Name] = tags
				m.fields = append(m.fields, field.Name)
				m.mainFields = append(m.mainFields, field.Name)
				for _, tag := range tags {
					if tag.Name == "pk" {
						if !pkFound {
							pkFound = true
							m.tagFields[tag.Name] = append(m.tagFields[tag.Name], field.Name)
						}
					} else {
						m.tagFields[tag.Name] = append(m.tagFields[tag.Name], field.Name)
					}
				}
			}
		} else {
			tags := getTags(field, pkFound)
			m.fieldTags[field.Name] = tags
			m.fields = append(m.fields, field.Name)
			if hasAnonymous {
				m.refFields = append(m.refFields, field.Name)
			} else {
				m.mainFields = append(m.mainFields, field.Name)
			}
			for _, tag := range tags {
				if tag.Name == "pk" {
					if !pkFound {
						pkFound = true
						m.tagFields[tag.Name] = append(m.tagFields[tag.Name], field.Name)
					}
				} else {
					m.tagFields[tag.Name] = append(m.tagFields[tag.Name], field.Name)
				}
			}
		}
	}
	if m.MainModelName == "" {
		m.MainModelName = t.Name()
	}
	mm.Models[t.Name()] = &m
}

// func getColumnName(field reflect.StructField) string {
// 	tags := strings.Split(field.Tag.Get("xorm"), " ")
// 	for _, tag := range tags {
// 		tagParts := strings.SplitN(tag, ":", 2)
// 		tagName := tagParts[0]
// 		if tagName[0] == '\'' {
// 			return strings.Trim(tagName, "'")
// 		}
// 	}

// 	return ""
// }

func getTags(field reflect.StructField, ignorePk bool) []*ModelTag {
	tags := strings.Split(field.Tag.Get("api"), " ")
	mTags := make([]*ModelTag, 0)

	// 从xorm中复制字段名
	xormTags := strings.Split(field.Tag.Get("xorm"), " ")
	for _, tag := range xormTags {
		tagParts := strings.SplitN(tag, ":", 2)
		tagName := tagParts[0]
		if len(tagName) > 0 {
			if tagName[0] == '\'' {
				tags = append(tags, "column:"+strings.Trim(tagName, "'"))
				break
			}
		}
	}

	if !ignorePk {
		// 从xorm中复制pk标记
		xormTags := strings.Split(field.Tag.Get("xorm"), " ")
		for _, v := range xormTags {
			if v == "pk" {
				tags = append(tags, "pk")
				break
			}
		}
	}

	var tagParts []string
	var tagName string
	var tagParams string
	for _, tag := range tags {
		tagParts = strings.SplitN(tag, ":", 2)
		tagName = tagParts[0]

		if len(tagParts) == 1 {
			mTags = append(mTags, &ModelTag{tagName, []string{}})
		} else {
			tagParams = tagParts[1]
			mTags = append(mTags, &ModelTag{tagName, strings.Split(tagParams, ",")})
		}
	}

	return mTags
}

// func getColumnName(engine *xorm.Engine, bean interface{}, fieldName string) string {
// 	for _, col := range engine.TableInfo(bean).Columns() {
// 		if col.FieldName == fieldName {
// 			return col.Name
// 		}
// 	}
// 	return ""
// }

// Unregister remove existing model from this manager.
func (mm *ModelManager) Unregister(modelName string) {
	delete(mm.Models, modelName)
}

// Has check if specified model is registered.
func (mm *ModelManager) Has(modelName string) bool {
	_, has := mm.Models[modelName]
	return has
}

// Get return specified model.
func (mm *ModelManager) Get(modelName string) *ModelDefine {
	m, has := mm.Models[modelName]
	if !has {
		return nil
	}
	return m
}

// TableName return table name of model.
func (m *ModelDefine) TableName(engine *xorm.Engine) string {
	if m.tableName != "" {
		return m.tableName
	}
	return engine.TableMapper.Obj2Table(m.MainModelName)
}

// Fields return all fields of model.
func (m *ModelDefine) Fields() []string {
	return m.fields
}

// Fields return fields of main table of model.
func (m *ModelDefine) MainFields() []string {
	return m.mainFields
}

// Fields return fields of reference table of model.
func (m *ModelDefine) RefFields() []string {
	return m.refFields
}

// TagFields return fields with specified tag name.
func (m *ModelDefine) TagFields(tagName string) []string {
	fields, ok := m.tagFields[tagName]
	if !ok {
		return []string{}
	}
	return fields
}

// FieldTags return tags of specified field.
func (m *ModelDefine) FieldTags(field string) []*ModelTag {
	tags, ok := m.fieldTags[field]
	if !ok {
		return []*ModelTag{}
	}
	return tags
}

// FieldHasTag check if field has specified tag.
func (m *ModelDefine) FieldHasTag(field, tagName string) bool {
	tags, ok := m.fieldTags[field]
	if !ok {
		return false
	}
	for _, tag := range tags {
		if tag.Name == tagName {
			return true
		}
	}
	return false
}

// FieldGetTag get specified tag of field.
func (m *ModelDefine) FieldGetTag(field, tagName string) *ModelTag {
	tags, ok := m.fieldTags[field]
	if !ok {
		return nil
	}
	for _, tag := range tags {
		if tag.Name == tagName {
			return tag
		}
	}
	return nil
}
