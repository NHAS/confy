package confy

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const confyTag = "confy"

var (
	supportedTags = []string{
		"json",
		"yaml",
		"toml",
	}
)

type fieldsData struct {
	path  []string
	value reflect.Value
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
		fieldName := typeData.Field(i).Name

		if fieldVal.Type().Kind() == reflect.Struct {
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
			})
		}
	}
	return fields
}

func setField(v interface{}, fieldPath []string, value reflect.Value) {
	r := reflect.ValueOf(v).Elem()

	for i, part := range fieldPath {
		if i == len(fieldPath)-1 {
			f := r.FieldByName(part)
			if f.IsValid() && f.CanAddr() {
				f.Set(value)
			} else {

				if !f.IsValid() {
					log.Printf("Field not found: %s", fieldPath)
				}

				if !f.CanAddr() {
					log.Printf("Cant address: %s", fieldPath)

				}
			}
		} else {
			r = r.FieldByName(part)
		}
	}
}

func getAllTagNames(tag reflect.StructTag) (result []string) {

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			return
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			return
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		tag = tag[i+1:]

		result = append(result, name)
	}

	return
}

type configDecoder interface {
	Decode(v any) (err error)
}

func LoadConfigAuto[T any](path string, strict bool) (result T, err error) {
	return LoadConfig[T](path, strict, Auto)
}

func LoadConfig[T any](path string, strict bool, configType ConfigType) (result T, err error) {
	clone, err := cloneWithNewTags(result)
	if err != nil {
		return *new(T), err
	}

	if configType == Auto {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yml", ".yaml":
			configType = Yaml
		case ".json", ".js":
			configType = Json
		case ".toml", ".tml":
			configType = Toml
		default:
			return result, fmt.Errorf("unsupported extension %q", strings.ToLower(filepath.Ext(path)))
		}
	}

	configFile, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("failed to open config file %q, err: %s", path, err)
	}

	var decoder configDecoder
	switch configType {
	case Json:
		jsDec := json.NewDecoder(configFile)
		if strict {
			jsDec.DisallowUnknownFields()
		}
		decoder = jsDec
	case Yaml:
		ymDec := yaml.NewDecoder(configFile)
		ymDec.KnownFields(strict)
		decoder = ymDec
	case Toml:
		tmlDec := toml.NewDecoder(configFile)
		if strict {
			tmlDec = tmlDec.DisallowUnknownFields()
		}
		decoder = tmlDec
	}

	err = decoder.Decode(clone)
	if err != nil {
		return result, fmt.Errorf("failed to decode config: %s", err)
	}

	PrettyPrintStruct(clone)

	fields := getFields(clone)

	for _, value := range fields {

		log.Println(value.path, value.value.Interface())
		setField(&result, value.path, value.value)
	}

	return result, nil
}

func PrettyPrintStruct(s interface{}) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	// Ensure we're working with a struct
	if t.Kind() != reflect.Struct {
		fmt.Println("Provided value is not a struct: ", t.Kind())
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get tag if exists
		tag := field.Tag
		if tag != "" {
			fmt.Printf("%s (tag: %s) = %v\n", field.Name, tag, value)
		} else {
			fmt.Printf("%s = %v\n", field.Name, value)
		}
	}
}

// CloneWithNewTags creates a new struct with modified tags, leaves it blank
func cloneWithNewTags(v interface{}) (interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input config was not struct")
	}

	// Create new struct type with modified tags
	newType := createModifiedType(val.Type())

	// Create new struct instance
	newValue := reflect.New(newType)

	return newValue.Interface(), nil
}

func createModifiedType(t reflect.Type) reflect.Type {
	fields := make([]reflect.StructField, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		newField := reflect.StructField{
			Name:      field.Name,
			Type:      field.Type,
			Tag:       field.Tag,
			Anonymous: field.Anonymous,
			PkgPath:   field.PkgPath,
		}

		// Handle nested structs
		if field.Type.Kind() == reflect.Struct {
			newField.Type = createModifiedType(field.Type)
		}

		existingTagNames := getAllTagNames(field.Tag)
		confyTagNames := map[string]string{}

		var fieldMarshallingName string
		confyInstruction, ok := field.Tag.Lookup(confyTag)
		if ok {
			parts := strings.Split(confyInstruction, ";")
			if len(parts) > 0 {
				fieldMarshallingName = parts[0]
			}
		}

		if fieldMarshallingName != "" {
			for _, supportedTag := range supportedTags {
				confyTagNames[supportedTag] = fieldMarshallingName
			}
		}

		alreadySetTags := map[string]bool{}
		tagsToSet := []string{}
		for _, tagName := range existingTagNames {
			// Preserve existing tags
			value, ok := field.Tag.Lookup(tagName)
			if !ok {
				continue
			}

			alreadySetTags[tagName] = true

			tagsToSet = append(tagsToSet, fmt.Sprintf("%s:\"%s\"", tagName, value))

		}

		for confyTagName, confyTagValue := range confyTagNames {
			if alreadySetTags[confyTagName] {
				continue
			}
			tagsToSet = append(tagsToSet, fmt.Sprintf("%s:\"%s\"", confyTagName, confyTagValue))
		}

		newField.Tag = reflect.StructTag(strings.Join(tagsToSet, " "))

		fields[i] = newField
	}

	return reflect.StructOf(fields)
}
