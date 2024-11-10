package confy

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type stringSlice struct {
	target *[]string
}

func newStringSlice(target interface{}) *stringSlice {
	t, ok := target.(*[]string)
	if !ok {
		panic("could not cast something that was suppose to be a pointer to a string slice")
	}
	return &stringSlice{
		target: t,
	}
}

func (s *stringSlice) String() string {
	if s == nil || s.target == nil {
		return ""
	}

	return strings.Join(*s.target, ",")
}

func (s *stringSlice) Set(value string) error {
	if s == nil || s.target == nil {
		return errors.New("nil")
	}

	*s.target = strings.Split(value, ",")
	return nil
}

type intSlice struct {
	target *[]int
}

func newIntSlice(target interface{}) *intSlice {
	t, ok := target.(*[]int)
	if !ok {
		panic("could not cast something that was suppose to be a pointer to a int slice")
	}
	return &intSlice{
		target: t,
	}
}

func (s *intSlice) String() string {
	if s == nil || s.target == nil {
		return ""
	}

	var result []string
	for _, i := range *s.target {
		result = append(result, fmt.Sprintf("%d", i))
	}

	return strings.Join(result, ",")
}

func (s *intSlice) Set(value string) error {
	if s == nil || s.target == nil {
		return errors.New("nil")
	}

	for _, potentialInt := range strings.Split(value, ",") {
		i, err := strconv.Atoi(potentialInt)
		if err != nil {
			return err
		}
		*s.target = append(*s.target, i)
	}
	return nil
}

type floatSlice struct {
	target *[]float64
}

func newFloatSlice(target interface{}) *floatSlice {
	t, ok := target.(*[]float64)
	if !ok {
		panic("could not cast something that was suppose to be a pointer to a float slice")
	}
	return &floatSlice{
		target: t,
	}
}

func (s *floatSlice) String() string {
	if s == nil || s.target == nil {
		return ""
	}

	var result []string
	for _, i := range *s.target {
		result = append(result, fmt.Sprintf("%f", i))
	}

	return strings.Join(result, ",")
}

func (s *floatSlice) Set(value string) error {
	if s == nil || s.target == nil {
		return errors.New("nil")
	}

	for _, potentialFloat := range strings.Split(value, ",") {
		i, err := strconv.ParseFloat(potentialFloat, 64)
		if err != nil {
			return err
		}
		*s.target = append(*s.target, i)
	}
	return nil
}

type boolSlice struct {
	target *[]bool
}

func newBoolSlice(target interface{}) *boolSlice {
	t, ok := target.(*[]bool)
	if !ok {
		panic("could not cast something that was suppose to be a pointer to a bool slice")
	}
	return &boolSlice{
		target: t,
	}
}

func (s *boolSlice) String() string {
	if s == nil || s.target == nil {
		return ""
	}

	var result []string
	for _, i := range *s.target {
		result = append(result, fmt.Sprintf("%t", i))
	}

	return strings.Join(result, ",")
}

func (s *boolSlice) Set(value string) error {
	if s == nil {
		return errors.New("")
	}

	for _, potentialBool := range strings.Split(value, ",") {
		*s.target = append(*s.target, potentialBool == "true")
	}
	return nil
}

type TextSlice struct {
	concrete reflect.Type
}

func newGenericSlice(concrete reflect.Type) *TextSlice {

	return &TextSlice{
		concrete: concrete,
	}
}

func (s *TextSlice) String() string {
	return string("empty")
}

func (s *TextSlice) Set(value string) error {
	if s == nil {
		return errors.New("nil")
	}

	values := strings.Split(value, ",")
	for _, v := range values {

		n := reflect.New(s.concrete).Interface().(encoding.TextUnmarshaler)

		err := n.UnmarshalText([]byte(v))
		if err != nil {
			return fmt.Errorf("failed to unmarshal item %q: %w", v, err)
		}
	}
	return nil
}

type ciParser[T any] struct {
	o *options
}

func newCliLoader[T any](o *options) *ciParser[T] {
	return &ciParser[T]{
		o: o,
	}
}

// GetGeneratedCliFlags return list of auto generated cli flag names that LoadCli/Config will check
func GetGeneratedCliFlags[T any](delimiter string) []string {
	var a T
	if reflect.TypeOf(a).Kind() != reflect.Struct {
		panic("GetGeneratedEnv(...) only supports configs of Struct type")
	}

	o := options{}
	FromCli(delimiter)(&o)
	cp := newCliLoader[T](&o)

	var result []string
	for _, field := range getFields(true, &a) {

		cliName, ok := determineVariableName(&a, cp.o.cli.delimiter, nil, field)
		if !ok {
			continue
		}

		result = append(result, cliName)
	}

	return result
}

// GetGeneratedCliFlagsWithTransform return list of auto generated cli flag names that LoadEnv/Config will check
// it optionally also takes a transform func that you can use to change the flag name
func GetGeneratedCliFlagsWithTransform[T any](delimiter string, transformFunc Transform) []string {
	var a T
	if reflect.TypeOf(a).Kind() != reflect.Struct {
		panic("GetGeneratedCliFlagsWithTransform(...) only supports configs of Struct type")
	}

	envs := GetGeneratedCliFlags[T](delimiter)
	for i := range envs {
		if transformFunc != nil {
			envs[i] = transformFunc(envs[i])
		}
	}

	return envs
}

// LoadCli populates a configuration file T from cli arguments
func LoadCli[T any](delimiter string) (result T, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("LoadCli(...) only supports configs of Struct type")
	}

	result, _, err = Config[T](FromCli(delimiter))

	return
}

// LoadCli populates a configuration file T from cli arguments and uses the transform to change the name of the cli flag
func LoadCliWithTransform[T any](delimiter string, transform func(string) string) (result T, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("LoadCli(...) only supports configs of Struct type")
	}

	result, _, err = Config[T](FromCli(delimiter), WithCliTransform(transform))

	return
}

func (cp *ciParser[T]) usage(f *flag.FlagSet) func() {

	return func() {
		fmt.Fprintf(f.Output(), "Structure options: \n")
		f.PrintDefaults()
	}
}

func (cp *ciParser[T]) apply(result *T) (somethingSet bool, err error) {

	if len(os.Args) == 0 {
		logger.Info("no os arguments supplied, not trying to parse cli")
		return false, nil
	}

	cp.o.cli.commandLine.SetOutput(os.Stdout)
	cp.o.cli.commandLine.Usage = cp.usage(cp.o.cli.commandLine)
	if len(os.Args) <= 1 {
		logger.Info("one os arguments supplied, not trying to parse cli")
		// There were no args to parse, so the user must not be using the cli
		return false, nil
	}

	// stop go flag from overwritting literally all configuration data on default write
	dummyCopy := new(T)

	type association struct {
		v    reflect.Value
		path []string
		tag  reflect.StructTag
	}

	flagAssociation := map[string]association{}

	const sourceHelpFlag = "struct-help"
	cp.o.cli.commandLine.Bool(sourceHelpFlag, true, "Print command line flags generated by confy")
	for _, field := range getFields(true, dummyCopy) {

		willAccess := field.value.CanAddr() && field.value.CanInterface()
		logger.Info("got field from config", slog.Any(strings.Join(field.path, "."), field.value.String()), "will_continue_parsing", fmt.Sprintf("%t (addr: %t, intf: %t)", willAccess, field.value.CanAddr(), field.value.CanInterface()))

		if willAccess {

			if field.value.Kind() == reflect.Ptr {
				field.value = field.value.Elem()
			}

			flagName, ok := determineVariableName(result, cp.o.cli.delimiter, cp.o.cli.transform, field)
			if !ok {
				// logging done in determine variable
				continue
			}

			logger.Info("resolved confy path", "resolved_path", flagName, "path", strings.Join(field.path, cp.o.cli.delimiter))

			description, ok := field.tag.Lookup(confyDescriptionTag)
			if !ok {
				logger.Info("could not find 'confy_description:' tag will auto generate from type", "tags", field.tag, "path", strings.Join(field.path, cp.o.cli.delimiter))

				typeName := field.value.Kind().String()
				if field.value.Kind() == reflect.Slice {
					typeName = field.value.Type().Elem().Kind().String() + " " + typeName
				} else if field.value.Kind() == reflect.Struct {

					pkg := field.value.Type().PkgPath()
					if pkg != "" {
						pkg = filepath.Base(pkg) + "."
					}
					typeName = pkg + field.value.Type().Name() + " " + typeName
				}
				description = fmt.Sprintf("A %s value, %s (%s)", typeName, strings.Join(field.path, cp.o.cli.delimiter), flagName)
			}

			logger.Info("adding flag", "flag", "-"+flagName, "type", field.value.Kind())
			flagAssociation[flagName] = association{v: field.value, path: field.path, tag: field.tag}

			switch field.value.Kind() {
			case reflect.String:
				cp.o.cli.commandLine.StringVar(field.value.Addr().Interface().(*string), flagName, "", description)
			case reflect.Int:
				cp.o.cli.commandLine.IntVar(field.value.Addr().Interface().(*int), flagName, 0, description)
			case reflect.Int64:
				cp.o.cli.commandLine.Int64Var(field.value.Addr().Interface().(*int64), flagName, 0, description)
			case reflect.Bool:
				cp.o.cli.commandLine.BoolVar(field.value.Addr().Interface().(*bool), flagName, false, description)
			case reflect.Float64:
				cp.o.cli.commandLine.Float64Var(field.value.Addr().Interface().(*float64), flagName, 0, description)
			case reflect.Slice:
				var parser flag.Value
				sliceContentType := field.value.Type().Elem()
				switch sliceContentType.Kind() {
				case reflect.String:
					parser = newStringSlice(field.value.Addr().Interface())
				case reflect.Int, reflect.Int64:
					parser = newIntSlice(field.value.Addr().Interface())
				case reflect.Float64:
					parser = newFloatSlice(field.value.Addr().Interface())
				case reflect.Bool:
					parser = newBoolSlice(field.value.Addr().Interface())
				default:
					inter := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
					if !reflect.PointerTo(sliceContentType).Implements(inter) {
						logger.Warn("type inside of complex slice did not implement encoding.TextUnmarshaler", "flag", flagName, "path", strings.Join(field.path, cp.o.cli.delimiter))
						continue
					}
					parser = newGenericSlice(sliceContentType)
				}

				cp.o.cli.commandLine.Var(parser, flagName, description)
			case reflect.Struct:

				textUnmarshaler, ok := field.value.Addr().Interface().(encoding.TextUnmarshaler)
				if !ok {
					logger.Warn("structure doesnt implement encoding.TextUnmarshaler", "flag", flagName, "path", strings.Join(field.path, cp.o.cli.delimiter))
					continue
				}

				textMarshaler, ok := field.value.Addr().Interface().(encoding.TextMarshaler)
				if !ok {
					logger.Warn("structure doesnt implement encoding.TextMarshaler", "flag", flagName, "path", strings.Join(field.path, cp.o.cli.delimiter))
					continue
				}

				cp.o.cli.commandLine.TextVar(textUnmarshaler, flagName, textMarshaler, description)
			default:
				logger.Warn("unsupported type for cli auto-addition", "type", field.value.Kind().String(), "path", strings.Join(field.path, cp.o.cli.delimiter))
				continue
			}
		}

	}
	err = cp.o.cli.commandLine.Parse(os.Args[1:])
	if err != nil {
		return false, err
	}

	help := false
	cp.o.cli.commandLine.Visit(func(f *flag.Flag) {
		if f.Name == sourceHelpFlag {
			logger.Info("the help flag was set", "flag", sourceHelpFlag)

			help = true
			return
		}

		association, ok := flagAssociation[f.Name]
		if !ok {
			return
		}

		v, _ := getField(result, association.path)

		v.Set(association.v)
		somethingSet = true

		logger.Info("CLI FLAG", "-"+f.Name, maskSensitive(f.Value.String(), association.tag))
	})

	if help {
		cp.o.cli.commandLine.PrintDefaults()
		return somethingSet, flag.ErrHelp
	}

	return somethingSet, nil
}
