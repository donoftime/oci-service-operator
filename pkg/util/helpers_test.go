/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package util

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery/cached/memory"
	discfake "k8s.io/client-go/discovery/fake"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	k8stesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"
)

func testLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func TestDoNotRequeue(t *testing.T) {
	result, err := DoNotRequeue()
	assert.NoError(t, err)
	assert.False(t, result.Requeue)
	assert.Zero(t, result.RequeueAfter)
}

func TestRequeueWithError(t *testing.T) {
	ctx := context.Background()
	duration := 5 * time.Second
	testErr := errors.New("something went wrong")
	log := testLogger()

	result, err := RequeueWithError(ctx, testErr, duration, log)
	assert.NoError(t, err)
	assert.Equal(t, duration, result.RequeueAfter)
}

func TestRequeueWithoutError(t *testing.T) {
	ctx := context.Background()
	duration := 10 * time.Second
	log := testLogger()

	result, err := RequeueWithoutError(ctx, duration, log)
	assert.NoError(t, err)
	assert.Equal(t, duration, result.RequeueAfter)
}

func TestGetOSOKStatusCondition_Found(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{
		Conditions: []v1beta1.OSOKCondition{
			{Type: v1beta1.Active, Status: v1.ConditionTrue, Message: "ok"},
		},
	}

	cond := GetOSOKStatusCondition(status, v1beta1.Active, log)
	assert.NotNil(t, cond)
	assert.Equal(t, v1beta1.Active, cond.Type)
	assert.Equal(t, v1.ConditionTrue, cond.Status)
}

func TestGetOSOKStatusCondition_NotFound(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{}

	cond := GetOSOKStatusCondition(status, v1beta1.Active, log)
	assert.Nil(t, cond)
}

func TestGetOSOKStatusCondition_WrongType(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{
		Conditions: []v1beta1.OSOKCondition{
			{Type: v1beta1.Active, Status: v1.ConditionTrue},
		},
	}

	cond := GetOSOKStatusCondition(status, v1beta1.Failed, log)
	assert.Nil(t, cond)
}

func TestUpdateOSOKStatusCondition_NewCondition(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{}

	updated := UpdateOSOKStatusCondition(status, v1beta1.Active, v1.ConditionTrue, "Reason", "all good", log)
	assert.Len(t, updated.Conditions, 1)
	assert.Equal(t, v1beta1.Active, updated.Conditions[0].Type)
	assert.Equal(t, v1.ConditionTrue, updated.Conditions[0].Status)
	assert.Equal(t, "all good", updated.Conditions[0].Message)
}

func TestUpdateOSOKStatusCondition_StatusChange(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{
		Conditions: []v1beta1.OSOKCondition{
			{Type: v1beta1.Active, Status: v1.ConditionFalse, Message: "not ready"},
		},
	}

	updated := UpdateOSOKStatusCondition(status, v1beta1.Active, v1.ConditionTrue, "Ready", "now ready", log)
	// Original condition stays + new appended
	assert.True(t, len(updated.Conditions) >= 1)
	last := updated.Conditions[len(updated.Conditions)-1]
	assert.Equal(t, v1.ConditionTrue, last.Status)
}

func TestUpdateOSOKStatusCondition_MessageChange(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{
		Conditions: []v1beta1.OSOKCondition{
			{Type: v1beta1.Active, Status: v1.ConditionTrue, Message: "old message"},
		},
	}

	updated := UpdateOSOKStatusCondition(status, v1beta1.Active, v1.ConditionTrue, "Reason", "new message", log)
	last := updated.Conditions[len(updated.Conditions)-1]
	assert.Equal(t, "new message", last.Message)
}

func TestUpdateOSOKStatusCondition_NoChange(t *testing.T) {
	log := testLogger()
	status := v1beta1.OSOKStatus{
		Conditions: []v1beta1.OSOKCondition{
			{Type: v1beta1.Active, Status: v1.ConditionTrue, Message: "same"},
		},
	}

	updated := UpdateOSOKStatusCondition(status, v1beta1.Active, v1.ConditionTrue, "Reason", "same", log)
	// No new condition appended when nothing changed
	assert.Equal(t, len(status.Conditions), len(updated.Conditions))
}

func TestConvertToOciDefinedTags_Basic(t *testing.T) {
	input := map[string]v1beta1.MapValue{
		"namespace1": {"key1": "val1", "key2": "val2"},
	}

	result := ConvertToOciDefinedTags(&input)
	assert.NotNil(t, result)
	assert.Equal(t, "val1", (*result)["namespace1"]["key1"])
	assert.Equal(t, "val2", (*result)["namespace1"]["key2"])
}

func TestConvertToOciDefinedTags_Empty(t *testing.T) {
	input := map[string]v1beta1.MapValue{}
	result := ConvertToOciDefinedTags(&input)
	assert.NotNil(t, result)
	assert.Empty(t, *result)
}

func TestConvertToOciDefinedTags_MultipleNamespaces(t *testing.T) {
	input := map[string]v1beta1.MapValue{
		"ns1": {"a": "1"},
		"ns2": {"b": "2"},
	}
	result := ConvertToOciDefinedTags(&input)
	assert.Len(t, *result, 2)
	assert.Equal(t, "1", (*result)["ns1"]["a"])
	assert.Equal(t, "2", (*result)["ns2"]["b"])
}

func TestUnzipWallet_ValidZip(t *testing.T) {
	// Create a temp zip file with test data
	tmpFile, err := os.CreateTemp("", "wallet*.zip")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, err := w.Create("tnsnames.ora")
	assert.NoError(t, err)
	_, err = f.Write([]byte("test-content"))
	assert.NoError(t, err)
	w.Close()

	_, err = tmpFile.Write(buf.Bytes())
	assert.NoError(t, err)
	tmpFile.Close()

	data, err := UnzipWallet(tmpFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, data, "tnsnames.ora")
	assert.Equal(t, []byte("test-content"), data["tnsnames.ora"])
}

func TestUnzipWallet_InvalidFile(t *testing.T) {
	_, err := UnzipWallet("/nonexistent/path/wallet.zip")
	assert.Error(t, err)
}

func TestInstallResource_InvalidYAML(t *testing.T) {
	ctx := context.Background()
	err := installResource(ctx, []byte("not: valid: yaml: ["), nil, nil)
	assert.Error(t, err)
}

func TestInstallResource_EmptyData(t *testing.T) {
	ctx := context.Background()
	err := installResource(ctx, []byte(""), nil, nil)
	assert.Error(t, err)
}

func TestInstallResource_ValidYAMLNilMapper(t *testing.T) {
	// Valid YAML gets past the decode step but panics on nil mapper.
	// This exercises the decode-success path of installResource.
	validYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
`
	ctx := context.Background()
	assert.Panics(t, func() {
		_ = installResource(ctx, []byte(validYAML), nil, nil)
	})
}

func TestInitOSOK_NilConfigPanics(t *testing.T) {
	// InitOSOK panics when given a nil *rest.Config because the k8s discovery
	// client dereferences the pointer. This covers the function entry and the
	// InfoLog path that precedes the discovery-client call.
	log := testLogger()
	assert.Panics(t, func() {
		InitOSOK(nil, log)
	})
}

func TestInitOSOK_FakeConfig(t *testing.T) {
	// A non-nil config with a fake host lets InitOSOK proceed through all the
	// client-creation steps and the file loop. On a normal system the root "/"
	// contains only directories and non-yaml files, so the loop exits cleanly
	// without hitting installResource.
	log := testLogger()
	config := &rest.Config{Host: "http://localhost:19999"}
	assert.NotPanics(t, func() {
		InitOSOK(config, log)
	})
}

func TestUnzipWallet_NotZip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "notazip*.zip")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("not a zip file"))
	assert.NoError(t, err)
	tmpFile.Close()

	_, err = UnzipWallet(tmpFile.Name())
	assert.Error(t, err)
}

// makeZipWithUnsupportedMethod writes a minimal ZIP archive containing a
// single file compressed with method 99 (not registered in Go's archive/zip).
// file.Open() on such an entry returns zip.ErrAlgorithm.
func makeZipWithUnsupportedMethod(t *testing.T) string {
	t.Helper()
	const filename = "test.txt"
	const method = uint16(99) // not Store(0) or Deflate(8)
	var buf bytes.Buffer

	localOffset := 0

	writeU16 := func(v uint16) { binary.Write(&buf, binary.LittleEndian, v) }
	writeU32 := func(v uint32) { binary.Write(&buf, binary.LittleEndian, v) }

	// Local file header
	writeU32(0x04034b50) // signature
	writeU16(20)         // version needed
	writeU16(0)          // flags
	writeU16(method)     // compression method
	writeU16(0)          // last mod time
	writeU16(0)          // last mod date
	writeU32(0)          // CRC-32
	writeU32(0)          // compressed size
	writeU32(0)          // uncompressed size
	writeU16(uint16(len(filename)))
	writeU16(0) // extra field length
	buf.WriteString(filename)

	cdOffset := buf.Len()

	// Central directory header
	writeU32(0x02014b50)
	writeU16(20)
	writeU16(20)
	writeU16(0)
	writeU16(method)
	writeU16(0)
	writeU16(0)
	writeU32(0)
	writeU32(0)
	writeU32(0)
	writeU16(uint16(len(filename)))
	writeU16(0)
	writeU16(0)
	writeU16(0)
	writeU16(0)
	writeU32(0)
	writeU32(uint32(localOffset))
	buf.WriteString(filename)

	cdSize := buf.Len() - cdOffset

	// End of central directory
	writeU32(0x06054b50)
	writeU16(0)
	writeU16(0)
	writeU16(1)
	writeU16(1)
	writeU32(uint32(cdSize))
	writeU32(uint32(cdOffset))
	writeU16(0)

	tmpFile, err := os.CreateTemp("", "badmethod*.zip")
	assert.NoError(t, err)
	_, err = tmpFile.Write(buf.Bytes())
	assert.NoError(t, err)
	tmpFile.Close()
	return tmpFile.Name()
}

func TestUnzipWallet_UnsupportedCompression(t *testing.T) {
	zipPath := makeZipWithUnsupportedMethod(t)
	defer os.Remove(zipPath)

	_, err := UnzipWallet(zipPath)
	assert.Error(t, err)
}

// makeZipWithBadDeflateData builds a ZIP whose single entry uses Deflate
// compression but whose "compressed" bytes are not a valid deflate stream.
// file.Open() succeeds (returns a flate reader), but ioutil.ReadAll fails.
func makeZipWithBadDeflateData(t *testing.T) string {
	t.Helper()
	const filename = "data.txt"
	garbage := []byte{0xFF, 0xFE} // not a valid deflate stream

	var buf bytes.Buffer
	writeU16 := func(v uint16) { binary.Write(&buf, binary.LittleEndian, v) }
	writeU32 := func(v uint32) { binary.Write(&buf, binary.LittleEndian, v) }

	localOffset := 0

	// Local file header
	writeU32(0x04034b50)
	writeU16(20)
	writeU16(0)
	writeU16(8) // Deflate
	writeU16(0); writeU16(0)
	writeU32(0)                    // CRC-32 (wrong, but only checked at EOF)
	writeU32(uint32(len(garbage))) // compressed size
	writeU32(10)                   // uncompressed size (claimed)
	writeU16(uint16(len(filename)))
	writeU16(0)
	buf.WriteString(filename)
	buf.Write(garbage)

	cdOffset := buf.Len()

	// Central directory header
	writeU32(0x02014b50)
	writeU16(20); writeU16(20)
	writeU16(0)
	writeU16(8) // Deflate
	writeU16(0); writeU16(0)
	writeU32(0)
	writeU32(uint32(len(garbage)))
	writeU32(10)
	writeU16(uint16(len(filename)))
	writeU16(0); writeU16(0); writeU16(0); writeU16(0)
	writeU32(0)
	writeU32(uint32(localOffset))
	buf.WriteString(filename)

	cdSize := buf.Len() - cdOffset

	// End of central directory
	writeU32(0x06054b50)
	writeU16(0); writeU16(0)
	writeU16(1); writeU16(1)
	writeU32(uint32(cdSize))
	writeU32(uint32(cdOffset))
	writeU16(0)

	tmp, err := os.CreateTemp("", "baddeflate*.zip")
	assert.NoError(t, err)
	_, err = tmp.Write(buf.Bytes())
	assert.NoError(t, err)
	tmp.Close()
	return tmp.Name()
}

func TestUnzipWallet_ReadAllError(t *testing.T) {
	zipPath := makeZipWithBadDeflateData(t)
	defer os.Remove(zipPath)

	_, err := UnzipWallet(zipPath)
	assert.Error(t, err)
}

// makeInstallMapper builds a DeferredDiscoveryRESTMapper backed by a fake
// discovery client that advertises the given API resource lists.
func makeInstallMapper(resources []*metav1.APIResourceList) *restmapper.DeferredDiscoveryRESTMapper {
	fakeDisc := &discfake.FakeDiscovery{Fake: &k8stesting.Fake{}}
	fakeDisc.Resources = resources
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(fakeDisc))
}

func TestInstallResource_CreateNamespacedResource(t *testing.T) {
	mapper := makeInstallMapper([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
	})
	fakeDyn := dynfake.NewSimpleDynamicClient(runtime.NewScheme())

	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
`
	err := installResource(context.Background(), []byte(yaml), mapper, fakeDyn)
	assert.NoError(t, err)
}

func TestInstallResource_PatchNamespacedResource(t *testing.T) {
	mapper := makeInstallMapper([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
	})
	existing := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata":   map[string]interface{}{"name": "test-cm", "namespace": "default"},
		},
	}
	fakeDyn := dynfake.NewSimpleDynamicClient(runtime.NewScheme(), existing)
	// Patch reactor: StrategicMergePatch on unstructured is fine, but add an
	// explicit reactor so the test does not depend on strategicpatch behaviour.
	fakeDyn.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]interface{}{"name": "test-cm", "namespace": "default"},
		}}, nil
	})

	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
`
	err := installResource(context.Background(), []byte(yaml), mapper, fakeDyn)
	assert.NoError(t, err)
}

func TestInstallResource_ClusterScoped(t *testing.T) {
	mapper := makeInstallMapper([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
			},
		},
	})
	fakeDyn := dynfake.NewSimpleDynamicClient(runtime.NewScheme())

	yaml := `apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
`
	err := installResource(context.Background(), []byte(yaml), mapper, fakeDyn)
	assert.NoError(t, err)
}

func TestInstallResource_MapperNoMatch(t *testing.T) {
	// Discovery has ConfigMap, but the YAML requests an unknown CRD type.
	// The mapper cache is populated (Fresh()=true) so RESTMapping returns a
	// NoMatchError immediately without retrying.
	mapper := makeInstallMapper([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
	})
	fakeDyn := dynfake.NewSimpleDynamicClient(runtime.NewScheme())

	yaml := `apiVersion: custom.group.io/v1alpha1
kind: MyCustomResource
metadata:
  name: test-crd
  namespace: default
`
	err := installResource(context.Background(), []byte(yaml), mapper, fakeDyn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get resource mapping")
}

func TestInstallResource_GetError_NonNotFound(t *testing.T) {
	mapper := makeInstallMapper([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
	})
	fakeDyn := dynfake.NewSimpleDynamicClient(runtime.NewScheme())
	fakeDyn.PrependReactor("get", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("internal server error")
	})

	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
`
	err := installResource(context.Background(), []byte(yaml), mapper, fakeDyn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get resource by name")
}
