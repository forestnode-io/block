package block

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type prefetchConfig struct {
	maxAge     time.Duration
	cookieName string
	userAgents []string
	path       string
	noCache    bool
}

type PrefetchOption func(*prefetchConfig)

// WithMaxAge sets the max age of the prefetch block cookie.
// Minimum is 1 second.
// Default is 1 second.
func WithMaxAge(maxAge time.Duration) PrefetchOption {
	return func(c *prefetchConfig) {
		c.maxAge = maxAge
	}
}

// WithCookieName sets the name of the prefetch block cookie.
// Default is "block-prefetch".
func WithCookieName(cookieName string) PrefetchOption {
	return func(c *prefetchConfig) {
		c.cookieName = cookieName
	}
}

// WithUserAgent sets the user agents to block prefetching for.
// The strings.Contains function is used to check if the user agent
// contains the given string, so partial matches are possible.
// Default is to apply the block to all user agents.
func WithUserAgent(userAgent ...string) PrefetchOption {
	return func(c *prefetchConfig) {
		c.userAgents = append(c.userAgents, userAgent...)
	}
}

// WithPath sets the path for the prefetch block cookie.
// Default is "/".
func WithPath(path string) PrefetchOption {
	return func(c *prefetchConfig) {
		c.path = path
	}
}

// WithNoCache sets the no-cache header on the response.
// Default is false.
func WithNoCache(noCache bool) PrefetchOption {
	return func(c *prefetchConfig) {
		c.noCache = noCache
	}
}

func Prefetch(next http.Handler, o ...PrefetchOption) http.Handler {
	c := prefetchConfig{
		maxAge:     1,
		cookieName: "block-prefetch",
		userAgents: []string{},
		path:       "/",
	}
	for _, opt := range o {
		opt(&c)
	}

	writePreClient := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if c.noCache {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		w.WriteHeader(http.StatusOK)
		payload := fmt.Sprintf(preClientTmplt,
			c.cookieName,
			time.Now().UnixNano(),
			int(c.maxAge.Seconds()),
			c.path,
		)
		_, _ = w.Write([]byte(payload))
	}
	cookieIsValid := func(cookie *http.Cookie) bool {
		if cookie == nil {
			return false
		}
		cv, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			return false
		}
		return c.maxAge < time.Since(time.Unix(0, cv))
	}
	applyBlock := func(userAgent string) bool {
		if len(c.userAgents) == 0 {
			return true
		}
		for _, ua := range c.userAgents {
			if strings.Contains(userAgent, ua) {
				return true
			}
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if !applyBlock(userAgent) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(c.cookieName)
		if err != nil || !cookieIsValid(cookie) {
			writePreClient(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

const preClientTmplt = `<!DOCTYPE html>
<html>
<head>
</head>
<body>
<script>
	if (document.visibilityState === 'visible') {
		document.cookie = '%s=%d' + '; max-age=%d; path=%s';
		window.location.reload();
	}
</script>
</body>
</html>`
