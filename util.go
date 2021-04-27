package gta

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	randomIntervalFactor = 0.2
)

// randomInterval for [interval,randomIntervalFactor*interval)
func randomInterval(interval time.Duration) time.Duration {
	return interval + time.Duration(randomIntervalFactor*rand.Float64()*float64(interval))
}

func valueJSON(value interface{}) (string, error) {
	jsonString, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}

func scanJSON(v interface{}, value interface{}) error {
	sVal := ""
	switch tv := v.(type) {
	case []byte:
		sVal = string(tv)
	case string:
		sVal = tv
	default:
		return fmt.Errorf("scanJSON: converting type %T to string", v)
	}

	if err := json.Unmarshal([]byte(sVal), value); err != nil {
		return err
	}
	return nil
}

func panicHandler() {
	if r := recover(); r != nil {
		logrus.Errorf("panic: %v\n%s", r, string(debug.Stack()))
	}
}

func minInt64(i ...int64) int64 {
	min := int64(math.MaxInt64)
	for _, a := range i {
		if a < min {
			min = a
		}
	}
	return min
}
