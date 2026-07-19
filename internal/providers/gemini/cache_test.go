package gemini

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/matthijn/hermes-logos/internal/protocol"
)

func TestIsValid(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		entry  CacheEntry
		expect bool
	}{
		{
			name:   "future expiry is valid",
			entry:  CacheEntry{ExpireTime: now.Add(5 * time.Minute)},
			expect: true,
		},
		{
			name:   "past expiry is invalid",
			entry:  CacheEntry{ExpireTime: now.Add(-1 * time.Second)},
			expect: false,
		},
		{
			name:   "exact expiry is invalid",
			entry:  CacheEntry{ExpireTime: now},
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValid(tt.entry, now)
			if got != tt.expect {
				t.Errorf("IsValid() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestShouldRenew(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		entry  CacheEntry
		expect bool
	}{
		{
			name:   "plenty of time remaining",
			entry:  CacheEntry{ExpireTime: now.Add(250 * time.Second), TTL: 300 * time.Second},
			expect: false,
		},
		{
			name:   "within 20% threshold",
			entry:  CacheEntry{ExpireTime: now.Add(50 * time.Second), TTL: 300 * time.Second},
			expect: true,
		},
		{
			name:   "exactly at threshold",
			entry:  CacheEntry{ExpireTime: now.Add(60 * time.Second), TTL: 300 * time.Second},
			expect: false,
		},
		{
			name:   "expired",
			entry:  CacheEntry{ExpireTime: now.Add(-1 * time.Second), TTL: 300 * time.Second},
			expect: false,
		},
		{
			name:   "1 second remaining on 300s ttl",
			entry:  CacheEntry{ExpireTime: now.Add(1 * time.Second), TTL: 300 * time.Second},
			expect: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRenew(tt.entry, now)
			if got != tt.expect {
				t.Errorf("ShouldRenew() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestCacheStore_FindValid(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		seed      map[string]CacheEntry
		hash      string
		expectOK  bool
		expectRes string
	}{
		{
			name:     "miss on empty store",
			seed:     nil,
			hash:     "abc",
			expectOK: false,
		},
		{
			name: "hit on valid entry",
			seed: map[string]CacheEntry{
				"abc": {ResourceName: "cachedContents/123", ExpireTime: now.Add(5 * time.Minute), TTL: 300 * time.Second},
			},
			hash:      "abc",
			expectOK:  true,
			expectRes: "cachedContents/123",
		},
		{
			name: "miss on expired entry",
			seed: map[string]CacheEntry{
				"abc": {ResourceName: "cachedContents/old", ExpireTime: now.Add(-1 * time.Second), TTL: 300 * time.Second},
			},
			hash:     "abc",
			expectOK: false,
		},
		{
			name: "miss on wrong key",
			seed: map[string]CacheEntry{
				"abc": {ResourceName: "cachedContents/123", ExpireTime: now.Add(5 * time.Minute), TTL: 300 * time.Second},
			},
			hash:     "def",
			expectOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewCacheStore()
			for k, v := range tt.seed {
				store.entries[k] = v
			}
			entry, ok := store.FindValid(tt.hash, now)
			if ok != tt.expectOK {
				t.Fatalf("FindValid() ok = %v, want %v", ok, tt.expectOK)
			}
			if ok && entry.ResourceName != tt.expectRes {
				t.Errorf("ResourceName = %q, want %q", entry.ResourceName, tt.expectRes)
			}
		})
	}
}

func TestCacheStore_FindValid_CleansExpired(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	store := NewCacheStore()
	store.entries["abc"] = CacheEntry{ExpireTime: now.Add(-1 * time.Second)}
	store.FindValid("abc", now)
	if _, ok := store.entries["abc"]; ok {
		t.Error("expired entry should be deleted from store")
	}
}

func TestCacheStore_AcquireOrWait_Singleflight(t *testing.T) {
	store := NewCacheStore()
	isCreator1, fl1 := store.AcquireOrWait("hash1")
	if !isCreator1 {
		t.Fatal("first caller should be creator")
	}

	isCreator2, fl2 := store.AcquireOrWait("hash1")
	if isCreator2 {
		t.Fatal("second caller should not be creator")
	}
	if fl2 != fl1 {
		t.Fatal("second caller should get same inflight")
	}

	entry := CacheEntry{ResourceName: "cachedContents/xyz", ExpireTime: time.Now().Add(5 * time.Minute), TTL: 300 * time.Second}
	store.CompleteInflight("hash1", entry, nil)

	got, err := WaitInflight(context.Background(), fl2)
	if err != nil {
		t.Fatalf("WaitInflight error: %v", err)
	}
	if got.ResourceName != "cachedContents/xyz" {
		t.Errorf("ResourceName = %q, want %q", got.ResourceName, "cachedContents/xyz")
	}

	stored, ok := store.FindValid("hash1", time.Now())
	if !ok {
		t.Fatal("entry should be stored after complete")
	}
	if stored.ResourceName != "cachedContents/xyz" {
		t.Errorf("stored ResourceName = %q, want %q", stored.ResourceName, "cachedContents/xyz")
	}
}

func TestCacheStore_ConcurrentFanout(t *testing.T) {
	store := NewCacheStore()
	const n = 30
	var wg sync.WaitGroup
	results := make([]CacheEntry, n)
	errors := make([]error, n)

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			isCreator, fl := store.AcquireOrWait("shared")
			if isCreator {
				entry := CacheEntry{
					ResourceName: "cachedContents/fanout",
					ExpireTime:   time.Now().Add(5 * time.Minute),
					TTL:          300 * time.Second,
				}
				store.CompleteInflight("shared", entry, nil)
				results[idx] = entry
			} else {
				results[idx], errors[idx] = WaitInflight(context.Background(), fl)
			}
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errors[i] != nil {
			t.Errorf("goroutine %d got error: %v", i, errors[i])
		}
		if results[i].ResourceName != "cachedContents/fanout" {
			t.Errorf("goroutine %d got ResourceName = %q, want %q", i, results[i].ResourceName, "cachedContents/fanout")
		}
	}
}

func TestCacheStore_MarkFirstRead(t *testing.T) {
	store := NewCacheStore()
	store.Store("abc", CacheEntry{
		ResourceName: "cachedContents/123",
		ExpireTime:   time.Now().Add(5 * time.Minute),
		TTL:          300 * time.Second,
	})

	tests := []struct {
		name   string
		hash   string
		expect bool
	}{
		{name: "first read returns true", hash: "abc", expect: true},
		{name: "second read returns false", hash: "abc", expect: false},
		{name: "third read returns false", hash: "abc", expect: false},
		{name: "unknown hash returns false", hash: "unknown", expect: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.MarkFirstRead(tt.hash)
			if got != tt.expect {
				t.Errorf("MarkFirstRead(%q) = %v, want %v", tt.hash, got, tt.expect)
			}
		})
	}
}

func TestCacheStore_UpdateExpiry(t *testing.T) {
	store := NewCacheStore()
	original := time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC)
	updated := time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC)

	store.Store("abc", CacheEntry{ResourceName: "cachedContents/123", ExpireTime: original, TTL: 300 * time.Second})
	store.UpdateExpiry("abc", updated)

	entry, ok := store.FindValid("abc", time.Date(2025, 1, 1, 12, 7, 0, 0, time.UTC))
	if !ok {
		t.Fatal("entry should still be valid after expiry update")
	}
	if !entry.ExpireTime.Equal(updated) {
		t.Errorf("ExpireTime = %v, want %v", entry.ExpireTime, updated)
	}
}

func TestAddCacheCreationCost(t *testing.T) {
	tests := []struct {
		name                  string
		usage                 *protocol.UsageResponse
		expectInput           int
		expectTotal           int
		expectCacheCreation   int
		expectCachedUnchanged int
	}{
		{
			name:  "nil usage unchanged",
			usage: nil,
		},
		{
			name: "no details unchanged",
			usage: &protocol.UsageResponse{
				InputTokens: 100,
				TotalTokens: 200,
			},
			expectInput: 100,
			expectTotal: 200,
		},
		{
			name: "zero cached unchanged",
			usage: &protocol.UsageResponse{
				InputTokens:        100,
				TotalTokens:        200,
				InputTokensDetails: &protocol.PromptTokensDetails{CachedTokens: 0},
			},
			expectInput: 100,
			expectTotal: 200,
		},
		{
			name: "first read adds creation tokens",
			usage: &protocol.UsageResponse{
				InputTokens:        5868,
				TotalTokens:        6500,
				InputTokensDetails: &protocol.PromptTokensDetails{CachedTokens: 5852},
			},
			expectInput:           5868 + 5852,
			expectTotal:           6500 + 5852,
			expectCacheCreation:   5852,
			expectCachedUnchanged: 5852,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addCacheCreationCost(tt.usage)
			if tt.usage == nil {
				if got != nil {
					t.Fatal("expected nil")
				}
				return
			}
			if got.InputTokens != tt.expectInput {
				t.Errorf("InputTokens = %d, want %d", got.InputTokens, tt.expectInput)
			}
			if got.TotalTokens != tt.expectTotal {
				t.Errorf("TotalTokens = %d, want %d", got.TotalTokens, tt.expectTotal)
			}
			if tt.expectCacheCreation > 0 {
				if got.InputTokensDetails == nil {
					t.Fatal("expected InputTokensDetails")
				}
				if got.InputTokensDetails.CacheCreationTokens != tt.expectCacheCreation {
					t.Errorf("CacheCreationTokens = %d, want %d", got.InputTokensDetails.CacheCreationTokens, tt.expectCacheCreation)
				}
				if got.InputTokensDetails.CachedTokens != tt.expectCachedUnchanged {
					t.Errorf("CachedTokens = %d, want %d", got.InputTokensDetails.CachedTokens, tt.expectCachedUnchanged)
				}
			}
		})
	}
}

func TestAddCacheCreationCost_DoesNotMutateInput(t *testing.T) {
	original := &protocol.UsageResponse{
		InputTokens:        5868,
		TotalTokens:        6500,
		InputTokensDetails: &protocol.PromptTokensDetails{CachedTokens: 5852},
	}
	got := addCacheCreationCost(original)
	if original.InputTokens != 5868 {
		t.Errorf("original InputTokens mutated: %d", original.InputTokens)
	}
	if original.InputTokensDetails.CacheCreationTokens != 0 {
		t.Errorf("original CacheCreationTokens mutated: %d", original.InputTokensDetails.CacheCreationTokens)
	}
	if got == original {
		t.Error("should return a new pointer")
	}
}
