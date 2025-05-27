package easycache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalCache_Get(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		lc      *LocalCache
		key     string
		wantVal any
		wantErr error
	}{
		{
			name: "basic",
			lc: func() *LocalCache {
				lc := NewLocalCacheBuilder().Build()
				err := lc.Set(nil, "key", "val", 0)
				require.NoError(t, err)
				return lc
			}(),
			key:     "key",
			wantVal: "val",
		}, {
			name:    "key not found",
			lc:      NewLocalCacheBuilder().Build(),
			key:     "key",
			wantErr: errKeyNotFound,
		}, {
			name: "expired",
			lc: func() *LocalCache {
				lc := NewLocalCacheBuilder().Build()
				err := lc.Set(nil, "key", "val", time.Millisecond)
				require.NoError(t, err)

				time.Sleep(time.Millisecond)
				return lc
			}(),
			key:     "key",
			wantErr: errKeyNotFound,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			val, err := tc.lc.Get(context.Background(), tc.key)
			assert.Equal(t, err, tc.wantErr)

			if err == nil {
				assert.Equal(t, val, tc.wantVal)
			}
		})
	}
}

func TestLocalCache_cleanExpiredKey(t *testing.T) {
	lc := NewLocalCacheBuilder().WithInterval(time.Second).Build()

	err := lc.Set(nil, "key_1", "val_1", time.Millisecond)
	require.NoError(t, err)

	err = lc.Set(nil, "key_2", "val_2", 10*time.Millisecond)
	require.NoError(t, err)

	err = lc.Set(nil, "key_3", "val_3", 100*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(time.Second)

	err = lc.Set(nil, "key_1", "val_new", 10*time.Second)
	require.NoError(t, err)

	item, ok := lc.data["key_1"]
	assert.True(t, ok)
	assert.Equal(t, "val_new", item.val)

	_, ok = lc.data["key_2"]
	assert.False(t, ok)

	_, ok = lc.data["key_3"]
	assert.False(t, ok)
}
