package confy

import (
	"os"
	"testing"
)

func TestConfigBasic(t *testing.T) {
	var d testStruct

	os.Args = []string{
		"dummyprogramname",
		"-marshal", "test marshalling",
		"-thonku_complex.Mff", "innername:42",
		"-my_boy", "2024-11-09T15:04:05Z", // Example for time.Time
		"-basic_array", "item1,item2,item3", // Example for BasicArray
		"-complex_array", "text1,text2,text3", // Example for ComplexArray (implementsTextUnmarshaler)
	}

	_, _, err := Config(d, Defaults("testdata/test.json"))
	if err != nil {
		t.Fatal(err)
	}
}
