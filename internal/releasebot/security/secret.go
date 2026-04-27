package security

import "crypto/subtle"

func HasSharedSecret(got, want string) bool {
	if got == "" || want == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
