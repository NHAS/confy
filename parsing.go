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

func setStruct(targetStruct, value reflect.Value) reflect.Value {
	newStruct := reflect.New(targetStruct.Type()).Elem()

	// Copy fields from elem to newElem
	for k := 0; k < value.NumField(); k++ {
		currentField := newStruct.Field(k)

		if currentField.Kind() == reflect.Array || currentField.Kind() == reflect.Slice {
			currentField.Set(setArray(currentField, value.Field(k)))
			continue
		}

		if currentField.Kind() == reflect.Struct {
			currentField.Set(setStruct(currentField, value.Field(k)))
			continue
		}

		currentField.Set(value.Field(k))
	}

	return newStruct
}

func setArray(targetArray, values reflect.Value) reflect.Value {

	var result reflect.Value
	if targetArray.Type().Kind() == reflect.Slice {
		result = reflect.MakeSlice(targetArray.Type(), values.Len(), values.Cap())
	} else if targetArray.Type().Kind() == reflect.Array {
		result = reflect.New(targetArray.Type()).Elem()
	}

	for j := 0; j < values.Len(); j++ {
		existingElement := values.Index(j)

		if existingElement.Kind() == reflect.Struct {

			newElem := reflect.New(targetArray.Type().Elem()).Elem()
			log.Println(newElem.Type())

			// Copy fields from elem to newElem
			for k := 0; k < existingElement.NumField(); k++ {
				currentField := newElem.Field(k)

				if currentField.Kind() == reflect.Array || currentField.Kind() == reflect.Slice {
					currentField.Set(setArray(currentField, existingElement.Field(k)))
					continue
				}

				if currentField.Kind() == reflect.Struct {
					currentField.Set(setStruct(currentField, existingElement.Field(k)))
					continue
				}

				currentField.Set(existingElement.Field(k))
			}

			result.Index(j).Set(newElem)
		} else {
			// Directly copy non-complex types
			result.Index(j).Set(existingElement)
		}
	}

	return result
}

func setField(v interface{}, fieldPath []string, value reflect.Value) {
	r := reflect.ValueOf(v).Elem()

	for i, part := range fieldPath {
		if i == len(fieldPath)-1 {
			f := r.FieldByName(part)
			if f.IsValid() && f.CanAddr() {
				if f.Type().Kind() != reflect.Array && f.Type().Kind() != reflect.Slice {
					f.Set(value)
				} else {
					// Due to the yaml parser being incredibly dumb, we have had to recursively go in to every struct
					// and make sure it has a yaml tag if the type is complex
					f.Set(setArray(f, value))
				}
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

	fields := getFields(clone)

	for _, value := range fields {
		setField(&result, value.path, value.value)
	}

	return result, nil
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

func createModifiedArray(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Array:
		// Modify the element type of the array, handling nested structs and arrays recursively
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			elemType = createModifiedType(elemType) // Modify nested structs
		} else if elemType.Kind() == reflect.Array || elemType.Kind() == reflect.Slice {
			elemType = createModifiedArray(elemType) // Handle nested arrays or slices
		}
		// Return a new array type with the modified element type
		return reflect.ArrayOf(t.Len(), elemType)

	case reflect.Slice:
		// Modify the element type of the slice
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			elemType = createModifiedType(elemType) // Modify nested structs
		} else if elemType.Kind() == reflect.Array || elemType.Kind() == reflect.Slice {
			elemType = createModifiedArray(elemType) // Handle nested arrays or slices
		}
		// Return a new slice type with the modified element type
		return reflect.SliceOf(elemType)

	default:
		// If not an array or slice, return the type unchanged
		return t
	}
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
		} else if field.Type.Kind() == reflect.Array || field.Type.Kind() == reflect.Slice {
			newField.Type = createModifiedArray(field.Type)
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
		} else {

			// because the go-yaml parser only maps things automatically if they're lower case, add this
			// to match other parsers
			confyTagNames["yaml"] = field.Name
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
