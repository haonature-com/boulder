package bdns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/miekg/dns"
)

// MockDNSClient is a mock
type MockDNSClient struct {
}

// LookupTXT is a mock
func (mock *MockDNSClient) LookupTXT(_ context.Context, hostname string) ([]string, error) {
	if hostname == "_acme-challenge.servfail.com" {
		return nil, fmt.Errorf("SERVFAIL")
	}
	if hostname == "_acme-challenge.good-dns01.com" {
		// base64(sha256("LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"
		//               + "." + "9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"))
		// expected token + test account jwk thumbprint
		return []string{"LPsIwTo7o8BoG0-vjCyGQGBWSVIPxI-i_X336eUOQZo"}, nil
	}
	if hostname == "_acme-challenge.wrong-dns01.com" {
		return []string{"a"}, nil
	}
	if hostname == "_acme-challenge.wrong-many-dns01.com" {
		return []string{"a", "b", "c", "d", "e"}, nil
	}
	if hostname == "_acme-challenge.long-dns01.com" {
		return []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, nil
	}
	if hostname == "_acme-challenge.no-authority-dns01.com" {
		// base64(sha256("LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"
		//               + "." + "9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"))
		// expected token + test account jwk thumbprint
		return []string{"LPsIwTo7o8BoG0-vjCyGQGBWSVIPxI-i_X336eUOQZo"}, nil
	}
	// empty-txts.com always returns zero TXT records
	if hostname == "_acme-challenge.empty-txts.com" {
		return []string{}, nil
	}
	return []string{"hostname"}, nil
}

// MockTimeoutError returns a a net.OpError for which Timeout() returns true.
func MockTimeoutError() *net.OpError {
	return &net.OpError{
		Err: os.NewSyscallError("ugh timeout", timeoutError{}),
	}
}

type timeoutError struct{}

func (t timeoutError) Error() string {
	return "so sloooow"
}
func (t timeoutError) Timeout() bool {
	return true
}

// LookupHost is a mock
func (mock *MockDNSClient) LookupHost(_ context.Context, hostname string) ([]net.IP, error) {
	if hostname == "always.invalid" ||
		hostname == "invalid.invalid" {
		return []net.IP{}, nil
	}
	if hostname == "always.timeout" {
		return []net.IP{}, &DNSError{dns.TypeA, "always.timeout", MockTimeoutError(), -1}
	}
	if hostname == "always.error" {
		return []net.IP{}, &DNSError{dns.TypeA, "always.error", &net.OpError{
			Op:  "read",
			Net: "udp",
			Err: errors.New("some net error"),
		}, -1}
	}
	// dual-homed host with an IPv6 and an IPv4 address
	if hostname == "ipv4.and.ipv6.localhost" {
		return []net.IP{
			net.ParseIP("::1"),
			net.ParseIP("127.0.0.1"),
		}, nil
	}
	if hostname == "ipv6.localhost" {
		return []net.IP{
			net.ParseIP("::1"),
		}, nil
	}
	ip := net.ParseIP("127.0.0.1")
	return []net.IP{ip}, nil
}

// LookupCAA returns mock records for use in tests.
func (mock *MockDNSClient) LookupCAA(_ context.Context, domain string) ([]*dns.CAA, error) {
	return nil, nil
}
