package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Pacemaker struct {
	Region string  `yaml:"region"`
	Interval int  `yaml:"interval"`
	Namespace string  `yaml:"namespace"`
	Metric string  `yaml:"metric"`
	DimensionValue string  `yaml:"dimension_value,omitempty"`
}

type Server struct {
	Port int  `yaml:"port"`
	Address string  `yaml:"bind"`
	Protocol string  `yaml:"proto"`
	Pass string  `yaml:"pass"`
}

type Upstream struct {
	Name string `yaml:"name"`
	Type string  `yaml:"type,omitempty"`
	Targets []string `yaml:"targets"`
	Hash string `yaml:"hash"`
}

type Settings struct {
	Upstreams	[]Upstream  `yaml:"upstreams"`
	Servers     []Server `yaml:"servers"`
	Pacemaker	Pacemaker `yaml:"pacemaker"`
}

func Load(filename string) (*Settings, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Settings{}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (m *Upstream) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Upstream

	// These are the default values for a basic metric config
	rawUpstream := plain{
		Type: "static",
	}
	if err := unmarshal(&rawUpstream); err != nil {
		return err
	}

	//TODO: Check for valid types

	*m = Upstream(rawUpstream)
	return nil
}