package fs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/jtblin/go-acme/backend"
	"github.com/jtblin/go-acme/types"
)

const (
	backendName   = "fs"
	storageDirEnv = "STORAGE_DIR"
)

type storage struct {
	StorageDir  string
	storageLock sync.RWMutex
}

// Name returns the display name of the backend.
func (s *storage) Name() string {
	return backendName
}

func (s *storage) key(domain string) string {
	return path.Join(s.StorageDir, domain) + ".json"
}

// SaveAccount saves the account to the filesystem.
func (s *storage) SaveAccount(account *types.Account) error {
	s.storageLock.Lock()
	defer s.storageLock.Unlock()
	// write account to file
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.key(account.DomainsCertificate.Domain.Main), data, 0644)
}

// LoadAccount loads the account from the filesystem.
func (s *storage) LoadAccount(domain string) (*types.Account, error) {
	storageFile := s.key(domain)
	// if certificates in storage, load them
	if fileInfo, err := os.Stat(storageFile); err != nil || fileInfo.Size() == 0 {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	s.storageLock.RLock()
	defer s.storageLock.RUnlock()

	account := types.Account{
		DomainsCertificate: &types.DomainCertificate{},
	}
	file, err := ioutil.ReadFile(storageFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(file, &account); err != nil {
		return nil, fmt.Errorf("Error loading account: %v", err)
	}
	return &account, nil
}

func newBackend() (backend.Interface, error) {
	storageDir := os.Getenv(storageDirEnv)
	if storageDir != "" {
		return &storage{StorageDir: storageDir}, nil

	}
	// default to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &storage{StorageDir: cwd}, nil
}

func init() {
	backend.RegisterBackend(backendName, func() (backend.Interface, error) {
		return newBackend()
	})
}
