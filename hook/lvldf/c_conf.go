package lvldf

import (
	"io/ioutil"
	"log"
)

import (
	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
)

type configure struct {
	Directory string `yaml:"directory"`
	Filename  string `yaml:"filename"`
	Level     string `yaml:"level"`
	Daily     bool   `yaml:"daily"`
}

func newConfig(configYaml string) *configure {

	var err error
	var bytes []byte

	if bytes, err = ioutil.ReadFile(configYaml); err == nil {
		var cfg = configure{Level: logrus.InfoLevel.String(), Daily: true}
		if err = yaml.Unmarshal(bytes, &cfg); err == nil {
			return &cfg
		}
	}
	if err != nil {
		log.Panicf("init log config fail, configYaml=%v, error=%v",
			configYaml,
			err.Error())
	}
	return nil
}
