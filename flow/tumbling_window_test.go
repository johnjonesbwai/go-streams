package flow_test

import (
	"fmt"
	"testing"
	"time"

	ext "github.com/johnjonesbwai/go-streams/extension"
	"github.com/johnjonesbwai/go-streams/flow"
	"github.com/johnjonesbwai/go-streams/internal/assert"
)

func TestTumblingWindow(t *testing.T) {
	in := make(chan any)
	out := make(chan any)

	source := ext.NewChanSource(in)
	tumblingWindow := flow.NewTumblingWindow[string](50 * time.Millisecond)
	sink := ext.NewChanSink(out)
	assert.NotEqual(t, tumblingWindow.Out(), nil)

	go func() {
		inputValues := []string{"a", "b", "c", "d", "e", "f", "g"}
		for _, v := range inputValues {
			ingestDeferred(v, in, 15*time.Millisecond)
		}
		closeDeferred(in, 160*time.Millisecond)
	}()

	go func() {
		source.
			Via(tumblingWindow).
			To(sink)
	}()

	outputValues := readSlice[[]string](sink.Out)
	fmt.Println(outputValues)

	assert.Equal(t, 3, len(outputValues)) // [[a b c] [d e f] [g]]

	assert.Equal(t, []string{"a", "b", "c"}, outputValues[0])
	assert.Equal(t, []string{"d", "e", "f"}, outputValues[1])
	assert.Equal(t, []string{"g"}, outputValues[2])
}

func TestTumblingWindow_Ptr(t *testing.T) {
	in := make(chan any)
	out := make(chan any)

	source := ext.NewChanSource(in)
	tumblingWindow := flow.NewTumblingWindow[*string](50 * time.Millisecond)
	sink := ext.NewChanSink(out)
	assert.NotEqual(t, tumblingWindow.Out(), nil)

	go func() {
		inputValues := ptrSlice([]string{"a", "b", "c", "d", "e", "f", "g"})
		for _, v := range inputValues {
			ingestDeferred(v, in, 15*time.Millisecond)
		}
		closeDeferred(in, 160*time.Millisecond)
	}()

	go func() {
		source.
			Via(tumblingWindow).
			To(sink)
	}()

	outputValues := readSlice[[]*string](sink.Out)
	fmt.Println(outputValues)

	assert.Equal(t, 3, len(outputValues)) // [[a b c] [d e f] [g]]

	assert.Equal(t, ptrSlice([]string{"a", "b", "c"}), outputValues[0])
	assert.Equal(t, ptrSlice([]string{"d", "e", "f"}), outputValues[1])
	assert.Equal(t, ptrSlice([]string{"g"}), outputValues[2])
}
