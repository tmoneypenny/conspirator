package bind

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
)

// rwMutex provides a lock for the zoneCache hashmap
var rwMutex = &sync.RWMutex{}

// a.domain.test = zoneRRS{RecordType: "A", TTL: 30, Record: []string{127.0.0.1, 192.168.0.1}}
// cname.domain.test = zoneRRS{RecordType: "CNAME", TTL: 30, Record: "test3.src"}
var zoneCache = make(map[string]zoneRRS)

// zoneRRS stores the RR in the cache, mapped by DomainName
type zoneRRS struct {
	RecordType string
	TTL        uint32 // See RFC 1034 & 2181
	Record     interface{}
}

// Errors
var (
	invalidRR          error = fmt.Errorf("Invalid RR Value")
	invalidType        error = fmt.Errorf("Invalid Type for RR")
	invalidDomainName  error = fmt.Errorf("Invalid DNS label")
	rrNotFound         error = fmt.Errorf("RR not found in cache")
	typeNotImplemented error = fmt.Errorf("RcodeNotImplemented")
)

// Default records
var (
	// a
	aDefaultHdr = dns.RR_Header{Rrtype: dns.TypeA,
		Class: dns.ClassINET, Ttl: 30}
	aDefaultRRS = []dns.RR{&dns.A{
		Hdr: aDefaultHdr,
		A:   net.IPv4(127, 0, 0, 1)}}
	// aaaa
	aaaaDefaultHdr = dns.RR_Header{Rrtype: dns.TypeAAAA,
		Class: dns.ClassINET, Ttl: 30}
	aaaaDefaultRRS = []dns.RR{&dns.AAAA{
		Hdr:  aaaaDefaultHdr,
		AAAA: net.IPv6loopback}}
	// cname
	cnameDefaultHdr = dns.RR_Header{Rrtype: dns.TypeCNAME,
		Class: dns.ClassINET, Ttl: 30}
	cnameDefaultRRS = &dns.CNAME{
		Hdr:    cnameDefaultHdr,
		Target: "default"}
	// txt
	txtDefaultHdr = dns.RR_Header{Rrtype: dns.TypeTXT,
		Class: dns.ClassINET, Ttl: 30}
	txtDefaultRRS = &dns.TXT{
		Hdr: txtDefaultHdr,
		Txt: []string{base64.StdEncoding.EncodeToString(seed.Bytes())}}
	// mx
	mxDefaultHdr = dns.RR_Header{Rrtype: dns.TypeMX,
		Class: dns.ClassINET, Ttl: 30}
	mxDefaultRRS = []dns.RR{&dns.MX{
		Hdr:        mxDefaultHdr,
		Preference: 10,
		Mx:         "default"}}
	// svr
	svrDefaultHdr = dns.RR_Header{Rrtype: dns.TypeSRV,
		Class: dns.ClassINET, Ttl: 30}
	svrDefaultRRS = []dns.RR{&dns.SRV{
		Hdr:      svrDefaultHdr,
		Priority: 10,
		Weight:   90,
		Port:     443,
		Target:   "default",
	}}
)

// validateIPRecord checks if the IPv4, IPv4-mapped IPv6, or IPv6 address
// in the RR is well-formed and returns the parsed IP
func validateIPRecord(record ...string) (*[]net.IP, bool) {
	var ipRecords []net.IP
	for i := range record {
		if ip := net.ParseIP(record[i]); ip == nil {
			return nil, false
		} else {
			ipRecords = append(ipRecords, ip)
		}
	}
	return &ipRecords, true
}

// RE:IsDomainName() Almost any string is a valid domain name as the DNS is 8 bit protocol.
// It checks if each label fits in 63 characters and that the entire name will fit
// into the 255 octet wire format limit.
func validateFQDN(record ...string) bool {
	for i := range record {
		if _, ok := dns.IsDomainName(record[i]); !ok {
			return false
		}
	}
	return true
}

// validateSRV parses a string for the wire format of an
// SRV record: <priority> <weight> <port> <domain>. If the
// record is valid, then it is added to a DNS SRV record,
// otherwise an empty set and bool will be set to false
func validateSRV(record ...string) ([]*dns.SRV, bool) {
	var err error
	const (
		priority = iota
		weight
		port
	)

	var srvRecordParts = map[uint16]string{
		priority: "priority",
		weight:   "weight",
		port:     "port",
	}

	var srvRecs []*dns.SRV
	for i := range record {
		srvSplit := strings.Split(record[i], " ")
		if len(srvSplit) != 4 {
			return []*dns.SRV{}, false
		}

		srvFields := make(map[string]uint64)
		for i := 0; i < len(srvSplit)-1; i++ {
			srvFields[srvRecordParts[uint16(i)]], err = strconv.ParseUint(srvSplit[i], 10, 16)
			if err != nil {
				log.Debug().Msgf("fields error: %v", err)
				return []*dns.SRV{}, false
			}
		}

		srvRecs = append(srvRecs, &dns.SRV{
			Priority: uint16(srvFields["priority"]),
			Weight:   uint16(srvFields["weight"]),
			Port:     uint16(srvFields["port"]),
		})
	}
	return srvRecs, true
}

// validateMX parses a string for the wire format of an
// MX record: <pref> <domain>. If the record is valid then
// it is added to a DNS MX record, otherwise an empty set and
// bool will be set to false
func validateMX(record ...string) ([]*dns.MX, bool) {
	var mxRecs []*dns.MX
	for i := range record {
		mxSplit := strings.Split(record[i], " ")
		if len(mxSplit) != 2 {
			return []*dns.MX{}, false
		}

		pref, err := strconv.ParseUint(mxSplit[0], 10, 16)
		if err != nil {
			return []*dns.MX{}, false
		}

		if !validateFQDN(mxSplit[1]) {
			return []*dns.MX{}, false
		}

		mxRecs = append(mxRecs, &dns.MX{
			Preference: uint16(pref),
			Mx:         dns.Fqdn(mxSplit[1])},
		)

	}
	return mxRecs, true
}

// findRecordInZone is used to find a RRS in the cache
func findRecordInZone(record *Record) (zoneRRS, error) {
	rwMutex.RLock()
	rec, ok := zoneCache[dns.Fqdn(*record.FQDN)]
	rwMutex.RUnlock()

	if !ok {
		return zoneRRS{}, rrNotFound
	}

	return rec, nil
}

// deleteRecord is a noop if the key is absent
func deleteRecord(record *Record) {
	rwMutex.Lock()
	delete(zoneCache, *record.FQDN)
	rwMutex.Unlock()
}

// upsertRRS adds the RRS to the cache for future retrieval
func upsertRRS(record *Record) error {
	if !validateFQDN(*record.FQDN) {
		return invalidDomainName
	}

	var rec []dns.RR
	var valid bool
	var header = dns.RR_Header{
		Name:   dns.Fqdn(*record.FQDN),
		Rrtype: 1, // default to A
		Class:  dns.ClassINET,
		Ttl:    *record.TTL,
	}

	switch rType := dns.StringToType[*record.RecordType]; rType {
	case dns.TypeA:
		var ipAddresses *[]net.IP
		header.Rrtype = rType

		switch record.Value.(type) {
		case string:
			ipAddresses, valid = validateIPRecord(record.Value.(string))
			if !valid {
				return invalidRR
			}
		case []string:
			ipAddresses, valid = validateIPRecord(record.Value.([]string)...)
			if !valid {
				return invalidRR
			}
		default:
			return invalidType
		}

		for _, v := range *ipAddresses {
			rec = append(rec, &dns.A{
				Hdr: header,
				A:   v,
			})
		}

	case dns.TypeAAAA:
		var ipAddresses *[]net.IP
		header.Rrtype = rType

		switch record.Value.(type) {
		case string:
			ipAddresses, valid = validateIPRecord(record.Value.(string))
			if !valid {
				return invalidRR
			}
		case []string:
			ipAddresses, valid = validateIPRecord(record.Value.([]string)...)
			if !valid {
				return invalidRR
			}
		default:
			return invalidType
		}

		for _, v := range *ipAddresses {
			rec = append(rec, &dns.AAAA{
				Hdr:  header,
				AAAA: v,
			})
		}

	case dns.TypeCNAME:
		header.Rrtype = rType
		switch record.Value.(type) {
		case string:
			if !validateFQDN(record.Value.(string)) {
				return invalidRR
			}
		default:
			return invalidRR
		}

		rec = append(rec, &dns.CNAME{
			Hdr:    header,
			Target: dns.Fqdn(record.Value.(string)),
		})

	case dns.TypeTXT:
		header.Rrtype = rType

		switch record.Value.(type) {
		case string:
			rec = append(rec, &dns.TXT{
				Hdr: header,
				Txt: []string{record.Value.(string)},
			})
		case []string:
			rec = append(rec, &dns.TXT{
				Hdr: header,
				Txt: record.Value.([]string),
			})
		default:
			return invalidRR
		}

	case dns.TypeMX:
		header.Rrtype = rType
		var mxAddresses []*dns.MX
		switch record.Value.(type) {
		case string:
			mxAddresses, valid = validateMX(record.Value.(string))
			if !valid {
				return invalidRR
			}
		case []string:
			mxAddresses, valid = validateMX(record.Value.([]string)...)
			if !valid {
				log.Debug().Msg("Invalid MX")
				return invalidRR
			}
		default:
			return invalidRR
		}

		for i := range mxAddresses {
			mxAddresses[i].Hdr = header
			rec = append(rec, mxAddresses[i])
		}
	case dns.TypeSRV:
		header.Rrtype = rType
		var srvTargets []*dns.SRV
		switch record.Value.(type) {
		case string:
			srvTargets, valid = validateSRV(record.Value.(string))
			if !valid {
				return invalidRR
			}
		case []string:
			srvTargets, valid = validateSRV(record.Value.([]string)...)
			if !valid {
				log.Debug().Msg("Invalid SRV")
				return invalidRR
			}
		default:
			return invalidRR
		}
		for i := range srvTargets {
			srvTargets[i].Hdr = header
			rec = append(rec, srvTargets[i])
		}
	default:
		return typeNotImplemented
	}

	rwMutex.Lock()
	zoneCache[dns.Fqdn(*record.FQDN)] = zoneRRS{
		RecordType: *record.RecordType,
		TTL:        *record.TTL,
		Record:     rec,
	}
	rwMutex.Unlock()

	return nil
}

// loadZoneIntoCache is called if the flag to import a zone
// file is present
func loadZoneIntoCache() {
	// ZoneParser
}

// flushCacheToDisk - blocking operation
func flushCacheToDisk() {
	rwMutex.RLock()
	for k, v := range zoneCache {
		fmt.Println(k, v)
	}
	rwMutex.RUnlock()
}
