package confy

import (
	"os"
	"testing"
)

func TestCliBasicTypes(t *testing.T) {

	var dummyConfig testStruct
	os.Args = []string{"dummyprogramname", "-b_bool", "-thing", "helloworld"}

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

}
