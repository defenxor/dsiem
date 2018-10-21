package logger

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLog(t *testing.T) {
	if err := Setup(false); err != nil {
		t.Fatal(err)
	}
	Debug(M{})

	if err := Setup(true); err != nil {
		t.Fatal(err)
	}

	text := "test"
	i := 1
	s := "s"
	n := uint64(1)
	msgs := []M{
		{Msg: text},
		{Msg: text, DId: i},
		{Msg: text, BId: s},
		{Msg: text, CId: n},
		{Msg: text, DId: i, BId: s},
		{Msg: text, BId: s, CId: n},
		{Msg: text, DId: i, BId: s, CId: n},
	}

	for _, m := range msgs {
		var o string
		o = captureZapOutput(func() {
			Info(m)
		})
		if !strings.Contains(o, "INFO") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = captureZapOutput(func() {
			Warn(m)
		})
		if !strings.Contains(o, "WARN") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = captureZapOutput(func() {
			Debug(m)
		})
		if !strings.Contains(o, "DEBUG") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
		o = captureZapOutput(func() {
			Error(m)
		})
		if !strings.Contains(o, "ERROR") {
			t.Fatal("Cannot find string in output, o: " + o)
		}
	}
}

func captureZapOutput(funcToRun func()) string {
	var buffer bytes.Buffer

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	writer := bufio.NewWriter(&buffer)

	zlog = zap.New(
		zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.DebugLevel))
	funcToRun()
	writer.Flush()

	return buffer.String()
}
