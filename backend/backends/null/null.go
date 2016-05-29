package null

import (
	"github.com/jtblin/go-acme/backend"
	"github.com/jtblin/go-acme/types"
)

const (
	backendName = "null"
)

type null struct{}

// Name returns the display name of the backend.
func (null *null) Name() string {
	return backendName
}

// SaveAccount saves the account to null.
func (null *null) SaveAccount(account *types.Account) error {
	return nil
}

// LoadAccount loads the account from null.
func (null *null) LoadAccount(domain string) (*types.Account, error) {
	return &types.Account{}, nil
}

func newBackend() (backend.Interface, error) {
	return &null{}, nil
}

func init() {
	backend.RegisterBackend(backendName, func() (backend.Interface, error) {
		return newBackend()
	})
}
