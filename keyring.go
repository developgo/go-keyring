package keyring

import "errors"

// provider set in the init function by the relevant os file e.g.:
// keyring_linux.go
var provider Keyring = fallbackServiceProvider{}

var (
	// ErrNotFound is the expected error if the secret isn't found in the
	// keyring.
	ErrNotFound = errors.New("secret not found in keyring")
)

// Keyring provides a simple set/get interface for a keyring service.
type Keyring interface {
	// Set password in keyring for user.
	Set(service, user, password string) error
	// Get password from keyring given service and user name.
	Get(service, user string) (string, error)
	// Delete secret from keyring.
	Delete(service, user string) error
}

// Set password in keyring for user.
func Set(service, user, password string) error {
	return provider.Set(service, user, password)
}

// Get password from keyring given service and user name.
func Get(service, user string) (string, error) {
	return provider.Get(service, user)
}

// Delete secret from keyring.
func Delete(service, user string) error {
	return provider.Delete(service, user)
}
