package labflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

type memBackend struct {
	m map[string][]byte
}

func (b *memBackend) Put(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	cid := "bafy" + hex.EncodeToString(sum[:16])
	if b.m == nil {
		b.m = map[string][]byte{}
	}
	b.m[cid] = append([]byte(nil), data...)
	return cid, nil
}

func (b *memBackend) Get(cid string) ([]byte, error) {
	d, ok := b.m[cid]
	if !ok {
		return nil, fmt.Errorf("missing")
	}
	return d, nil
}

func TestIPFSCreateTransitionVerify(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewStore(&memBackend{}, dir))
	res, err := svc.Create(CreateInput{Type: "water", Actor: "tech", OrgID: "lab1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.SampleCID == "" || res.RootCID == "" {
		t.Fatalf("expected CIDs, got sample=%s root=%s", res.SampleCID, res.RootCID)
	}
	_, _, _, err = svc.Transition(res.Sample.ID, TransitionInput{ToStatus: StatusAssigned, Actor: "tech"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = svc.Transition(res.Sample.ID, TransitionInput{ToStatus: StatusReleased, Actor: "tech"})
	if err == nil {
		t.Fatal("expected invalid jump to RELEASED")
	}
	for _, st := range []Status{StatusInProgress, StatusQCReview, StatusReleased} {
		if _, _, _, err := svc.Transition(res.Sample.ID, TransitionInput{ToStatus: st, Actor: "tech"}); err != nil {
			t.Fatalf("%s: %v", st, err)
		}
	}
	evs, err := svc.Events(res.Sample.ID)
	if err != nil || len(evs) < 4 {
		t.Fatalf("events=%d err=%v", len(evs), err)
	}
	view, err := svc.Verify(res.Sample.ID)
	if err != nil || !view.Verified || view.Status != StatusReleased {
		t.Fatalf("verify: %+v err=%v", view, err)
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewStore(&memBackend{}, dir))
	_, err := svc.Create(CreateInput{Type: "water", Actor: "a"})
	if err != nil {
		t.Fatal(err)
	}
	st, err := svc.Stats()
	if err != nil || st.Total != 1 || st.Received != 1 {
		t.Fatalf("stats=%+v err=%v", st, err)
	}
}
