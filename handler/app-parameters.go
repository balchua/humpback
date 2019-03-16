package handler

//AppParameters struct,  so we can easily pass it to other handlers
type AppParameters struct {
	Application string
	Command     string
	Namespace   string
	KubeConfig  string
	ConfigPath  string
}
