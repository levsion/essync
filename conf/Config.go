package conf

//levsion需要配置
type EsDoc struct {
	Parameter     string `json:"parameter"`
	Result        string `json:"result"`
	AppId         string `json:"appId"`
	AppName       string `json:"appName"`
	CallDate      int64  `json:"callDate"`
	InterfaceName string `json:"interfaceName"`
	InterfaceUrl  string `json:"interfaceUrl"`
	RequestMethod string `json:"requestMethod"`
	Status        int    `json:"status"`
}
