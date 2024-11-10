package confy

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type configParser[T any] struct {
	o             *options
	supportedTags []string
}

func (cp *configParser[T]) setStruct(targetStruct, value reflect.Value) reflect.Value {
	newStruct := reflect.New(targetStruct.Type()).Elem()

	// Copy fields from elem to newElem
	for k := 0; k < value.NumField(); k++ {
		currentField := newStruct.Field(k)

		if currentField.Kind() == reflect.Array || currentField.Kind() == reflect.Slice {
			currentField.Set(cp.setArray(currentField, value.Field(k)))
			continue
		}

		if currentField.Kind() == reflect.Struct {
			currentField.Set(cp.setStruct(currentField, value.Field(k)))
			continue
		}

		currentField.Set(value.Field(k))
	}

	return newStruct
}

func (cp *configParser[T]) setArray(targetArray, values reflect.Value) reflect.Value {

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

			// Copy fields from elem to newElem
			for k := 0; k < existingElement.NumField(); k++ {
				currentField := newElem.Field(k)

				if currentField.Kind() == reflect.Array || currentField.Kind() == reflect.Slice {
					currentField.Set(cp.setArray(currentField, existingElement.Field(k)))
					continue
				}

				if currentField.Kind() == reflect.Struct {
					currentField.Set(cp.setStruct(currentField, existingElement.Field(k)))
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

func (cp *configParser[T]) setField(v interface{}, fieldPath []string, value reflect.Value) {
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
					f.Set(cp.setArray(f, value))
				}
			} else {

				if !f.IsValid() {
					logger.Warn("Field not valid", "field", fieldPath)
				}

				if !f.CanAddr() {
					logger.Warn("Cant address Field", "field", fieldPath)
				}
			}
		} else {
			r = r.FieldByName(part)
		}
	}
}

func (cp *configParser[T]) getAllTagNames(tag reflect.StructTag) (result []string) {

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

// LoadConfigBytes loads a configuration from yaml, json or toml bytes and returns the populated structure
func LoadConfigBytes[T any](data []byte, strict bool, configType ConfigType) (result T, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("LoadConfigBytes(...) only supports configs of Struct type")
	}

	result, _, err = Config[T](FromConfigBytes(data, strict, configType))

	return
}

// LoadConfigFileAuto loads a yaml, json or toml file from path and returns the populated structure
// sets automatic file type determination
func LoadConfigFileAuto[T any](path string, strict bool) (result T, err error) {
	return LoadConfigFile[T](path, strict, Auto)
}

// LoadConfigFile loads a yaml, json or toml file from path and returns the populated structure
func LoadConfigFile[T any](path string, strict bool, configType ConfigType) (result T, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("LoadConfigFile(...) only supports configs of Struct type")
	}

	result, _, err = Config[T](FromConfigFile(path, strict, configType))

	return
}

func newConfigLoader[T any](o *options) *configParser[T] {
	return &configParser[T]{
		o: o,
		supportedTags: []string{
			"json",
			"yaml",
			"toml",
		},
	}
}

func (cp *configParser[T]) apply(result *T) (err error) {
	if cp.o.config.dataMethod == nil {
		panic("No data method available for getting config data, this is a mistake")
	}

	clone, err := cp.cloneWithNewTags(result)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("constructed value (with auto added tags): %#v", clone))

	configData, configType, err := cp.o.config.dataMethod()
	if err != nil {
		return err
	}

	type configDecoder interface {
		Decode(v any) (err error)
	}

	var decoder configDecoder
	switch configType {
	case Json:
		jsDec := json.NewDecoder(configData)
		if cp.o.config.strictParsing {
			jsDec.DisallowUnknownFields()
		}
		decoder = jsDec
	case Yaml:
		ymDec := yaml.NewDecoder(configData)
		ymDec.KnownFields(cp.o.config.strictParsing)
		decoder = ymDec
	case Toml:
		tmlDec := toml.NewDecoder(configData)
		if cp.o.config.strictParsing {
			tmlDec = tmlDec.DisallowUnknownFields()
		}
		decoder = tmlDec
	}

	err = decoder.Decode(clone)
	if err != nil {
		return fmt.Errorf("failed to decode config: %s", err)
	}

	fields := getFields(false, clone)

	for _, value := range fields {
		logger.Info("setting field of config file", "path", strings.Join(value.path, "."), "value", value.value.String(), "tag", value.tag)

		cp.setField(result, value.path, value.value)
	}

	return nil
}

// CloneWithNewTags creates a new struct with modified tags, leaves it blank
func (cp *configParser[T]) cloneWithNewTags(v interface{}) (interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input config was not struct")
	}

	// Create new struct type with modified tags
	newType := cp.createModifiedType(val.Type())

	// Create new struct instance
	newValue := reflect.New(newType)

	return newValue.Interface(), nil
}

func (cp *configParser[T]) createModifiedArray(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Array:
		// Modify the element type of the array, handling nested structs and arrays recursively
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			elemType = cp.createModifiedType(elemType) // Modify nested structs
		} else if elemType.Kind() == reflect.Array || elemType.Kind() == reflect.Slice {
			elemType = cp.createModifiedArray(elemType) // Handle nested arrays or slices
		}
		// Return a new array type with the modified element type
		return reflect.ArrayOf(t.Len(), elemType)

	case reflect.Slice:
		// Modify the element type of the slice
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			elemType = cp.createModifiedType(elemType) // Modify nested structs
		} else if elemType.Kind() == reflect.Array || elemType.Kind() == reflect.Slice {
			elemType = cp.createModifiedArray(elemType) // Handle nested arrays or slices
		}
		// Return a new slice type with the modified element type
		return reflect.SliceOf(elemType)

	default:
		// If not an array or slice, return the type unchanged
		return t
	}
}

func (cp *configParser[T]) createModifiedType(t reflect.Type) reflect.Type {
	fields := make([]reflect.StructField, t.NumField())

	confyTagsOnThisLevel := map[string]bool{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		newField := reflect.StructField{
			Name:      field.Name,
			Type:      field.Type,
			Tag:       field.Tag,
			Anonymous: field.Anonymous,
			PkgPath:   field.PkgPath,
		}

		logger.Info("cloning struct fields", "struct", t.Name(), "field", field.Name, "type", field.Type.Kind())

		// Handle nested structs
		if field.Type.Kind() == reflect.Struct {
			newField.Type = cp.createModifiedType(field.Type)
		} else if field.Type.Kind() == reflect.Array || field.Type.Kind() == reflect.Slice {
			newField.Type = cp.createModifiedArray(field.Type)
		}

		existingTagNames := cp.getAllTagNames(field.Tag)
		confyTagNames := map[string]string{}

		var fieldMarshallingName string
		confyInstruction, ok := field.Tag.Lookup(confyTag)
		if ok {
			parts := strings.Split(confyInstruction, ";")
			if len(parts) > 0 {
				fieldMarshallingName = parts[0]
			}

			logger.Info("field had 'confy:' tag", "struct", t.Name(), "field", field.Name, "tag_value", confyInstruction)

			if confyTagsOnThisLevel[fieldMarshallingName] {
				panic(fmt.Sprintf("duplicate confy:\"%s\" found on %s (type %s)", fieldMarshallingName, field.Name, field.Type))
			}
			confyTagsOnThisLevel[fieldMarshallingName] = true
		} else {
			logger.Info("field had NO 'confy:' tag", "struct", t.Name(), "field", field.Name, "all_tags", field.Tag)
		}

		if fieldMarshallingName != "" {
			for _, supportedTag := range cp.supportedTags {
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
				logger.Warn("could not preserve existing tag", "tag_name", tagName, "all_tags", field.Tag)
				continue
			}

			alreadySetTags[tagName] = true

			tagsToSet = append(tagsToSet, fmt.Sprintf("%s:\"%s\"", tagName, value))

		}

		for confyTagName, confyTagValue := range confyTagNames {
			if alreadySetTags[confyTagName] {
				logger.Warn("not adding auto generated tag as already exists", "tag_name", confyTagName, "existing_value", field.Tag.Get(confyTagName))
				continue
			}
			tagsToSet = append(tagsToSet, fmt.Sprintf("%s:\"%s\"", confyTagName, confyTagValue))
		}

		newField.Tag = reflect.StructTag(strings.Join(tagsToSet, " "))

		fields[i] = newField
	}

	return reflect.StructOf(fields)
}
