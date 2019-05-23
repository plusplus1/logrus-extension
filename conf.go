package logrus_extension

import (
	"io/ioutil"
	"strings"
)

import (
	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
)

type lvlConf struct {
	Directory string `yaml:"directory"`
	Filename  string `yaml:"filename"`
	Level     string `yaml:"level"`
	Rotate    string `yaml:"rotate"`
}

func newLvlConf(yamlFile string) (conf lvlConf, err error) {
	var bytes []byte
	if bytes, err = ioutil.ReadFile(yamlFile); err == nil {
		conf = lvlConf{Level: logrus.InfoLevel.String(), Rotate: "D"}
		if err = yaml.Unmarshal(bytes, &conf); err == nil {
			conf.Level = strings.ToLower(conf.Level)
			conf.Rotate = strings.ToUpper(conf.Rotate)
			return
		}
	}
	return
}
