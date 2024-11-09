package confy

import (
	"flag"
	"log"
	"os"
	"reflect"
	"strings"
)

func loadEnv[T any](o options, result *T) (err error) {

	CommandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	CommandLine.SetOutput(os.Stdout)
	if len(os.Args) <= 1 {
		// There were no args to parse, so the user must not be using the cli
		return nil
	}

	log.Println(o.env)

	CommandLine.Bool("confy-help", true, "Print command line flags generated by confy")
	for _, field := range getFields(false, result) {
		if field.value.CanAddr() {

			resolvedPath := []string{}
			for i := range field.path {

				currentPath := field.path[i]
				_, ft := getField(result, field.path[:i+1])
				if value, ok := ft.Tag.Lookup(confyTag); ok {
					currentPath = value
				}

				resolvedPath = append(resolvedPath, currentPath)
			}
			field.path = resolvedPath

			flagName := strings.Join(field.path, o.cli.delimiter)

			switch field.value.Kind() {
			case reflect.String:
				CommandLine.StringVar(field.value.Addr().Interface().(*string), flagName, "", "A string value, generated by confy")
			case reflect.Int:
				CommandLine.IntVar(field.value.Addr().Interface().(*int), flagName, 0, "A int value, generated by confy")
			case reflect.Int64:
				CommandLine.Int64Var(field.value.Addr().Interface().(*int64), flagName, 0, "A int64 value, generated by confy")
			case reflect.Bool:
				CommandLine.BoolVar(field.value.Addr().Interface().(*bool), flagName, false, "A bool value, generated by confy")
			case reflect.Float64:
				CommandLine.Float64Var(field.value.Addr().Interface().(*float64), flagName, 0, "A float64 value, generated by confy")
			default:
				continue
			}
		}

	}
	err = CommandLine.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	help := false
	CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == "confy-help" {
			help = true
		}
	})

	if help {
		CommandLine.PrintDefaults()
		return flag.ErrHelp
	}

	return nil
}
