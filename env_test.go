package confy

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEnvBasicTypes(t *testing.T) {

	os.Setenv("thing", "helloworld")
	os.Setenv("b_bool", "true")
	os.Setenv("thonku_complex_Mff", "toaster")

	dummyConfig, err := LoadEnv[testStruct](ENVDelimiter)
	if err != nil {
		t.Fatal(err)
	}

	if !dummyConfig.B {
		t.Fatalf("%+v", dummyConfig)
	}

	if dummyConfig.Thing != "helloworld" {
		t.Fatalf("%+v", dummyConfig)
	}

	if dummyConfig.Thonku.Mff != "toaster" {
		t.Fatalf("%+v", dummyConfig)
	}

}

func TestEnvComplexTypes(t *testing.T) {

	os.Setenv("marshal", "test marshalling")
	os.Setenv("thonku_complex_Mff", "innername:42")
	os.Setenv("my_boy", "2024-11-09T15:04:05Z")
	os.Setenv("basic_array", "item1,item2,item3")
	os.Setenv("complex_array", "text1,text2,text3")

	dummyConfig, err := LoadEnv[testCliStruct](ENVDelimiter)
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

func TestEnvHelperMethod(t *testing.T) {
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
		"Nested_NestedVal",
	}

	vals := GetGeneratedEnv[Small](ENVDelimiter)

	if !reflect.DeepEqual(expectedContents, vals) {
		t.Fatalf("expected %v got %v", expectedContents, vals)
	}

}

func TestEnvTransform(t *testing.T) {

	os.Setenv("MARSHAL", "test marshalling")
	os.Setenv("THONKU_COMPLEX_MFF", "innername:42")
	os.Setenv("MY_BOY", "2024-11-09T15:04:05Z")
	os.Setenv("BASIC_ARRAY", "item1,item2,item3")
	os.Setenv("COMPLEX_ARRAY", "text1,text2,text3")

	dummyConfig, err := LoadEnvWithTransform[testCliStruct](ENVDelimiter, strings.ToUpper)
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
