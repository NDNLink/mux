package mux

import (
	"reflect"
	"testing"

	"github.com/NDNLink/ndn"
)

func TestPublisher(t *testing.T) {
	c := ndn.NewCache(1)
	p := NewPublisher(c)
	p.Use(fakeMiddleware)
	p.Publish(fakeData(), fakeMiddleware)

	want := fakeData()
	got := c.Get(&ndn.Interest{
		Name: want.Name,
	})
	if got != nil {
		// reset signature for deep equal
		got.SignatureInfo = ndn.SignatureInfo{}
		got.SignatureValue = nil
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expect %v, got %v", want, got)
	}
}
