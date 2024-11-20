//go:build windows

package proxy

import (
	"os"
	"strings"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/sys/windows/registry"
)

type windowsProxyConfig struct {
	ProxyServer   string
	ProxyOverride string
	ProxyEnable   uint64
	AutoConfigURL string
}

func fromWindowsProxy() (values windowsProxyConfig, err error) {
	var proxySettingsPerUser uint64 = 1 // 1 is the default value to consider current user
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `Software\Policies\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err == nil {
		//We had used the below variable tempPrxUsrSettings, because the Golang method GetIntegerValue
		//sets the value to zero even it fails.
		tempPrxUsrSettings, _, err := k.GetIntegerValue("ProxySettingsPerUser")
		if err == nil {
			//consider the value of tempPrxUsrSettings if it is a success
			proxySettingsPerUser = tempPrxUsrSettings
		}
		k.Close()
	}
	var hkey registry.Key
	if proxySettingsPerUser == 0 {
		hkey = registry.LOCAL_MACHINE
	} else {
		hkey = registry.CURRENT_USER
	}
	k, err = registry.OpenKey(hkey, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	values.ProxyServer, _, err = k.GetStringValue("ProxyServer")
	if err != nil && err != registry.ErrNotExist {
		return
	}
	values.ProxyOverride, _, err = k.GetStringValue("ProxyOverride")
	if err != nil && err != registry.ErrNotExist {
		return
	}

	values.ProxyEnable, _, err = k.GetIntegerValue("ProxyEnable")
	if err != nil && err != registry.ErrNotExist {
		return
	}

	values.AutoConfigURL, _, err = k.GetStringValue("AutoConfigURL")
	if err != nil && err != registry.ErrNotExist {
		return
	}
	err = nil
	return
}

func FromEnvironment() *httpproxy.Config {
	cfg := &httpproxy.Config{
		HTTPProxy:  getEnvAny("HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"),
		HTTPSProxy: getEnvAny("HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"),
		NoProxy:    getEnvAny("NO_PROXY", "no_proxy"),
		CGI:        os.Getenv("REQUEST_METHOD") != "",
	}

	if values, err := fromWindowsProxy(); err == nil && values.ProxyEnable > 1 {
		protocol := make(map[string]string)
		for _, s := range strings.Split(values.ProxyServer, ";") {
			if s == "" {
				continue
			}
			pair := strings.SplitN(s, "=", 2)
			if len(pair) > 1 {
				protocol[pair[0]] = pair[1]
			} else {
				protocol[""] = pair[0]
			}
		}
		getProtocolAny := func(keys ...string) string {
			for _, a := range keys {
				if v, ok := protocol[a]; ok {
					return v
				}
			}
			return ""
		}
		// Overide
		if len(cfg.NoProxy) == 0 {
			cfg.NoProxy = strings.Replace(values.ProxyOverride, ";", ",", -1)
		}
		if len(cfg.HTTPProxy) == 0 {
			cfg.HTTPProxy = getProtocolAny("http", "")
		}
		if len(cfg.HTTPSProxy) == 0 {
			cfg.HTTPSProxy = getProtocolAny("https", "")
		}
	}
	return cfg
}
