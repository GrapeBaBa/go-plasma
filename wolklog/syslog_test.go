package wolklog

import (
	"testing"
)

func TestNegativeBackendPOA(t *testing.T) {
	err := SysLog("teststring", CLOUD)
	if err != nil {
		t.Fatalf("Error: %+v", err)
	}
}
