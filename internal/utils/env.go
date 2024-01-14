package utils

import (
	"fmt"
	"os"
	"strconv"
)

type Env interface {
	uint | bool | string
}

func GetEnv[T Env](key string, defaultVal string, required bool) T {
	var retVal T

	val, ok := os.LookupEnv(key)
	if !ok {
		if required {
			panic(fmt.Sprintf("env %s is required", key))
		}

		val = defaultVal
	}

	switch ptr := any(&retVal).(type) {
	case *uint:
		parsedVal, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("error: parsing env %s=%s", key, val))
		}

		*ptr = uint(parsedVal)
	case *bool:
		parsedVal, err := strconv.ParseBool(val)
		if err != nil {
			panic(fmt.Sprintf("error: parsing env %s=%s", key, val))
		}
		*ptr = parsedVal
	case *string:
		*ptr = val
	}

	return retVal
}
