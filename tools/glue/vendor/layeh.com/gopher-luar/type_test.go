package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_type_slice(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type ints []int

	L.SetGlobal("newInts", NewType(L, ints{}))

	testReturn(t, L, `ints = newInts(1); return #ints, ints:capacity()`, "1", "1")
	testReturn(t, L, `ints = newInts(0, 10); return #ints, ints:capacity()`, "0", "10")
}

func Test_type(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	tim := &StructTestPerson{
		Name: "Tim",
	}

	L.SetGlobal("user1", New(L, tim))
	L.SetGlobal("Person", NewType(L, StructTestPerson{}))
	L.SetGlobal("People", NewType(L, map[string]*StructTestPerson{}))

	testReturn(t, L, `user2 = Person(); user2.Name = "John"; user2.Friend = user1`)
	testReturn(t, L, `return user2.Name`, "John")
	testReturn(t, L, `return user2.Friend.Name`, "Tim")
	testReturn(t, L, `everyone = People(); everyone["tim"] = user1; everyone["john"] = user2`)

	everyone := L.GetGlobal("everyone").(*lua.LUserData).Value.(map[string]*StructTestPerson)
	if len(everyone) != 2 {
		t.Fatalf("expecting len(everyone) = 2, got %d", len(everyone))
	}
}
