package environment

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

type keyValuePair struct {
	key   string
	value string
}

func TestAddCommandWithMissingParams(t *testing.T) {
	cmdTests := []struct {
		desc    string
		flags   []keyValuePair
		wantErr string
	}{
		{"Missing env-name flag",
			[]keyValuePair{flag("pipelines-folder", "~/pipelines.yaml")},
			`required flag(s) "env-name" not set`},
	}
	for _, tt := range cmdTests {
		t.Run(tt.desc, func(rt *testing.T) {
			_, _, err := executeCommand(NewCmdAddEnv("add", "odo pipelines environment"), tt.flags...)
			if err.Error() != tt.wantErr {
				rt.Errorf("got %s, want %s", err, tt.wantErr)
			}
		})
	}
}
func executeCommand(cmd *cobra.Command, flags ...keyValuePair) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	for _, flag := range flags {
		err := cmd.Flags().Set(flag.key, flag.value)
		if err != nil {
			return nil, "", err
		}
	}
	c, err = cmd.ExecuteC()
	return c, buf.String(), err
}

func flag(k, v string) keyValuePair {
	return keyValuePair{
		key:   k,
		value: v,
	}
}
