package confy

import (
	"flag"
	"log"
	"os"
	"reflect"
	"strings"
)

func loadCli[T any](o options, result *T) (err error) {

	CommandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	CommandLine.SetOutput(os.Stdout)
	if len(os.Args) <= 1 {
		// There were no args to parse, so the user must not be using the cli
		return nil
	}

	CommandLine.Bool("confy-help", true, "Print command line flags generated by confy")

	for _, field := range getFields(result) {

		if field.value.CanAddr() {
			log.Println("setting: ", strings.Join(field.path, o.cli.delimiter), field.value.Kind())
			switch field.value.Kind() {
			case reflect.String:
				CommandLine.StringVar(field.value.Addr().Interface().(*string), strings.Join(field.path, o.cli.delimiter), "", "A string value, generated by confy")
			case reflect.Int:
				CommandLine.IntVar(field.value.Addr().Interface().(*int), strings.Join(field.path, o.cli.delimiter), 0, "A int value, generated by confy")
			case reflect.Int64:
				CommandLine.Int64Var(field.value.Addr().Interface().(*int64), strings.Join(field.path, o.cli.delimiter), 0, "A int64 value, generated by confy")
			case reflect.Bool:
				CommandLine.BoolVar(field.value.Addr().Interface().(*bool), strings.Join(field.path, o.cli.delimiter), false, "A bool value, generated by confy")
			case reflect.Float64:
				CommandLine.Float64Var(field.value.Addr().Interface().(*float64), strings.Join(field.path, o.cli.delimiter), 0, "A float64 value, generated by confy")
			default:
				continue
			}
		}

	}
	err = CommandLine.Parse(os.Args[1:])
	if err != nil {
		// Think about what to do here TODO
		log.Println("failed to parse: ", err)
		return nil
	}

	help := false
	CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == "confy-help" {
			help = true
		}
	})

	if help {
		CommandLine.PrintDefaults()
	}

	return nil
}
