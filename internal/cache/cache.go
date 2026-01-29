package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/o0n1x/mass-translate-package/format"
	"github.com/o0n1x/mass-translate-package/provider"
	"github.com/redis/go-redis/v9"
)

// handles anything related to redis caching

const translationTTL = time.Hour * 2

func SetCache(ctx context.Context, Redis *redis.Client, clienttype provider.Provider, req provider.Request, translation provider.Response) error {

	data, err := json.Marshal(translation)
	if err != nil {
		return err
	}

	status := Redis.Set(ctx, getCacheKey(clienttype, req), data, translationTTL)
	if status.Err() != nil {
		return status.Err()
	}
	return nil
}

func getCacheKey(clienttype provider.Provider, req provider.Request) string {
	var reqHash string
	if req.ReqType == format.Text {
		h := sha256.Sum256([]byte(strings.Join(req.Text, "|")))
		reqHash = hex.EncodeToString(h[:])

	} else {
		h := sha256.Sum256(req.Binary)
		reqHash = hex.EncodeToString(h[:])
	}

	return fmt.Sprintf("translate:%s:%s:%s:%s", clienttype, req.From, req.To, reqHash)
}

func GetCache(ctx context.Context, Redis *redis.Client, clienttype provider.Provider, req provider.Request) (provider.Response, bool, error) {
	result, err := Redis.Get(ctx, getCacheKey(clienttype, req)).Result()
	if errors.Is(err, redis.Nil) {
		return provider.Response{}, false, nil
	}
	if err != nil {
		return provider.Response{}, false, err
	}
	var params provider.Response
	err = json.Unmarshal([]byte(result), &params)
	if err != nil {
		return provider.Response{}, false, err
	}
	return params, true, nil
}
