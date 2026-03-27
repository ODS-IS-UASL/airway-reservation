package value

import (
	"database/sql"

	"uasl-reservation/internal/pkg/myerror"
	"uasl-reservation/internal/pkg/util"

	uuid "github.com/satori/go.uuid"
)

type ModelID string

func (id *ModelID) ToString() (res string) {
	if id == nil {
		return ""
	}
	return string(*id)
}
func (id *ModelID) Encode() (res string) {
	if id == nil {
		return ""
	}
	strID := id.ToString()
	return util.Encode(strID)
}
func NewModelIDFromEncodedString(id string) (res ModelID, err error) {
	if id == "" {
		return NewEmptyModelID(), nil
	}
	uid, err := util.Decode(id)
	if err != nil {
		return res, err
	}
	res = ModelID(uid.String())
	return res, nil
}
func NewModelIDFromUUIDString(id string) (res ModelID, err error) {
	if id == "" {
		return res, myerror.Errorf(myerror.BadRequest, "id is empty")
	}
	_, err = uuid.FromString(id)
	if err != nil {
		return res, err
	}
	res = ModelID(id)
	return res, nil
}

func NewModelIDFromNullString(id sql.NullString) (res ModelID, err error) {
	if id.Valid {
		return NewModelIDFromUUIDString(id.String)
	}
	return NewEmptyModelID(), nil
}

func NewEmptyModelID() (res ModelID) {
	return ModelID("")
}
