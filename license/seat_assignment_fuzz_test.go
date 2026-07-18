package license

import (
	"encoding/json"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzBugSeatInviteAddressParserNeverAcceptsInvalidState(f *testing.F) {
	f.Add("member@example.com")
	f.Add(" Member@example.com\n")
	f.Add("member@@example.com")
	f.Fuzz(func(t *testing.T, value string) {
		address, err := ParseBugSeatInviteAddress(value)
		if err != nil {
			return
		}
		if err := address.Validate(); err != nil {
			t.Fatalf("accepted address failed Validate(): %v", err)
		}
		raw, err := json.Marshal(address)
		if err != nil {
			t.Fatalf("accepted address failed MarshalJSON(): %v", err)
		}
		var decoded BugSeatInviteAddress
		if err := json.Unmarshal(raw, &decoded); err != nil || decoded != address {
			t.Fatalf("address round trip = %v, %v", decoded, err)
		}
	})
}

func FuzzBugSeatInviteTokenParserNeverAcceptsInvalidState(f *testing.F) {
	f.Add(BugSeatInviteTokenPrefix + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	f.Add("")
	f.Add(BugSeatInviteTokenPrefix + "***")
	f.Fuzz(func(t *testing.T, value string) {
		token, err := ParseBugSeatInviteToken(value)
		if err != nil {
			return
		}
		if err := token.Validate(); err != nil {
			t.Fatalf("accepted token failed Validate(): %v", err)
		}
		if digest, err := token.Digest(); err != nil || digest.Validate() != nil {
			t.Fatalf("accepted token digest = %v, %v", digest, err)
		}
	})
}

func FuzzBugSeatMemberStrictJSONNeverPanics(f *testing.F) {
	f.Add([]byte(`{"schema":"bug-seat-member-v2026","member_id":"bug-member-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","account_subject":"account-a","invite_address":"member@example.com","status":"active","generation":1,"activated_at":10}`))
	f.Add([]byte(`{"schema":"bug-seat-member-v2026","generation":1,"generation":2}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		member, err := core.DecodeStrictJSON[BugSeatMember](data)
		if err == nil && member.Validate() != nil {
			t.Fatalf("strict decoder returned invalid member")
		}
	})
}
