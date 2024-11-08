package confy

import (
	"log"
	"reflect"
	"testing"
)

type innerStruct struct {
	Mff  string
	Oorg int
}

type testStruct struct {
	Thonku innerStruct `confy:"thonku_complex"`
	Thing  string      `confy:"thing"`
	I      int         `confy:"i_int"`
	B      bool        `confy:"b_bool"`
	Things []string    `confy:"things_array"`

	Ahhh []innerStruct `confy:"array_complex"`
}

var dummy = testStruct{
	Thonku: innerStruct{
		Mff:  "inner_string",
		Oorg: 3,
	},
	Thing:  "example_string",
	I:      42,
	B:      true,
	Things: []string{"string1", "string2", "string3"},

	Ahhh: []innerStruct{
		{
			Mff: "first_inner",
		},
		{
			Mff: "second_inner",
		},
	},
}

func TestParser(t *testing.T) {

	config, err := LoadConfigAuto[testStruct]("testdata/test.yaml", false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(config, dummy) {
		log.Printf("%+v", config)
		t.Fatal()
	}

}
