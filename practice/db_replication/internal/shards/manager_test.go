package shards

import "testing"

func TestByUserID(t *testing.T) {
	m := &Manager{shards: []Shard{{ID: 0}, {ID: 1}}}
	for id, want := range map[int64]int{2: 0, 4: 0, 1: 1, 3: 1} {
		if got := m.ByUserID(id).ID; got != want {
			t.Fatalf("user %d: shard %d, want %d", id, got, want)
		}
	}
}
