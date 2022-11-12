package fat

import (
	"io"
	"os"
	"testing"
)

func getTestFile(t *testing.T) io.ReadSeeker {
	testFile := "test_data/alphabet.txt"
	f, e := os.Open(testFile)
	if e != nil {
		t.Logf("Error opening %s: %s\n", testFile, e)
		t.FailNow()
	}
	return f
}

func TestLimitedReader(t *testing.T) {
	underlying := getTestFile(t)
	_, e := LimitReadSeeker(underlying, 15, 14)
	if e == nil {
		t.Logf("Didn't get expected error with invalid base/offset.\n")
		t.FailNow()
	}
	t.Logf("Got expected error when base offset larger than limit: %s\n", e)
	_, e = LimitReadSeeker(underlying, 53, 100)
	limited, e := LimitReadSeeker(underlying, 25, 27)
	if e != nil {
		t.Logf("Failed getting limited reader: %s\n", e)
		t.FailNow()
	}
	dst := make([]byte, 10)
	amount, e := limited.Read(dst)
	if e == nil {
		t.Logf("Didn't get EOF error when reading beyond limit.\n")
		t.FailNow()
	}
	t.Logf("Expected error when reading beyond the limit: %s\n", e)
	if amount != 2 {
		t.Logf("Didn't get expected two-byte limited read. Read %d bytes.\n",
			amount)
		t.FailNow()
	}
	if string(dst[0:amount]) != "zA" {
		t.Logf("Didn't read expected contents. Expected \"zA\", got \"%s\".\n",
			string(dst[0:amount]))
		t.FailNow()
	}
}

func TestNestedLimiting(t *testing.T) {
	underlying := getTestFile(t)
	limited, e := LimitReadSeeker(underlying, 3, 30)
	if e != nil {
		t.Logf("Failed initial limiting: %s\n", e)
		t.FailNow()
	}
	_, e = LimitReadSeeker(limited, 1, 30)
	if e == nil {
		t.Logf("Didn't get expected error when attempting to extend a " +
			"nested limitedReadSeeker past the underlying limit.\n")
		t.FailNow()
	}
	t.Logf("Got expected error when attempting to extend a nested "+
		"limitedReadSeeker past its underlying limit: %s\n", e)
	limited, e = LimitReadSeeker(limited, 1, 20)
	dst := make([]byte, 3)
	_, e = limited.Read(dst)
	if e != nil {
		t.Logf("Failed reading from nested LimitedReadSeeker: %s\n", e)
		t.FailNow()
	}
	expected := "efg"
	if string(dst) != expected {
		t.Logf("Didn't read expected contents. Expected \"%s\", got \"%s\".\n",
			expected, dst)
		t.FailNow()
	}
}
