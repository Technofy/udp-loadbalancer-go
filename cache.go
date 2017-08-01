package main

import (
	"time"
)

type CacheEntry struct {
	Data interface{}
	ExpirationTime int64
}

func (ce *CacheEntry) IsExpired() bool {
	return time.Now().Unix() > ce.ExpirationTime;
}

type CacheManager struct {
	CheckingTime int64
	DefaultExpirationTime int64
	Entries map[uint32]CacheEntry
}

func (cm *CacheManager) GetKeys() []uint32 {
	keys := make([]uint32, len(cm.Entries))

	i := 0
	for k := range cm.Entries {
		keys[i] = k
		i++
	}

	return keys
}

func (cm *CacheManager) Exists(entry uint32) bool {
	_, ok := cm.Entries[entry]

	return ok
}

func (cm *CacheManager) DeleteEntry(entry uint32) {
	delete(cm.Entries, entry)
}

func (cm *CacheManager) Invalidate() {
	cm.Entries = make(map[uint32]CacheEntry)
}

func (cm *CacheManager) Get(entry uint32) interface{} {
	return cm.Entries[entry].Data
}

func (cm *CacheManager) Add(entry uint32, data interface{}) {
	cm.Entries[entry] = CacheEntry{
		Data: data,
		ExpirationTime: cm.DefaultExpirationTime,
	}
}

func (cm *CacheManager) ResetTimer(entry uint32) {
	centry := cm.Entries[entry]
	centry.ExpirationTime = time.Now().Unix() + cm.DefaultExpirationTime
	cm.Entries[entry] = centry
}

func (cm *CacheManager) PerformCleanup() {
	for k, v := range cm.Entries {
		if v.IsExpired() {
			delete(cm.Entries, k)
		}
	}
}

func (cm *CacheManager) AutoCacheCleaner() {
	ticker := time.NewTicker(time.Second * time.Duration(cm.CheckingTime))
	for range ticker.C {
		cm.PerformCleanup()
	}
}

func NewCacheManager(defaultExpiration int64, checkingTime int64) (*CacheManager, error) {
	return &CacheManager{
		DefaultExpirationTime: defaultExpiration,
		CheckingTime: checkingTime,
		Entries: make(map[uint32]CacheEntry),
	}, nil
}

func MustNewCacheManager(defaultExpiration int64, checkingTime int64) *CacheManager {
	cm, err := NewCacheManager(defaultExpiration, checkingTime)
	if err != nil {
		panic(err)
	}

	return cm
}
