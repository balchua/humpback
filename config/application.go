package config

//ApplicationConfiguration holder
type ApplicationConfiguration struct {
	Name      string                 `mapstructure:"name"`
	Template  string                 `mapstructure:"template"`
	Container ContainerConfiguration `mapstructure:"container"`
	UniqueId  string
}
