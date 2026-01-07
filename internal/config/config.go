package config

import "fmt"

type key[T any] struct {
	name     string
	value    T
	parent   Key[T]
	children KeyList[T]
}

func (k *key[T]) String() string {
	return fmt.Sprintf("%s: %v", k.Path(), k.Get())
}

func (k *key[T]) Key(keys ...string) Key[T] {
	if len(keys) == 0 {
		return k
	}
	for _, c := range k.children {
		if c.Name() == keys[0] {
			if len(keys) > 1 {
				return c.Key(keys[1:]...)
			}
			return c.Key()
		}
	}
	newKey := key[T]{
		name:   keys[0],
		parent: Key[T](k),
	}
	k.children = append(k.children, &newKey)
	if len(keys) > 1 {
		return newKey.Key(keys[1:]...)
	}
	return newKey.Key()
}

func (k *key[T]) Name() string {
	return k.name
}

func (k *key[T]) Get() T {
	return k.value
}

func (k *key[T]) Set(newValue T) {
	k.value = newValue
}

func (k *key[T]) Path() string {
	if k.parent.Name() == "" {
		return k.Name()
	}
	return k.Parent().Path() + "." + k.Name()
}

func (k *key[T]) Parent() Key[T] {
	return k.parent
}

func (k *key[T]) Children() KeyList[T] {
	return k.children
}

type KeyList[T any] []Key[T]

func (k KeyList[T]) Keys() []string {
	keys := make([]string, len(k))
	for i, key := range k {
		keys[i] = key.Name()
	}
	return keys
}

var Config key[any]

type Key[T any] interface {
	String() string
	Key(keys ...string) Key[T]
	Name() string
	Get() T
	Set(newValue T)
	Path() string
	Parent() Key[T]
	Children() KeyList[T]
}
