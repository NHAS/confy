package confy

import (
	"reflect"
	"testing"
)

type innerStructWithNest struct {
	Thing string
	I     innerStruct
}

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

	SimplePtr *string `confy:"simple_ptr"`

	VeryAhh []innerStructWithNest `confy:"array_very_complex"`
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
	VeryAhh: []innerStructWithNest{
		{
			Thing: "test",
			I: innerStruct{
				Mff:  "very_complex_inner",
				Oorg: 2,
			},
		},
	},
}

func TestAutoParser(t *testing.T) {

	s := "present"
	dummy.SimplePtr = &s

	config, err := LoadConfigFileAuto[testStruct]("testdata/test.json", false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(config, dummy) {
		t.Fatalf("%+v", config)
	}

	config, err = LoadConfigFileAuto[testStruct]("testdata/test.toml", false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(config, dummy) {
		t.Fatalf("%+v", config)
	}

	config, err = LoadConfigFileAuto[testStruct]("testdata/test.yaml", false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(config, dummy) {
		t.Fatalf("%+v", config)
	}
}

func TestDuplicateTagDetection(t *testing.T) {

	type duplicates struct {
		Test  string `confy:"test"`
		Toast string `confy:"test"`
	}

	defer func() {
		if a := recover(); a == nil {
			// didnt detect a duplicate tag
			t.Fail()
		}
	}()

	LoadConfigFileAuto[duplicates]("testdata/duplicates.json", false)

}
