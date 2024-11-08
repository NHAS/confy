package confy

import (
	"log"
	"reflect"
	"strconv"
	"strings"
)

type fieldsData struct {
	path  []string
	value reflect.Value
	tag   reflect.StructTag
}

func getFields(v interface{}) []fieldsData {
	t := reflect.ValueOf(v)
	typeData := reflect.TypeOf(v)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		typeData = typeData.Elem()
	}

	fields := []fieldsData{}

	if t.Kind() != reflect.Struct {
		return []fieldsData{
			{
				path:  []string{typeData.Name()},
				value: t,
			},
		}
	}

	for i := 0; i < t.NumField(); i++ {
		fieldVal := t.Field(i)

		fieldTag := typeData.Field(i).Tag
		fieldName := typeData.Field(i).Name

		if fieldVal.Type().Kind() == reflect.Struct {

			if fieldVal.CanAddr() {
				fieldVal = fieldVal.Addr()
			}

			subFields := getFields(fieldVal.Interface())
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

func setBasicField(v interface{}, fieldPath string, value string) {
	r := reflect.ValueOf(v).Elem()
	parts := strings.Split(fieldPath, ".")

	for i, part := range parts {
		if i == len(parts)-1 {
			f := r.FieldByName(part)
			if f.IsValid() {
				switch f.Kind() {
				case reflect.Bool:
					f.SetBool(value == "true")
				case reflect.Slice:
					f.Set(reflect.ValueOf(strings.Split(value, ",")))
				case reflect.String:
					f.SetString(value)
				case reflect.Int:

					reflectedVal, err := strconv.Atoi(value)
					if err != nil {
						log.Println(fieldPath, " should be int, but couldnt be parsed as one: ", err)
						continue
					}
					f.SetInt(int64(reflectedVal))

				default:
					log.Printf("Unsupported field type for field: %s", fieldPath)
				}
			} else {
				log.Printf("Field not found: %s", fieldPath)
			}
		} else {
			r = r.FieldByName(part)
		}
	}
}
