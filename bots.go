package block

import (
	"net/http"
	"strings"
)

// botHeaders are the known User-Agent header values in use by bots / machines
var botHeaders []string = []string{
	"bot",
	"Bot",
	"facebookexternalhit",
}

func isBot(headers []string) bool {
	for _, header := range headers {
		for _, botHeader := range botHeaders {
			if strings.Contains(header, botHeader) {
				return true
			}
		}
	}

	return false
}

type botsConfig struct {
	botHandler http.Handler
}

type BotsOption func(*botsConfig)

// WithBotHandler sets the handler to be called when a bot is detected.
// Default is to return a 200 OK response.
func WithBotHandler(botHandler http.Handler) BotsOption {
	return func(c *botsConfig) {
		c.botHandler = botHandler
	}
}

func Bots(next http.Handler, o ...BotsOption) http.Handler {
	c := botsConfig{
		botHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	for _, opt := range o {
		opt(&c)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaHeaders, ok := r.Header["User-Agent"]
		if ok {
			if isBot(uaHeaders) {
				c.botHandler.ServeHTTP(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
