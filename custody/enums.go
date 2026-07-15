package custody

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type RetentionClass uint8

type TimestampAuthority uint8

const (
	RetentionClassTokenConditional            = "conditional"
	RetentionClassTokenPrepaid                = "prepaid"
	SessionOpenDispositionTokenUploadRequired = "upload_required"
	SessionOpenDispositionTokenReceiptReused  = "receipt_reused"
	TimestampAuthorityTokenFreeTSA            = "freetsa"
)

const (
	retentionClassInvalid RetentionClass = iota
	RetentionClassConditional
	RetentionClassPrepaid
)

const (
	timestampAuthorityInvalid TimestampAuthority = iota
	TimestampAuthorityFreeTSA
)

func timestampAuthorityNames() [TimestampAuthorityFreeTSA + 1]string {
	return [...]string{TimestampAuthorityFreeTSA: TimestampAuthorityTokenFreeTSA}
}

func (a TimestampAuthority) String() string {
	if a.IsValid() {
		return timestampAuthorityNames()[a]
	}
	return ""
}

func (a TimestampAuthority) IsValid() bool {
	return a > timestampAuthorityInvalid && int(a) < len(timestampAuthorityNames()) && timestampAuthorityNames()[a] != ""
}

func (a TimestampAuthority) Validate() error {
	if !a.IsValid() {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return nil
}

func ParseTimestampAuthority(token string) (TimestampAuthority, error) {
	for authority := TimestampAuthorityFreeTSA; int(authority) < len(timestampAuthorityNames()); authority++ {
		if timestampAuthorityNames()[authority] == token {
			return authority, nil
		}
	}
	return timestampAuthorityInvalid, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
}

func (a TimestampAuthority) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(a.String())
}

func (a *TimestampAuthority) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	parsed, err := ParseTimestampAuthority(token)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func retentionClassNames() [RetentionClassPrepaid + 1]string {
	return [...]string{
		RetentionClassConditional: RetentionClassTokenConditional,
		RetentionClassPrepaid:     RetentionClassTokenPrepaid,
	}
}

type SessionOpenDisposition uint8

const (
	sessionOpenDispositionInvalid SessionOpenDisposition = iota
	SessionOpenDispositionUploadRequired
	SessionOpenDispositionReceiptReused
)

func sessionOpenDispositionNames() [SessionOpenDispositionReceiptReused + 1]string {
	return [...]string{
		SessionOpenDispositionUploadRequired: SessionOpenDispositionTokenUploadRequired,
		SessionOpenDispositionReceiptReused:  SessionOpenDispositionTokenReceiptReused,
	}
}

func (d SessionOpenDisposition) String() string {
	if d.IsValid() {
		return sessionOpenDispositionNames()[d]
	}
	return ""
}

func (d SessionOpenDisposition) IsValid() bool {
	return d > sessionOpenDispositionInvalid && int(d) < len(sessionOpenDispositionNames()) && sessionOpenDispositionNames()[d] != ""
}

func (d SessionOpenDisposition) Validate() error {
	if !d.IsValid() {
		return fmt.Errorf(ErrFmtOpenDisposition, core.ErrCustodyContract)
	}
	return nil
}

func ParseSessionOpenDisposition(token string) (SessionOpenDisposition, error) {
	for disposition := SessionOpenDispositionUploadRequired; int(disposition) < len(sessionOpenDispositionNames()); disposition++ {
		if sessionOpenDispositionNames()[disposition] == token {
			return disposition, nil
		}
	}
	return sessionOpenDispositionInvalid, fmt.Errorf(ErrFmtOpenDisposition, core.ErrCustodyContract)
}

func (d SessionOpenDisposition) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}

func (d *SessionOpenDisposition) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtOpenDisposition, core.ErrCustodyContract)
	}
	parsed, err := ParseSessionOpenDisposition(token)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (c RetentionClass) String() string {
	if c.IsValid() {
		return retentionClassNames()[c]
	}
	return ""
}

func (c RetentionClass) IsValid() bool {
	return c > retentionClassInvalid && int(c) < len(retentionClassNames()) && retentionClassNames()[c] != ""
}

func (c RetentionClass) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	return nil
}

func (c RetentionClass) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func ParseRetentionClass(token string) (RetentionClass, error) {
	for class := RetentionClassConditional; int(class) < len(retentionClassNames()); class++ {
		if retentionClassNames()[class] == token {
			return class, nil
		}
	}
	return retentionClassInvalid, fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
}

func (c *RetentionClass) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	parsed, err := ParseRetentionClass(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
