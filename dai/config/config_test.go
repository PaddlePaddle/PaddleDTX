package config

import (
	"encoding/json"
	"testing"
)

// TestInitConfig
func TestInitConfig(t *testing.T) {
	path := "./../conf/config.toml"

	InitConfig(path)
	executorConf, err := json.MarshalIndent(GetExecutorConf(), "", "    ")
	t.Logf("executorConf: %+v, %s", string(executorConf), err)
	t.Logf("Log: %+v", GetLogConf())
}

func TestInitCliConfig(t *testing.T) {
	paths := []string{
		"./../conf/config.toml",
		"./../conf/config-cli.toml",
	}

	for _, value := range paths {
		t.Run("testInitConfig: "+value, func(t *testing.T) {
			if err := InitCliConfig(value); err != nil {
				t.Error(err)
			}
			t.Logf("Log: %+v", GetLogConf())
			cliConf, _ := json.MarshalIndent(GetCliConf(), "", "    ")
			t.Logf("cliConf: %+v", string(cliConf))
		})
	}
}
