package first_start

import (
	"testing"
	"os"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

func TestWriteToJson(t *testing.T) {
	target := map[string]string{"username": "test", "password": "test12345", "serverUrl": "http://test.test", "syncFolder": "testfolder"}
	writeToJson("test", "test12345", "http://test.test", "testfolder")

	var m map[string]string
	USER := os.Getenv("USER")
	os.MkdirAll("/home/" + USER + "/.config/NextcloudClient/config.json", 0755)

	f, err := ioutil.ReadFile("/home/" + USER + "/.config/NextcloudClient/config.json")
	if err != nil {
		t.Fatal(err)
	}

	json.Unmarshal(f, &m)

	fmt.Println(m)
	if len(m) == 0 {
		t.Fatal("Config was created empty")
	}
	for key, value := range m {
		if value != target[key] {
			t.Fatal("Target and created config maps are not equal")
		}
	}

	os.Remove("/home/" + USER + "/.config/NextcloudClient/config.json")
}