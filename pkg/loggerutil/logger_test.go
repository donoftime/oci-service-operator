package loggerutil

import (
	"context"
	"errors"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"sync"
	"testing"
)

func Test_ConcurrentLoggerUpdating(t *testing.T) {
	testRoutineNum := 20
	var wg sync.WaitGroup
	wg.Add(testRoutineNum)

	logger := OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")}

	for i := 0; i < testRoutineNum; i++ {
		go func(i int) {
			defer wg.Done()
			ctx := context.Background()
			fixedLogMap := make(map[string]string)
			fixedLogMap["index"] = strconv.Itoa(i)
			ctx = context.WithValue(ctx, FixedLogMapCtxKey, fixedLogMap)
			assert.NotPanics(t, func() {
				logger.InfoLogWithFixedMessage(ctx, "test concurrent info log")
				logger.ErrorLogWithFixedMessage(ctx, errors.New("test error"), "test concurrent error log")
				logger.DebugLogWithFixedMessage(ctx, "test concurrent debug log")
			})
		}(i)
	}

	wg.Wait()
}

// discardLogger returns a discard logger for use in tests.
func discardLogger() OSOKLogger {
	return OSOKLogger{Logger: logr.Discard()}
}

// ---------------------------------------------------------------------------
// Tests: DebugLog, InfoLog, ErrorLog
// ---------------------------------------------------------------------------

func TestDebugLog_NoArgs(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.DebugLog("debug message") })
}

func TestDebugLog_WithKeyValues(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.DebugLog("debug message", "key", "value") })
}

func TestDebugLog_EmptyMessage(t *testing.T) {
	l := discardLogger()
	// Empty message results in empty finalMessage — no log call.
	assert.NotPanics(t, func() { l.DebugLog("") })
}

func TestDebugLog_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	// Non-string key/value triggers error log path.
	assert.NotPanics(t, func() { l.DebugLog("msg", "key", 42) })
}

func TestInfoLog_NoArgs(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLog("info message") })
}

func TestInfoLog_WithKeyValues(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLog("info message", "key", "value") })
}

func TestInfoLog_EmptyMessage(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLog("") })
}

func TestInfoLog_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLog("msg", "k", 123) })
}

func TestErrorLog_NoArgs(t *testing.T) {
	l := discardLogger()
	err := errors.New("some error")
	assert.NotPanics(t, func() { l.ErrorLog(err, "error message") })
}

func TestErrorLog_WithKeyValues(t *testing.T) {
	l := discardLogger()
	err := errors.New("some error")
	assert.NotPanics(t, func() { l.ErrorLog(err, "error message", "key", "value") })
}

func TestErrorLog_EmptyMessage(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.ErrorLog(errors.New("err"), "") })
}

func TestErrorLog_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.ErrorLog(errors.New("err"), "msg", "k", 99) })
}

// ---------------------------------------------------------------------------
// Tests: DebugLogWithFixedMessage, InfoLogWithFixedMessage, ErrorLogWithFixedMessage
// ---------------------------------------------------------------------------

func contextWithFixedMap(kv map[string]string) context.Context {
	return context.WithValue(context.Background(), FixedLogMapCtxKey, kv)
}

func TestDebugLogWithFixedMessage_WithContext(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "abc"})
	assert.NotPanics(t, func() { l.DebugLogWithFixedMessage(ctx, "debug msg") })
}

func TestDebugLogWithFixedMessage_EmptyContext(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.DebugLogWithFixedMessage(context.Background(), "debug msg") })
}

func TestDebugLogWithFixedMessage_WithKeyValues(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "xyz"})
	assert.NotPanics(t, func() { l.DebugLogWithFixedMessage(ctx, "msg", "key", "val") })
}

func TestDebugLogWithFixedMessage_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.DebugLogWithFixedMessage(context.Background(), "msg", "k", 1) })
}

func TestInfoLogWithFixedMessage_WithContext(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "abc"})
	assert.NotPanics(t, func() { l.InfoLogWithFixedMessage(ctx, "info msg") })
}

func TestInfoLogWithFixedMessage_EmptyContext(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLogWithFixedMessage(context.Background(), "info msg") })
}

func TestInfoLogWithFixedMessage_WithKeyValues(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "xyz"})
	assert.NotPanics(t, func() { l.InfoLogWithFixedMessage(ctx, "msg", "key", "val") })
}

func TestInfoLogWithFixedMessage_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() { l.InfoLogWithFixedMessage(context.Background(), "msg", "k", 1) })
}

func TestErrorLogWithFixedMessage_WithContext(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "abc"})
	assert.NotPanics(t, func() {
		l.ErrorLogWithFixedMessage(ctx, errors.New("err"), "error msg")
	})
}

func TestErrorLogWithFixedMessage_EmptyContext(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() {
		l.ErrorLogWithFixedMessage(context.Background(), errors.New("err"), "error msg")
	})
}

func TestErrorLogWithFixedMessage_WithKeyValues(t *testing.T) {
	l := discardLogger()
	ctx := contextWithFixedMap(map[string]string{"reqid": "xyz"})
	assert.NotPanics(t, func() {
		l.ErrorLogWithFixedMessage(ctx, errors.New("err"), "msg", "key", "val")
	})
}

func TestErrorLogWithFixedMessage_InvalidKeyValueType(t *testing.T) {
	l := discardLogger()
	assert.NotPanics(t, func() {
		l.ErrorLogWithFixedMessage(context.Background(), errors.New("err"), "msg", "k", 1)
	})
}

// ---------------------------------------------------------------------------
// Tests: extractKeyValuePairs (internal helper)
// ---------------------------------------------------------------------------

func Test_extractKeyValuePairs_Empty(t *testing.T) {
	result, err := extractKeyValuePairs(nil)
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func Test_extractKeyValuePairs_SinglePair(t *testing.T) {
	result, err := extractKeyValuePairs([]interface{}{"key", "value"})
	assert.NoError(t, err)
	assert.Equal(t, "key: value", result)
}

func Test_extractKeyValuePairs_MultiplePairs(t *testing.T) {
	result, err := extractKeyValuePairs([]interface{}{"k1", "v1", "k2", "v2"})
	assert.NoError(t, err)
	assert.Contains(t, result, "k1: v1")
	assert.Contains(t, result, "k2: v2")
}

func Test_extractKeyValuePairs_NonStringKey(t *testing.T) {
	_, err := extractKeyValuePairs([]interface{}{42, "value"})
	assert.Error(t, err)
}

func Test_extractKeyValuePairs_NonStringValue(t *testing.T) {
	_, err := extractKeyValuePairs([]interface{}{"key", 42})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Tests: finalMessageBuilder (internal helper)
// ---------------------------------------------------------------------------

func Test_finalMessageBuilder_AllEmpty(t *testing.T) {
	result := finalMessageBuilder("", "", "")
	assert.Equal(t, "", result)
}

func Test_finalMessageBuilder_MessageOnly(t *testing.T) {
	result := finalMessageBuilder("hello", "", "")
	assert.Contains(t, result, "hello")
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
}

func Test_finalMessageBuilder_ExtraParamsOnly(t *testing.T) {
	result := finalMessageBuilder("", "", "key: val")
	assert.Contains(t, result, "key: val")
}

func Test_finalMessageBuilder_FixedMessageOnly(t *testing.T) {
	result := finalMessageBuilder("", "fixed", "")
	assert.Contains(t, result, "fixed")
}

func Test_finalMessageBuilder_MessageAndExtra(t *testing.T) {
	result := finalMessageBuilder("msg", "", "extra")
	assert.Contains(t, result, "msg")
	assert.Contains(t, result, "extra")
}

func Test_finalMessageBuilder_AllPresent(t *testing.T) {
	result := finalMessageBuilder("msg", "fixed", "extra")
	assert.Contains(t, result, "msg")
	assert.Contains(t, result, "fixed")
	assert.Contains(t, result, "extra")
}

// ---------------------------------------------------------------------------
// Tests: fixedMessageBuilder (internal helper)
// ---------------------------------------------------------------------------

func Test_fixedMessageBuilder_NilContext(t *testing.T) {
	result := fixedMessageBuilder(nil)
	assert.Equal(t, "", result)
}

func Test_fixedMessageBuilder_NoFixedMap(t *testing.T) {
	result := fixedMessageBuilder(context.Background())
	assert.Equal(t, "", result)
}

func Test_fixedMessageBuilder_WithFixedMap(t *testing.T) {
	ctx := contextWithFixedMap(map[string]string{"k": "v"})
	result := fixedMessageBuilder(ctx)
	assert.Contains(t, result, "k: v")
}

func Test_fixedMessageBuilder_EmptyFixedMap(t *testing.T) {
	ctx := context.WithValue(context.Background(), FixedLogMapCtxKey, map[string]string{})
	result := fixedMessageBuilder(ctx)
	assert.Equal(t, "", result)
}

func Test_fixedMessageBuilder_WrongValueType(t *testing.T) {
	// Context has the right key but wrong type — should return empty string.
	ctx := context.WithValue(context.Background(), FixedLogMapCtxKey, "not-a-map")
	result := fixedMessageBuilder(ctx)
	assert.Equal(t, "", result)
}
