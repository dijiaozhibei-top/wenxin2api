package config

import "strings"

func (a Account) Identifier() string {
	if strings.TrimSpace(a.Email) != "" {
		return strings.TrimSpace(a.Email)
	}
	if mobile := NormalizeMobileForStorage(a.Mobile); mobile != "" {
		return mobile
	}
	if name := strings.TrimSpace(a.Name); name != "" {
		return name
	}
	if strings.TrimSpace(a.ChatToken) != "" {
		return "wenxin-account"
	}
	if strings.TrimSpace(a.Token) != "" {
		return "token-account"
	}
	return ""
}
