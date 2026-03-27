package value

import (
	"fmt"
	"time"

	"uasl-reservation/internal/pkg/constant"
)

type Timestamp time.Time

func (t *Timestamp) ToString() (res string) {
	fmt.Println("ToString t", t)
	if t.IsEmpty() {
		return ""
	}
	return time.Time(*t).Format(constant.TIME_MICRO_FORMAT)
}
func (t *Timestamp) ToTime() (res time.Time) {
	if t == nil {
		return time.Time{}
	}
	return time.Time(*t)
}
func (t *Timestamp) IsEmpty() bool {
	if t == nil {
		return true
	}
	return time.Time(*t).IsZero()
}
func NewTimestampFromTime(t time.Time) (res Timestamp) {
	return Timestamp(t)
}

func NewTimestampFromString(str string) (res Timestamp, err error) {
	if str == "" {
		return Timestamp(time.Time{}), nil
	}
	t, err := time.Parse(time.RFC3339, str)
	if err == nil {
		return Timestamp(t), nil
	}
	return res, err
}
