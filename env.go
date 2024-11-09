package confy

import (
	"encoding"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type envParser[T any] struct {
	o *options
}

func newEnvLoader[T any](o *options) *envParser[T] {
	return &envParser[T]{
		o: o,
	}
}

// LoadEnv populates a configuration file T from environment variables
func LoadEnv[T any](delimiter string) (result T, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("LoadEnv(...) only supports configs of Struct type")
	}

	o := &options{
		env: struct{ delimiter string }{
			delimiter: delimiter,
		},
	}

	initLogger(o, LoggingDisabled)

	err = newEnvLoader[T](o).apply(&result)

	return
}

// GetGeneratedEnv return list of auto generated environment variable names that LoadEnv/Config will check
func GetGeneratedEnv[T any](delimiter string) []string {
	var a T
	if reflect.TypeOf(a).Kind() != reflect.Struct {
		panic("GetGeneratedEnv(...) only supports configs of Struct type")
	}

	var result []string
	for _, field := range getFields(true, &a) {
		result = append(result, strings.Join(resolvePath(&a, field.path), delimiter))
	}

	return result
}

func (ep *envParser[T]) apply(result *T) (err error) {

	for _, field := range getFields(true, result) {
		// Update GetGeneratedEnv if this changes
		envVariable := strings.Join(resolvePath(result, field.path), ep.o.env.delimiter)
		envVarValue := os.Getenv(envVariable)

		printedValue := envVarValue

		isSensitive := false
		value, ok := field.tag.Lookup(confyTag)
		if ok {
			parts := strings.Split(value, ";")
			if len(parts) > 1 {
				isSensitive = strings.TrimSpace(parts[1]) == "sensitive"
			}
		}

		if isSensitive && envVarValue != "" {
			printedValue = "**********"
		}

		ep.o.logger.Info("ENV", envVariable, printedValue)

		if envVarValue != "" {
			ep.setBasicFieldFromString(result, field.path, envVarValue)
		}
	}

	return nil
}

func (ep *envParser[T]) setBasicFieldFromString(v interface{}, fieldPath []string, value string) {
	r := reflect.ValueOf(v).Elem()

	flagName := strings.Join(resolvePath(v, fieldPath), ep.o.cli.delimiter)

outer:
	for i, part := range fieldPath {
		if i == len(fieldPath)-1 {
			f := r.FieldByName(part)
			if f.IsValid() {

				switch f.Kind() {
				case reflect.String:
					f.SetString(value)
				case reflect.Int, reflect.Int64:
					reflectedVal, err := strconv.Atoi(value)
					if err != nil {
						ep.o.logger.Error("field should be float", "err", err, "path", flagName)
						continue
					}
					f.SetInt(int64(reflectedVal))
				case reflect.Bool:
					switch value {
					case "true", "false":
						f.SetBool(value == "true")
					default:
						ep.o.logger.Error("field should be bool", "value", value, "path", flagName)
						continue
					}
				case reflect.Float64:
					reflectedVal, err := strconv.ParseFloat(value, 64)
					if err != nil {
						ep.o.logger.Error("field should be float", "err", err, "path", flagName)
						continue
					}
					f.SetFloat(reflectedVal)
				case reflect.Slice:
					sliceParts := strings.Split(value, ",")

					sliceContentType := f.Type().Elem()
					switch sliceContentType.Kind() {
					case reflect.String:
						f.Set(reflect.ValueOf(sliceParts))
					case reflect.Int, reflect.Int64:
						var resultingArray []int
						for _, p := range sliceParts {
							a, err := strconv.Atoi(p)
							if err != nil {
								ep.o.logger.Error("expected int could not parse", "err", err, "value", p, "path", flagName)

								continue outer
							}

							resultingArray = append(resultingArray, a)
						}

						f.Set(reflect.ValueOf(resultingArray))

					case reflect.Float64:
						var resultingArray []float64
						for _, p := range sliceParts {
							a, err := strconv.ParseFloat(p, 64)
							if err != nil {
								ep.o.logger.Error("expected float could not parse", "err", err, "value", p, "path", flagName)

								continue outer
							}

							resultingArray = append(resultingArray, a)
						}

						f.Set(reflect.ValueOf(resultingArray))
					case reflect.Bool:
						var resultingArray []bool
						for _, p := range sliceParts {

							switch p {
							case "true", "false":
								resultingArray = append(resultingArray, p == "true")
							default:
								ep.o.logger.Error("expected bool could not parse", "value", p, "path", flagName)
								continue outer
							}
						}

						f.Set(reflect.ValueOf(resultingArray))
					default:

						inter := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
						if !reflect.PointerTo(sliceContentType).Implements(inter) {
							ep.o.logger.Warn("type inside of complex slice did not implement encoding.TextUnmarshaler", "flag", flagName, "path", flagName)
							continue
						}

						sliceVal := reflect.MakeSlice(reflect.SliceOf(sliceContentType), 0, len(sliceParts))
						for _, p := range sliceParts {
							n := reflect.New(sliceContentType).Interface().(encoding.TextUnmarshaler)

							err := n.UnmarshalText([]byte(p))
							if err != nil {
								ep.o.logger.Error("could not unmarshal text for complex inner slice type", "err", err, "flag", flagName, "path", flagName)
								continue outer
							}

							// Append to our slice - need to get the element value, not pointer
							elemVal := reflect.ValueOf(n).Elem()
							sliceVal = reflect.Append(sliceVal, elemVal)
						}

						f.Set(sliceVal)

					}

				case reflect.Struct:

					_, ok := f.Addr().Interface().(encoding.TextUnmarshaler)
					if !ok {
						ep.o.logger.Warn("structure doesnt implement encoding.TextUnmarshaler", "flag", flagName, "path", flagName)
						continue
					}

					n := reflect.New(f.Type())

					err := n.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value))
					if err != nil {
						ep.o.logger.Error("unmarshaling struct (TextUnmarshaler) failed", "err", err, "path", flagName)
						continue
					}

					f.Set(n.Elem())

				default:
					ep.o.logger.Warn("unsupported type for env auto-addition", "type", f.Kind().String(), "path", flagName)
					continue
				}
			} else {
				ep.o.logger.Error("Field not found", "path", flagName)
			}
		} else {
			r = r.FieldByName(part)
		}
	}
}
