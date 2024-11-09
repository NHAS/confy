package confy

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCliBasicTypes(t *testing.T) {

	var dummyConfig testStruct
	os.Args = []string{"dummyprogramname", "-thing", "helloworld", "-b_bool", "-thonku_complex.Mff", "toaster"}

	err := loadCli(options{
		cli: struct{ delimiter string }{
			delimiter: ".",
		},
	}, &dummyConfig)
	if err != nil {
		t.Fatal(err)
	}

	if !dummyConfig.B {
		t.Fatal()
	}

	if dummyConfig.Thing != "helloworld" {
		t.Fatalf("%q", dummyConfig.Thing)
	}

	if dummyConfig.Thonku.Mff != "toaster" {
		t.Fatalf("%q", dummyConfig.Thing)
	}
}

type implementsTextUnmarshaler struct {
	content string
}

func (i *implementsTextUnmarshaler) UnmarshalText(data []byte) error {

	i.content = strings.TrimSpace(string(data))
	return nil
}

func (s *implementsTextUnmarshaler) MarshalText() ([]byte, error) {
	return []byte(s.content), nil
}

type testCliStruct struct {
	ImplementsMarshaller implementsTextUnmarshaler `confy:"marshal"`

	Thonku  innerStruct `confy:"thonku_complex"`
	ItsTime time.Time   `confy:"my_boy"`

	BasicArray []string `confy:"basic_array"`

	ComplexArray []implementsTextUnmarshaler `confy:"complex_array"`
}

func TestCliComplexTypes(t *testing.T) {

	var dummyConfig testCliStruct
	os.Args = []string{
		"dummyprogramname",
		"-marshal", "test marshalling",
		"-thonku_complex.Mff", "innername:42",
		"-my_boy", "2024-11-09T15:04:05Z", // Example for time.Time
		"-basic_array", "item1,item2,item3", // Example for BasicArray
		"-complex_array", "text1,text2,text3", // Example for ComplexArray (implementsTextUnmarshaler)
	}

	err := loadCli(options{
		cli: struct{ delimiter string }{
			delimiter: ".",
		},
	}, &dummyConfig)
	if err != nil {
		t.Fatal(err)
	}

	if dummyConfig.ImplementsMarshaller.content != "test marshalling" {
		t.Errorf("Expected ImplementsMarshaller content 'test marshalling', got '%s'", dummyConfig.ImplementsMarshaller.content)
	}

	if dummyConfig.Thonku.Mff != "innername:42" {
		t.Errorf("Expected Thonku Mff innername:42  got %s", dummyConfig.Thonku.Mff)
	}

	expectedTime := time.Date(2024, time.November, 9, 15, 4, 5, 0, time.UTC)
	if !dummyConfig.ItsTime.Equal(expectedTime) {
		t.Errorf("Expected ItsTime to be '%v', got '%v'", expectedTime, dummyConfig.ItsTime)
	}

	expectedBasicArray := []string{"item1", "item2", "item3"}
	if !equalStringSlices(dummyConfig.BasicArray, expectedBasicArray) {
		t.Errorf("Expected BasicArray to be '%v', got '%v'", expectedBasicArray, dummyConfig.BasicArray)
	}

	expectedComplexArray := []implementsTextUnmarshaler{
		{content: "text1"},
		{content: "text2"},
		{content: "text3"},
	}
	for i, v := range dummyConfig.ComplexArray {
		if v.content != expectedComplexArray[i].content {
			t.Errorf("Expected ComplexArray[%d] to be '%s', got '%s'", i, expectedComplexArray[i].content, v.content)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
