package main

import "testing"

func TestListenAddrLoopbackOnly(t *testing.T) {
	got, err := listenAddr(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:8787" {
		t.Fatalf("default: %s", got)
	}
	got, err = listenAddr([]string{"8790"})
	if err != nil || got != "127.0.0.1:8790" {
		t.Fatalf("port: %s %v", got, err)
	}
	if _, err := listenAddr([]string{"0.0.0.0:8787"}); err == nil {
		t.Fatal("must refuse 0.0.0.0")
	}
}
