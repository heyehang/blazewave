package bytes

import (
	"reflect"
	"testing"
)

func TestWriterBasic(t *testing.T) {
	w := NewWriterSize(8)
	if w.Len() != 0 {
		t.Error("Initial length should be 0")
	}
	if w.Size() != 8 {
		t.Error("Initial size should be 8")
	}
	b := []byte("hi")
	w.Write(b)
	if !reflect.DeepEqual(w.Buffer(), b) {
		t.Error("Buffer content mismatch after Write")
	}
	w.Reset()
	if w.Len() != 0 {
		t.Error("Length should be 0 after Reset")
	}
}

func TestWriterPeek(t *testing.T) {
	w := NewWriterSize(4)
	buf := w.Peek(2)
	if len(buf) != 2 {
		t.Error("Peek did not reserve correct length")
	}
	copy(buf, []byte{1, 2})
	if w.Len() != 2 {
		t.Error("Position not advanced after Peek")
	}
	peeked := w.Peek(0)
	if peeked != nil {
		t.Error("Peek(0) should return nil")
	}
	peeked = w.Peek(-1)
	if peeked != nil {
		t.Error("Peek(-1) should return nil")
	}
}

func TestWriterGrow(t *testing.T) {
	w := NewWriterSize(2)
	w.Write([]byte("abcde"))
	if w.Len() != 5 {
		t.Error("Length should be 5 after large write")
	}
	if string(w.Buffer()) != "abcde" {
		t.Error("Buffer content mismatch after grow")
	}
}

func TestWriterWriteEmpty(t *testing.T) {
	w := NewWriterSize(2)
	w.Write(nil)
	if w.Len() != 0 {
		t.Error("Write(nil) should not change length")
	}
}

func TestWriterParallelInstances(t *testing.T) {
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			w := NewWriterSize(4)
			w.Write([]byte("go"))
			if string(w.Buffer()) != "go" {
				t.Error("Parallel Writer instance failed")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
