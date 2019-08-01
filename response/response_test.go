package response

import "testing"

func TestSuccess(t *testing.T) {
	response := Success("ok")
	if response.Message != "ok" {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}

func TestFail(t *testing.T) {
	response := Fail("ko")
	if response.Message != "ko" {
		t.Fail()
	}
	if response.Code != 1 {
		t.Fail()
	}
}
