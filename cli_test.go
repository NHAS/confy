package confy

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCliBasicTypes(t *testing.T) {

	os.Args = []string{"dummyprogramname", "-thing", "helloworld", "-b_bool", "-thonku_complex.Mff", "toaster"}
	dummyConfig, err := LoadCli[testStruct](CLIDelimiter)
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

	os.Args = []string{
		"dummyprogramname",
		"-marshal", "test marshalling",
		"-thonku_complex.Mff", "innername:42",
		"-my_boy", "2024-11-09T15:04:05Z", // Example for time.Time
		"-basic_array", "item1,item2,item3", // Example for BasicArray
		"-complex_array", "text1,text2,text3", // Example for ComplexArray (implementsTextUnmarshaler)
	}

	dummyConfig, err := LoadCli[testCliStruct](CLIDelimiter)
	if err != nil {
		t.Fatal(err)
	}

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

func TestCliEmptyStringSlice(t *testing.T) {

	type lotsOfArrays struct {
		BasicArrayInt []int    `confy:"aaaaa"`
		BasicArray    []string `confy:"basic_array"`
		BasicBool     []bool
	}

	os.Args = []string{
		"dummy", "-confy-help",
	}

	_, err := LoadCli[lotsOfArrays](CLIDelimiter)
	if err == nil {
		t.Fail()
	}
}

func TestCliHelperMethod(t *testing.T) {
	type Small struct {
		Thing  string
		Nested struct {
			NestedVal string
		}
	}

	os.Args = []string{
		"dummy",
	}

	expectedContents := []string{
		"Thing",
		"Nested",
		"Nested.NestedVal",
	}

	vals := GetGeneratedCliFlags[Small](CLIDelimiter)

	if !reflect.DeepEqual(expectedContents, vals) {
		t.Fatalf("expected %v got %v", expectedContents, vals)
	}

}

func TestCliTransform(t *testing.T) {

	os.Args = []string{
		"dummyprogramname",
		"-MARSHAL", "test marshalling",
		"-THONKU_COMPLEX.MFF", "innername:42",
		"-MY_BOY", "2024-11-09T15:04:05Z", // Example for time.Time
		"-BASIC_ARRAY", "item1,item2,item3", // Example for BasicArray
		"-COMPLEX_ARRAY", "text1,text2,text3", // Example for ComplexArray (implementsTextUnmarshaler)
	}

	dummyConfig, err := LoadCliWithTransform[testCliStruct](CLIDelimiter, strings.ToUpper)
	if err != nil {
		t.Fatal(err)
	}

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
