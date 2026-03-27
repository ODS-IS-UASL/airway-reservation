package util

import (
	"fmt"
	"math"
	"strconv"

	"uasl-reservation/internal/pkg/logger"
)

func SafeIntToInt32(val int) int32 {
	if val > math.MaxInt32 {
		logger.LogError("value exceeds int32 max, capping", "value", val)
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		logger.LogError("value below int32 min, capping", "value", val)
		return math.MinInt32
	}
	return int32(val)
}

func SafeIntToInt32WithError(val int) (int32, error) {
	if val > math.MaxInt32 || val < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d is out of int32 range", val)
	}
	return int32(val), nil
}

func SafeInt32FromPtr(p *int) int32 {
	if p == nil {
		return 0
	}
	return SafeIntToInt32(*p)
}

func ParseInt32FromInterface(value interface{}) (int32, bool) {
	switch v := value.(type) {
	case float64:

		if math.IsNaN(v) || math.IsInf(v, 0) {
			logger.LogError("provided value is not a finite number, ignoring", "value", v)
			return 0, false
		}

		if v != math.Trunc(v) {
			logger.LogError("provided value is not an integer, truncating toward zero", "value", v)
		}
		return SafeIntToInt32(int(v)), true
	case int:
		return SafeIntToInt32(v), true
	case int32:
		return v, true
	case int64:

		if v > math.MaxInt {
			return SafeIntToInt32(math.MaxInt), true
		}
		if v < math.MinInt {
			return SafeIntToInt32(math.MinInt), true
		}
		return SafeIntToInt32(int(v)), true
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {

			if parsed > math.MaxInt {
				return SafeIntToInt32(math.MaxInt), true
			}
			if parsed < math.MinInt {
				return SafeIntToInt32(math.MinInt), true
			}
			return SafeIntToInt32(int(parsed)), true
		} else {
			logger.LogError("failed to parse string as integer, ignoring", "value", v, "error", err.Error())
			return 0, false
		}
	default:
		logger.LogError("unsupported type for int32 conversion, ignoring", "type", fmt.Sprintf("%T", value))
		return 0, false
	}
}

func ParseFloat64FromInterface(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			logger.LogError("provided value is not a finite number, ignoring", "value", v)
			return 0, false
		}
		return v, true
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			logger.LogError("provided value is not a finite number, ignoring", "value", f)
			return 0, false
		}
		return f, true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			logger.LogError("failed to parse string as float64, ignoring", "value", v, "error", err.Error())
			return 0, false
		}
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			logger.LogError("provided value is not a finite number, ignoring", "value", parsed)
			return 0, false
		}
		return parsed, true
	default:
		logger.LogError("unsupported type for float64 conversion, ignoring", "type", fmt.Sprintf("%T", value))
		return 0, false
	}
}
