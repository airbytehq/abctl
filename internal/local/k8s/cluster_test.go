package k8s

import (
	"fmt"
	"testing"
)

func TestKindK8s_Create(t *testing.T) {
	name := "cole-test"
	k, err := NewCluster(KindProvider)
	if err != nil {
		t.Fatal(err)
	}

	//t.Cleanup(func() { k.Delete(name) })

	if err := k.Create(); err != nil {
		t.Fatal(err)
	}
	fmt.Println("created")
	//if err := k.Delete(name); err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println("deleted")
}