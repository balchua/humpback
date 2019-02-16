package config

type ContainerConfiguration struct {
	Image           string `mapstructure:"image"`
	Arguments       string
	ResourceRequest ResourceConfiguration `mapstructure:"resource-requests"`
	ResourceLimit   ResourceConfiguration `mapstructure:"resource-limits"`
	UID             string                `mapstructure:"uid"`
	GID             string                `mapstructure:"gid"`
}

type ResourceConfiguration struct {
	Memory string `mapstructure:"memory"`
	Cpu    string `mapstructure:"cpu"`
}
