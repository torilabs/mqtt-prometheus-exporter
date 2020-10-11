package mqtt

import "strings"

func getTopicPart(topic string, idx int) string {
	s := strings.Split(topic, "/")
	switch true {
	case idx > 0 && idx < len(s):
		return s[idx]
	case idx < 0 && idx >= -len(s):
		return s[len(s)+idx]
	default:
		return ""
	}
}
