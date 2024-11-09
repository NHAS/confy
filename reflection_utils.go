package confy

import (
	"reflect"
	"strings"
)

type fieldsData struct {
	path  []string
	value reflect.Value
	tag   reflect.StructTag
}

func getFields(returnStructs bool, v interface{}) []fieldsData {
	t := reflect.ValueOf(v)
	typeData := reflect.TypeOf(v)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		typeData = typeData.Elem()
	}

	if t.Kind() != reflect.Struct {

		return []fieldsData{
			{
				path:  []string{typeData.Name()},
				value: t,
			},
		}
	}

	var fields []fieldsData

	for i := 0; i < t.NumField(); i++ {
		fieldVal := t.Field(i)

		fieldTag := typeData.Field(i).Tag
		fieldName := typeData.Field(i).Name

		if !fieldVal.CanInterface() || !fieldVal.CanAddr() {
			continue
		}

		if fieldVal.Type().Kind() == reflect.Struct {

			if returnStructs {
				fields = append(fields, fieldsData{
					path:  []string{fieldName},
					value: fieldVal,
					tag:   fieldTag,
				})
			}

			if fieldVal.CanAddr() {
				fieldVal = fieldVal.Addr()
			}

			subFields := getFields(returnStructs, fieldVal.Interface())
			for _, value := range subFields {

				currentFieldPath := value
				currentFieldPath.path = append([]string{fieldName}, value.path...)

				fields = append(fields, currentFieldPath)
			}
		} else {
			fields = append(fields, fieldsData{
				path: []string{
					fieldName,
				},
				value: fieldVal,
				tag:   fieldTag,
			})
		}
	}

	return fields
}

func getField(v interface{}, fieldPath []string) (reflect.Value, reflect.StructField) {
	r := reflect.ValueOf(v).Elem()
	t := reflect.TypeOf(v).Elem()

	for i, part := range fieldPath {
		if i == len(fieldPath)-1 {
			f := r.FieldByName(part)
			ft, _ := t.FieldByName(part)
			if f.IsValid() {
				return f, ft
			}
		} else {
			r = r.FieldByName(part)
		}
	}

	return reflect.Value{}, reflect.StructField{}
}

func resolvePath(v interface{}, fieldPath []string) []string {
	resolvedPath := []string{}
	for i := range fieldPath {

		currentPath := fieldPath[i]
		_, ft := getField(v, fieldPath[:i+1])
		if value, ok := ft.Tag.Lookup(confyTag); ok {
			currentPath = value
		}

		resolvedPath = append(resolvedPath, currentPath)
	}
	return resolvedPath
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func maskSensitive(value string, tag reflect.StructTag) string {

	printedValue := value

	isSensitive := false
	value, ok := tag.Lookup(confyTag)
	if ok {
		parts := strings.Split(value, ";")
		if len(parts) > 1 {
			isSensitive = strings.TrimSpace(parts[1]) == "sensitive"
		}
	}

	if isSensitive && value != "" {
		printedValue = "**********"
	}

	return printedValue
}
