package env

import (
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
)

func MustGetEnv[T any](key string) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatal().Str("key", key).Msg("environment variable not found")
	}

	var result any
	var err error

	switch any(*new(T)).(type) {
	case string:
		result = value
	case int:
		result, err = strconv.Atoi(value)
	case bool:
		result, err = strconv.ParseBool(value)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	default:
		log.Fatal().Str("key", key).Str("type", "unsupported").Msg("unsupported type for environment variable")
	}

	if err != nil {
		log.Fatal().Str("key", key).Str("value", value).Err(err).Msg("failed to parse environment variable")
	}

	return result.(T)
}
