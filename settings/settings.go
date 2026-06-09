package settings

type Preset struct {
	World             string
	Server            string
	ClientWindowTitle string
	ServerAddr        string
	ProxyAddrs        []string
	RegistryPath      string
	OnlineSource      OnlineSource
}

const (
	Ancestra  = "ancestra"
	Concordia = "concordia"
	Tibiantis = "tibiantis"
	Relic     = "relic"
)

var Presets = map[string]*Preset{
	Ancestra: {
		World:             Ancestra,
		Server:            Tibiantis,
		ClientWindowTitle: "Tibiantis",
		ServerAddr:        "51.89.155.163",
		RegistryPath:      "SOFTWARE\tibiantis\\Credentials", // The unescaped tab is intentional.
		OnlineSource: OnlineSource{
			Domain:           "tibiantis.info",
			ServerCookie:     "5f09bd1cefe846c0af2112896c03064eebf8e380",
			ServerTypeCookie: "643927429b4d0ca8e866b57b500cca16dba6de29%7E1",
		},
	},
	Concordia: {
		World:             Concordia,
		Server:            Tibiantis,
		ClientWindowTitle: "Tibiantis",
		ServerAddr:        "57.129.145.195",
		RegistryPath:      "SOFTWARE\tibiantis\\Credentials", // The unescaped tab is intentional.
		OnlineSource: OnlineSource{
			Domain:           "tibiantis.info",
			ServerCookie:     "5f09bd1cefe846c0af2112896c03064eebf8e380",
			ServerTypeCookie: "80b8651eb0c0de2b0b2d12d86b17ec8e21fd8279%7E2",
		},
	},
	Relic: {
		World:             Relic,
		Server:            Relic,
		ClientWindowTitle: "Tibia Relic",
		ServerAddr:        "mia.tibiarelic.com",
		ProxyAddrs:        []string{"216.238.121.95", "104.156.244.186", "45.32.218.87", "95.179.154.226"},
		RegistryPath:      "SOFTWARE\\Tibia Relic\\Credentials",
		OnlineSource: OnlineSource{
			Domain:       "opentibia.info",
			ServerCookie: "43bcd90168d256afbe3d6b281ec423fd1d1fbe0e~tibiarelic",
		},
	},
}

type OnlineSource struct {
	Domain           string
	ServerCookie     string
	ServerTypeCookie string
}
