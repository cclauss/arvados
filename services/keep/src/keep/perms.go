/*
Permissions management on Arvados locator hashes.

The permissions structure for Arvados is as follows (from
https://arvados.org/issues/2328)

A Keep locator string has the following format:

    [hash]+[size]+A[signature]@[timestamp]

The "signature" string here is a cryptographic hash, expressed as a
string of hexadecimal digits, and timestamp is a 32-bit Unix timestamp
expressed as a hexadecimal number.  e.g.:

    acbd18db4cc2f85cedef654fccc4a4d8+3+A257f3f5f5f0a4e4626a18fc74bd42ec34dcb228a@7fffffff

The signature represents a guarantee that this locator was generated
by either Keep or the API server for use with the supplied API token.
If a request to Keep includes a locator with a valid signature and is
accompanied by the proper API token, the user has permission to
perform any action on that object (GET, PUT or DELETE).

The signature may be generated either by Keep (after the user writes a
block) or by the API server (if the user has can_read permission on
the specified object). Keep and API server share a secret that is used
to generate signatures.

To verify a permission hint, Keep generates a new hint for the
requested object (using the locator string, the timestamp, the
permission secret and the user's API token, which must appear in the
request headers) and compares it against the hint included in the
request. If the permissions do not match, or if the API token is not
present, Keep returns a 401 error.
*/

package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The PermissionSecret is the secret key used to generate SHA1
// digests for permission hints. apiserver and Keep must use the same
// key.
var PermissionSecret []byte

// makePermSignature returns a string representing the signed permission
// hint for the blob identified by blob_hash, api_token and timestamp.
func makePermSignature(blob_hash string, api_token string, timestamp string) string {
	hmac := hmac.New(sha1.New, PermissionSecret)
	hmac.Write([]byte(blob_hash))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(api_token))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(timestamp))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

// SignLocator takes a blob_locator, an api_token and a timestamp, and
// returns a signed locator string.
func SignLocator(blob_locator string, api_token string, timestamp time.Time) string {
	// Extract the hash from the blob locator, omitting any size hint that may be present.
	blob_hash := strings.Split(blob_locator, "+")[0]
	// Return the signed locator string.
	timestamp_hex := fmt.Sprintf("%08x", timestamp.Unix())
	return blob_locator +
		"+A" + makePermSignature(blob_hash, api_token, timestamp_hex) +
		"@" + timestamp_hex
}

// VerifySignature returns true if the signature on the signed_locator
// can be verified using the given api_token.
func VerifySignature(signed_locator string, api_token string) bool {
	if re, err := regexp.Compile(`^(.*)\+A(.*)@(.*)$`); err == nil {
		if matches := re.FindStringSubmatch(signed_locator); matches != nil {
			blob_locator := matches[1]
			timestamp_hex := matches[3]
			if expire_ts, err := ParseHexTimestamp(timestamp_hex); err == nil {
				// Fail signatures with expired timestamps.
				if expire_ts.Before(time.Now()) {
					return false
				}
				return signed_locator == SignLocator(blob_locator, api_token, expire_ts)
			}
		}
	}
	return false
}

func ParseHexTimestamp(timestamp_hex string) (ts time.Time, err error) {
	if ts_int, e := strconv.ParseInt(timestamp_hex, 16, 0); e == nil {
		ts = time.Unix(ts_int, 0)
	} else {
		err = e
	}
	return ts, err
}
