package confy

import (
	"encoding"
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
			logger.Warn("unable to access field", "field_name", fieldName, "can_intf", fieldVal.CanInterface(), "can_addr", fieldVal.CanAddr())
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
	t := r.Type()

	for i, part := range fieldPath {
		if i == len(fieldPath)-1 {
			f := r.FieldByName(part)
			ft, ok := t.FieldByName(part)
			if !ok {
				logger.Error("failed to get type by field name", "field_name", part)
			}
			if f.IsValid() {
				return f, ft
			} else {
				logger.Error("field was invalid when searching struct", "field_name", part)
			}
		} else {
			r = r.FieldByName(part)
			t = r.Type()
		}
	}

	return reflect.Value{}, reflect.StructField{}
}

func resolvePath(v interface{}, fieldPath []string) []string {
	resolvedPath := []string{}
	for i := range fieldPath {

		currentPath := fieldPath[i]
		_, ft := getField(v, fieldPath[:i+1])

		value, ok := ft.Tag.Lookup(confyTag)
		if ok {

			parts := strings.Split(value, ";")
			if len(parts) > 0 && parts[0] != "" {
				currentPath = parts[0]
			}
		}

		logger.Info("resolving path", "tags", ft.Tag, "had_confy_tag", ok, "current_path", fieldPath[:i+1])

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

// determineVariableName returns the variable name after resolving and transforming
// ok: bool indicates whether this is a decodable type
func determineVariableName[T any](result *T, delimiter string, transform Transform, field fieldsData) (string, bool) {
	variable := strings.Join(resolvePath(result, field.path), delimiter)
	if transform != nil {
		variable = transform(variable)
		logger.Info("using transform func on variable", "before_func", strings.Join(resolvePath(result, field.path), delimiter), "after_func", variable)
	}

	if field.value.Kind() == reflect.Struct {
		current := field.value
		_, ok := current.Addr().Interface().(encoding.TextUnmarshaler)
		if !ok {
			logger.Warn("type doesnt implement encoding.TextUnmarshaler skipping looking for an ENV variable for it", "path", strings.Join(field.path, delimiter))
			return "", false
		}
	}

	if field.value.Kind() == reflect.Array || field.value.Kind() == reflect.Slice {
		sliceContentType := field.value.Type().Elem()

		switch sliceContentType.Kind() {
		case reflect.String, reflect.Int, reflect.Int64, reflect.Float64, reflect.Bool:
		default:
			inter := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
			if !reflect.PointerTo(sliceContentType).Implements(inter) {
				logger.Warn("type inside of complex slice did not implement encoding.TextUnmarshaler", "path", strings.Join(field.path, delimiter))
				return "", false
			}
		}
	}

	return variable, true
}
