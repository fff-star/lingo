package cli

import (
	"encoding/json"
	"os/exec"
)

func prettyJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
