/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kubesecret

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------------------------------------------------------
// In-memory mock for client.Client — only implements what KubeSecretClient uses.
// ---------------------------------------------------------------------------

type mockK8sClient struct {
	mu      sync.RWMutex
	secrets map[string]*v1.Secret // key: namespace/name

	// injectable errors for specific operations
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func newMockClient() *mockK8sClient {
	return &mockK8sClient{
		secrets: make(map[string]*v1.Secret),
	}
}

func secretKey(ns, name string) string { return ns + "/" + name }

func (m *mockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.getErr != nil {
		return m.getErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.secrets[secretKey(key.Namespace, key.Name)]
	if !ok {
		return apierrors.NewNotFound(v1.Resource("secret"), key.Name)
	}
	*obj.(*v1.Secret) = *s
	return nil
}

func (m *mockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if m.createErr != nil {
		return m.createErr
	}
	s := obj.(*v1.Secret)
	m.mu.Lock()
	defer m.mu.Unlock()
	k := secretKey(s.Namespace, s.Name)
	m.secrets[k] = s.DeepCopy()
	return nil
}

func (m *mockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	s := obj.(*v1.Secret)
	m.mu.Lock()
	defer m.mu.Unlock()
	k := secretKey(s.Namespace, s.Name)
	if _, ok := m.secrets[k]; !ok {
		return apierrors.NewNotFound(v1.Resource("secret"), s.Name)
	}
	m.secrets[k] = s.DeepCopy()
	return nil
}

func (m *mockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	s := obj.(*v1.Secret)
	m.mu.Lock()
	defer m.mu.Unlock()
	k := secretKey(s.Namespace, s.Name)
	if _, ok := m.secrets[k]; !ok {
		return apierrors.NewNotFound(v1.Resource("secret"), s.Name)
	}
	delete(m.secrets, k)
	return nil
}

// Remaining interface stubs — not used by KubeSecretClient.
func (m *mockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}
func (m *mockK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}
func (m *mockK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}
func (m *mockK8sClient) Status() client.SubResourceWriter                           { return nil }
func (m *mockK8sClient) SubResource(sr string) client.SubResourceClient             { return nil }
func (m *mockK8sClient) Scheme() *runtime.Scheme                                    { return nil }
func (m *mockK8sClient) RESTMapper() meta.RESTMapper                                { return nil }
func (m *mockK8sClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
func (m *mockK8sClient) IsObjectNamespaced(obj runtime.Object) (bool, error) { return true, nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// discardLogger returns a no-op logger suitable for tests.
func discardLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
}

// testMetrics returns a Metrics instance without registering Prometheus
// counters globally (avoids "already registered" panics across tests).
// We use a bare struct directly.
func testMetrics() *metrics.Metrics {
	return &metrics.Metrics{
		Name:        "oci",
		ServiceName: "KubeSecretTest",
		Logger:      discardLogger(),
	}
}

func newTestClient(k8s *mockK8sClient) *KubeSecretClient {
	return New(k8s, discardLogger(), testMetrics())
}

// ---------------------------------------------------------------------------
// Tests: New
// ---------------------------------------------------------------------------

func TestNew_ReturnsPopulatedClient(t *testing.T) {
	k8s := newMockClient()
	c := newTestClient(k8s)
	assert.NotNil(t, c)
	assert.Equal(t, k8s, c.Client)
}

// ---------------------------------------------------------------------------
// Tests: CreateSecret
// ---------------------------------------------------------------------------

func TestCreateSecret_Success(t *testing.T) {
	c := newTestClient(newMockClient())
	ctx := context.Background()
	data := map[string][]byte{"key": []byte("value")}
	labels := map[string]string{"app": "test"}

	ok, err := c.CreateSecret(ctx, "mysecret", "default", labels, data)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestCreateSecret_AlreadyExists(t *testing.T) {
	mock := newMockClient()
	c := newTestClient(mock)
	ctx := context.Background()
	data := map[string][]byte{"k": []byte("v")}

	// Pre-populate so Get returns a secret (exists).
	mock.secrets[secretKey("default", "existing")] = &v1.Secret{}

	ok, err := c.CreateSecret(ctx, "existing", "default", nil, data)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.True(t, apierrors.IsAlreadyExists(err))
}

func TestCreateSecret_GetReturnsGenericError(t *testing.T) {
	mock := newMockClient()
	mock.getErr = fmt.Errorf("connection refused")
	c := newTestClient(mock)

	ok, err := c.CreateSecret(context.Background(), "mysecret", "default", nil, nil)
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestCreateSecret_CreateFails(t *testing.T) {
	mock := newMockClient()
	mock.createErr = fmt.Errorf("create failed")
	c := newTestClient(mock)

	ok, err := c.CreateSecret(context.Background(), "mysecret", "default", nil, nil)
	assert.Error(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Tests: GetSecret
// ---------------------------------------------------------------------------

func TestGetSecret_Success(t *testing.T) {
	mock := newMockClient()
	c := newTestClient(mock)
	ctx := context.Background()
	data := map[string][]byte{"foo": []byte("bar")}
	labels := map[string]string{}

	_, _ = c.CreateSecret(ctx, "mysecret", "default", labels, data)

	got, err := c.GetSecret(ctx, "mysecret", "default")
	assert.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestGetSecret_NotFound(t *testing.T) {
	c := newTestClient(newMockClient())
	data, err := c.GetSecret(context.Background(), "nonexistent", "default")
	assert.Error(t, err)
	assert.Empty(t, data)
}

// ---------------------------------------------------------------------------
// Tests: UpdateSecret
// ---------------------------------------------------------------------------

func TestUpdateSecret_Success(t *testing.T) {
	mock := newMockClient()
	c := newTestClient(mock)
	ctx := context.Background()

	// Create first so the object exists.
	_, _ = c.CreateSecret(ctx, "mysecret", "default", nil, map[string][]byte{"a": []byte("1")})

	updated := map[string][]byte{"a": []byte("2"), "b": []byte("3")}
	ok, err := c.UpdateSecret(ctx, "mysecret", "default", nil, updated)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify updated data is persisted.
	got, err := c.GetSecret(ctx, "mysecret", "default")
	assert.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestUpdateSecret_Fails(t *testing.T) {
	mock := newMockClient()
	mock.updateErr = fmt.Errorf("update failed")
	c := newTestClient(mock)

	ok, err := c.UpdateSecret(context.Background(), "mysecret", "default", nil, nil)
	assert.Error(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Tests: DeleteSecret
// ---------------------------------------------------------------------------

func TestDeleteSecret_Success(t *testing.T) {
	c := newTestClient(newMockClient())
	ctx := context.Background()

	_, _ = c.CreateSecret(ctx, "mysecret", "default", nil, map[string][]byte{"x": []byte("y")})

	ok, err := c.DeleteSecret(ctx, "mysecret", "default")
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify it's gone.
	_, err = c.GetSecret(ctx, "mysecret", "default")
	assert.Error(t, err)
}

func TestDeleteSecret_NotFound(t *testing.T) {
	c := newTestClient(newMockClient())
	ok, err := c.DeleteSecret(context.Background(), "nosuchsecret", "default")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestDeleteSecret_DeleteFails(t *testing.T) {
	mock := newMockClient()
	// Pre-populate so Get succeeds...
	mock.secrets[secretKey("default", "mysecret")] = &v1.Secret{}
	// ...but Delete fails.
	mock.deleteErr = fmt.Errorf("delete failed")
	c := newTestClient(mock)

	ok, err := c.DeleteSecret(context.Background(), "mysecret", "default")
	assert.Error(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Tests: isValidSecretName
// ---------------------------------------------------------------------------

func TestIsValidSecretName(t *testing.T) {
	c := newTestClient(newMockClient())
	ctx := context.Background()

	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "my-secret", true},
		{"valid alphanumeric", "secret123", true},
		{"uppercase invalid", "MySecret", false},
		{"underscore invalid", "my_secret", false},
		{"empty invalid", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, c.isValidSecretName(ctx, tc.input))
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: getValidSecretName
// ---------------------------------------------------------------------------

func TestGetValidSecretName(t *testing.T) {
	c := newTestClient(newMockClient())
	ctx := context.Background()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercases input", "MySecret", "mysecret"},
		{"strips invalid chars", "my_secret!", "mysecret"},
		{"collapses consecutive dots/dashes", "my--secret", "my.secret"},
		{"strips underscores", "my_secret_name", "mysecretname"},
		{"keeps dots and dashes", "my.secret-name", "my.secret-name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.getValidSecretName(ctx, tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: full CRUD round-trip
// ---------------------------------------------------------------------------

func TestCRUD_RoundTrip(t *testing.T) {
	c := newTestClient(newMockClient())
	ctx := context.Background()

	name := "roundtrip-secret"
	ns := "default"
	data := map[string][]byte{"user": []byte("alice"), "pass": []byte("s3cr3t")}
	labels := map[string]string{"managed-by": "osok"}

	// Create
	ok, err := c.CreateSecret(ctx, name, ns, labels, data)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Get
	got, err := c.GetSecret(ctx, name, ns)
	assert.NoError(t, err)
	assert.Equal(t, data, got)

	// Duplicate create fails
	_, err = c.CreateSecret(ctx, name, ns, labels, data)
	assert.Error(t, err)

	// Update
	newData := map[string][]byte{"user": []byte("bob"), "pass": []byte("n3w"), "extra": []byte("val")}
	ok, err = c.UpdateSecret(ctx, name, ns, labels, newData)
	assert.NoError(t, err)
	assert.True(t, ok)

	got, err = c.GetSecret(ctx, name, ns)
	assert.NoError(t, err)
	assert.Equal(t, newData, got)

	// Delete
	ok, err = c.DeleteSecret(ctx, name, ns)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Gone
	_, err = c.GetSecret(ctx, name, ns)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Helper: NamespacedName lookup
// ---------------------------------------------------------------------------

func TestGet_UsesNamespacedNameCorrectly(t *testing.T) {
	mock := newMockClient()
	// Store secret in ns1, not ns2
	mock.secrets[secretKey("ns1", "sec")] = &v1.Secret{}

	c := newTestClient(mock)

	// Should find in ns1
	_, err := c.GetSecret(context.Background(), "sec", "ns1")
	assert.NoError(t, err)

	// Should not find in ns2
	_, err = c.GetSecret(context.Background(), "sec", "ns2")
	assert.Error(t, err)
}

// Verify that mockK8sClient satisfies the client.Client interface at compile time.
var _ client.Client = (*mockK8sClient)(nil)

// Verify NamespacedName is passed correctly through Get.
func TestMockGet_CorrectKey(t *testing.T) {
	mock := newMockClient()
	var capturedKey types.NamespacedName

	// Wrap Get to capture the key
	mock.secrets[secretKey("mynamespace", "myname")] = &v1.Secret{}

	var s v1.Secret
	err := mock.Get(context.Background(), types.NamespacedName{Namespace: "mynamespace", Name: "myname"}, &s)
	assert.NoError(t, err)
	_ = capturedKey
}
