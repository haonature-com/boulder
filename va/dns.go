package va

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/letsencrypt/boulder/bdns"
	"github.com/letsencrypt/boulder/core"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
)

// getAddr will query for all A/AAAA records associated with hostname and return
// the preferred address, the first net.IP in the addrs slice, and all addresses
// resolved. This is the same choice made by the Go internal resolution library
// used by net/http. If there is an error resolving the hostname, or if no
// usable IP addresses are available then a berrors.DNSError instance is
// returned with a nil net.IP slice.
func (va ValidationAuthorityImpl) getAddrs(ctx context.Context, hostname string) ([]net.IP, error) {
	addrs, err := va.dnsClient.LookupHost(ctx, hostname)
	if err != nil {
		va.logDNSError(hostname, err)
		return nil, berrors.DNSError("%v", err)
	}

	if len(addrs) == 0 {
		return nil, berrors.DNSError("No valid IP addresses found for %s", hostname)
	}
	va.log.Debugf("Resolved addresses for %s: %s", hostname, addrs)
	return addrs, nil
}

// availableAddresses takes a ValidationRecord and splits the AddressesResolved
// into a list of IPv4 and IPv6 addresses.
func availableAddresses(allAddrs []net.IP) (v4 []net.IP, v6 []net.IP) {
	for _, addr := range allAddrs {
		if addr.To4() != nil {
			v4 = append(v4, addr)
		} else {
			v6 = append(v6, addr)
		}
	}
	return
}

func (va *ValidationAuthorityImpl) validateDNS01(ctx context.Context, ident identifier.ACMEIdentifier, challenge core.Challenge) ([]core.ValidationRecord, *probs.ProblemDetails) {
	if ident.Type != identifier.DNS {
		va.log.Infof("Identifier type for DNS challenge was not DNS: %s", ident)
		return nil, probs.Malformed("Identifier type for DNS was not itself DNS")
	}

	// Compute the digest of the key authorization file
	h := sha256.New()
	h.Write([]byte(challenge.ProvidedKeyAuthorization))
	authorizedKeysDigest := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// Look for the required record in the DNS
	challengeSubdomain := fmt.Sprintf("%s.%s", core.DNSPrefix, ident.Value)
	txts, err := va.dnsClient.LookupTXT(ctx, challengeSubdomain)
	if err != nil {
		va.logDNSError(ident.Value, err)
		return nil, probs.DNS(err.Error())
	}

	// If there weren't any TXT records return a distinct error message to allow
	// troubleshooters to differentiate between no TXT records and
	// invalid/incorrect TXT records.
	if len(txts) == 0 {
		return nil, probs.Unauthorized("No TXT record found at %s", challengeSubdomain)
	}

	for _, element := range txts {
		if subtle.ConstantTimeCompare([]byte(element), []byte(authorizedKeysDigest)) == 1 {
			// Successful challenge validation
			return []core.ValidationRecord{{Hostname: ident.Value}}, nil
		}
	}

	invalidRecord := txts[0]
	if len(invalidRecord) > 100 {
		invalidRecord = invalidRecord[0:100] + "..."
	}
	var andMore string
	if len(txts) > 1 {
		andMore = fmt.Sprintf(" (and %d more)", len(txts)-1)
	}
	return nil, probs.Unauthorized("Incorrect TXT record %q%s found at %s",
		replaceInvalidUTF8([]byte(invalidRecord)), andMore, challengeSubdomain)
}

// logDNSError logs the provided error, but only if it's one of our DNS error
// types, and only if it has an underlying error. This excludes "normal" DNS
// errors like NXDOMAIN and SERVFAIL that we successfully received from our
// resolver, but includes errors in communicating with our resolver.
// We're interested in logging these separately because the problem document
// that gets sent to the user (and logged) includes only a more generic message
// like "networking error."
func (va *ValidationAuthorityImpl) logDNSError(ident string, err error) {
	if dnsErr, ok := err.(*bdns.DNSError); ok {
		underlying := dnsErr.Underlying()
		// Excluded canceled and deadline exceeded requests because those are
		// expected and are generally the "fault" of the authoritative resolver, not
		// ours.
		if underlying != nil && underlying != context.Canceled && underlying != context.DeadlineExceeded {
			va.log.Errf("For identifier %q: err=[%s], underlying=[%s]",
				ident, err, underlying)
		}
	}
}
