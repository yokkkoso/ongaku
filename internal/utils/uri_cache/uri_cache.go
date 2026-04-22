package uri_cache

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

var (
	uriCache *ttlcache.Cache[string, string]
)

func init() {
	uriCache = ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](10 * time.Minute),
	)

	go uriCache.Start()
}

func generateShortHash(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hash := hasher.Sum(nil)

	encoded := base64.URLEncoding.EncodeToString(hash)
	return strings.TrimRight(encoded[:12], "=")
}

func CacheURI(uri string) string {
	hash := generateShortHash(uri)
	uriCache.Set(hash, uri, ttlcache.DefaultTTL)
	return hash
}

func GetCachedURI(hash string) *ttlcache.Item[string, string] {
	return uriCache.Get(hash)
}
