package builder

import "testing"

func mkBuilder(id, hostname string) *Builder {
	return &Builder{ID: id, Hostname: hostname, Port: 22, SSHUser: DefaultSSHUser, Systems: []string{"x86_64-linux"}}
}

func TestIndex_AddSnapshot(t *testing.T) {
	idx := NewIndex()
	idx.Add(mkBuilder("a", "a.local"))

	got := idx.Snapshot()
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestIndex_AddReplacesByID(t *testing.T) {
	idx := NewIndex()
	idx.Add(mkBuilder("a", "a.local"))
	idx.Add(mkBuilder("a", "a-renamed.local"))

	got := idx.Snapshot()
	if len(got) != 1 || got[0].Hostname != "a-renamed.local" {
		t.Fatalf("expected replace by ID, got %+v", got)
	}
}

func TestIndex_Remove(t *testing.T) {
	idx := NewIndex()
	idx.Add(mkBuilder("a", "a.local"))
	idx.Remove("a")

	if got := idx.Snapshot(); len(got) != 0 {
		t.Fatalf("expected empty snapshot after remove, got %+v", got)
	}
}

func TestIndex_Remove_NonExistent(t *testing.T) {
	idx := NewIndex()
	idx.Remove("does-not-exist")
}

func TestIndex_Snapshot_SortedByHostname(t *testing.T) {
	idx := NewIndex()
	idx.Add(mkBuilder("c", "c.local"))
	idx.Add(mkBuilder("a", "a.local"))
	idx.Add(mkBuilder("b", "b.local"))

	got := idx.Snapshot()
	if len(got) != 3 || got[0].Hostname != "a.local" || got[1].Hostname != "b.local" || got[2].Hostname != "c.local" {
		t.Fatalf("expected sorted snapshot, got %+v", got)
	}
}
