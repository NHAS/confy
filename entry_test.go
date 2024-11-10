package confy

import (
	"log/slog"
	"os"
	"testing"
)

func TestMain(m *testing.M) {

	level.Set(slog.LevelDebug)
	code := m.Run()

	os.Exit(code)
}

func TestConfigBasic(t *testing.T) {

	os.Args = []string{
		"dummyprogramname",
		"-marshal", "test marshalling",
		"-thonku_complex.Mff", "innername:42",
		"-my_boy", "2024-11-09T15:04:05Z", // Example for time.Time
		"-basic_array", "item1,item2,item3", // Example for BasicArray
		"-complex_array", "text1,text2,text3", // Example for ComplexArray (implementsTextUnmarshaler)
	}

	_, _, err := Config[testStruct](DefaultsFromPath("testdata/test.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfigurationNothingSet(t *testing.T) {
	os.Args = []string{
		"dummy",
	}

	_, _, err := Config[testStruct](Defaults("config", "config.yaml"))
	if err == nil {
		t.Fatal("should return that help was asked for")
	}

}

func TestConfigurationHelp(t *testing.T) {
	os.Args = []string{
		"dummy", "-h",
	}

	_, _, err := Config[testStruct](Defaults("config", "config.yaml"))
	if err == nil {
		t.Fatal("should return that help was asked for")
	}

	t.Fail()
}
