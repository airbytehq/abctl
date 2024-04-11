package cluster

import (
	"fmt"
	"testing"
)

func TestKindProvider(t *testing.T) {
	name := "cole-test"
	k := New(Kind)
	fmt.Println("exists", k.Exists(name))
	fmt.Println("create", k.Create(name))
	fmt.Println("exists", k.Exists(name))
	fmt.Println("delete", k.Delete(name))
	fmt.Println("exists", k.Exists(name))
}
