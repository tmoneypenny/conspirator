package bind

import (
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/tmoneypenny/conspirator/internal/pkg/util"
)

var domain = "src"

func TestUpsertRRS(t *testing.T) {
	validTestCases := []struct {
		input    *Record
		expected error
	}{
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("a-multi.%s", domain)),
				RecordType: util.StrToPtr("A"),
				TTL:        util.Uint32ToPtr(10),
				Value:      []string{"127.0.0.1", "127.0.0.2"},
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("a.%s", domain)),
				RecordType: util.StrToPtr("A"),
				TTL:        util.Uint32ToPtr(10),
				Value:      "127.0.0.1",
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("mx.%s", domain)),
				RecordType: util.StrToPtr("MX"),
				TTL:        util.Uint32ToPtr(30),
				Value:      "10 mail.src",
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("mx-multi.%s", domain)),
				RecordType: util.StrToPtr("MX"),
				TTL:        util.Uint32ToPtr(30),
				Value:      []string{"10 mail.src", "20 mail2.src"},
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("txt.%s", domain)),
				RecordType: util.StrToPtr("TXT"),
				TTL:        util.Uint32ToPtr(30),
				Value:      "you found a text string",
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("cname.%s", domain)),
				RecordType: util.StrToPtr("CNAME"),
				TTL:        util.Uint32ToPtr(30),
				Value:      "cname.target.src",
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("srv.%s", domain)),
				RecordType: util.StrToPtr("SRV"),
				TTL:        util.Uint32ToPtr(30),
				Value:      "10 90 23 srv.src",
			},
			nil,
		},
		{
			&Record{
				FQDN:       util.StrToPtr(fmt.Sprintf("srv-multi.%s", domain)),
				RecordType: util.StrToPtr("SRV"),
				TTL:        util.Uint32ToPtr(30),
				Value:      []string{"10 90 23 srv-m.src", "20 80 23 srv2-m.src"},
			},
			nil,
		},
	}

	invalidTestCases := []struct {
		input    *Record
		expected error
	}{
		{
			&Record{
				FQDN:       util.StrToPtr("test1.src"),
				RecordType: util.StrToPtr("A"),
				TTL:        util.Uint32ToPtr(30),
				Value:      "127.0.0",
			},
			fmt.Errorf("Invalid"),
		},
		{
			&Record{
				FQDN:       util.StrToPtr("test2.src"),
				RecordType: util.StrToPtr("A"),
				TTL:        util.Uint32ToPtr(30),
				Value:      []string{"127.0.0.1", "127.0.0"},
			},
			fmt.Errorf("Invalid"),
		},
		{
			&Record{
				FQDN:       util.StrToPtr("test3.src"),
				RecordType: util.StrToPtr("A"),
				TTL:        util.Uint32ToPtr(30),
				Value:      []int{127, 0, 0, 1},
			},
			fmt.Errorf("Invalid"),
		},
	}

	for _, tc := range validTestCases {
		//fmt.Println("Valid Case:", i+1, tc)
		assert.Nil(t, tc.expected, upsertRRS(tc.input))
	}

	for _, tc := range invalidTestCases {
		//fmt.Println("Invalid Case:", i)
		assert.NotNil(t, tc.expected, upsertRRS(tc.input))
	}

	if rec, found := findRecordInZone(validTestCases[0].input); found == nil {
		fmt.Println("Found Record", rec.Record.([]dns.RR)[0].String())
	}

	// quick check to see if it is in the cache
	flushCacheToDisk()
}
