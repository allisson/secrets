package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	t.Run("Success_CreateProviderWithNamespace", func(t *testing.T) {
		provider, err := NewProvider("test_app")

		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.meterProvider)
		assert.NotNil(t, provider.exporter)
		assert.NotNil(t, provider.registry)
	})

	t.Run("Success_CreateProviderWithEmptyNamespace", func(t *testing.T) {
		provider, err := NewProvider("")

		require.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestProvider_MeterProvider(t *testing.T) {
	provider, err := NewProvider("test_app")
	require.NoError(t, err)

	meterProvider := provider.MeterProvider()
	assert.NotNil(t, meterProvider)
}

func TestProvider_Handler(t *testing.T) {
	provider, err := NewProvider("test_app")
	require.NoError(t, err)

	handler := provider.Handler()
	assert.NotNil(t, handler)
}

func TestProvider_Shutdown(t *testing.T) {
	t.Run("Success_ShutdownProvider", func(t *testing.T) {
		provider, err := NewProvider("test_app")
		require.NoError(t, err)

		err = provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Success_ShutdownNilProvider", func(t *testing.T) {
		provider := &Provider{meterProvider: nil}

		err := provider.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}
