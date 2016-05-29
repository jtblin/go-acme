/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backend

import (
	"fmt"
	"sync"

	"github.com/jtblin/go-acme/types"
)

// All registered backends.
var backendsMutex sync.Mutex
var backends = make(map[string]Factory)

// Factory is a function that returns a backend.Interface.
type Factory func() (Interface, error)

// Interface represents a backend.
type Interface interface {
	// LoadAccount loads the account from the backend store.
	LoadAccount(domain string) (*types.Account, error)
	// Name returns the display name of the backend.
	Name() string
	// SaveAccount saves the account to the backend store.
	SaveAccount(*types.Account) error
}

// RegisterBackend registers a backend.
func RegisterBackend(name string, backend Factory) {
	backendsMutex.Lock()
	defer backendsMutex.Unlock()
	if _, found := backends[name]; found {
		panic(fmt.Sprintf("Authenticator backend %q was registered twice\n", name))
	}
	backends[name] = backend
}

// GetBackend creates an instance of the named backend, or nil if
// the name is not known.  The error return is only used if the named provider
// was known but failed to initialize.
func GetBackend(name string) (Interface, error) {
	backendsMutex.Lock()
	defer backendsMutex.Unlock()
	f, found := backends[name]
	if !found {
		return nil, nil
	}
	return f()
}

// InitBackend creates an instance of the named backend.
func InitBackend(name string) (Interface, error) {
	var backend Interface
	var err error

	if name == "" {
		return nil, nil
	}

	backend, err = GetBackend(name)
	if err != nil {
		return nil, fmt.Errorf("Could not init backend %q: %v", name, err)
	}
	if backend == nil {
		return nil, fmt.Errorf("Unknown backend %q", name)
	}

	return backend, nil
}
