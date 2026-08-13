package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Registry indexes facade-owned precompile adapters without depending on their implementations.
type Registry[T any] struct {
	entries   []T
	addresses []common.Address
	byName    map[string]T
}

// MustRegistry creates an immutable precompile registry and rejects ambiguous entries.
func MustRegistry[T any](entries []T, addressOf func(T) common.Address, nameOf func(T) string) *Registry[T] {
	registry := &Registry[T]{
		entries: append([]T(nil), entries...),
		byName:  make(map[string]T, len(entries)),
	}
	seenAddresses := make(map[common.Address]struct{}, len(entries))
	for _, entry := range entries {
		address := addressOf(entry)
		name := nameOf(entry)
		if address == (common.Address{}) {
			panic("cryptoupgrade precompile has empty address")
		}
		if name == "" {
			panic("cryptoupgrade precompile has empty name")
		}
		if _, exists := seenAddresses[address]; exists {
			panic(fmt.Sprintf("duplicate cryptoupgrade precompile address %s", address))
		}
		if _, exists := registry.byName[name]; exists {
			panic(fmt.Sprintf("duplicate cryptoupgrade precompile name %s", name))
		}
		seenAddresses[address] = struct{}{}
		registry.addresses = append(registry.addresses, address)
		registry.byName[name] = entry
	}
	return registry
}

// Entries returns a copy of registered adapters.
func (r *Registry[T]) Entries() []T {
	return append([]T(nil), r.entries...)
}

// Addresses returns a copy of registered precompile addresses.
func (r *Registry[T]) Addresses() []common.Address {
	return append([]common.Address(nil), r.addresses...)
}

// ByName returns a registered adapter by stable algorithm name.
func (r *Registry[T]) ByName(name string) (T, bool) {
	entry, ok := r.byName[name]
	return entry, ok
}
