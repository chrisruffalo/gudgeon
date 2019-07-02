package rule

import (
	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/events"
	"sync"
)

type reloadingStore struct {
	handlers []*events.Handle
	delegate Store
	mux      sync.RWMutex
}

func (reloadingStore *reloadingStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.Lock()
		reloadingStore.delegate.Init(sessionRoot, config, lists)
		reloadingStore.mux.Unlock()
	}
}

func (reloadingStore *reloadingStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.Lock()
		reloadingStore.delegate.Clear(config, list)
		reloadingStore.mux.Unlock()
	}
}

func (reloadingStore *reloadingStore) Load(list *config.GudgeonList, rule string) {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.Lock()
		reloadingStore.delegate.Load(list, rule)
		reloadingStore.mux.Unlock()
	}
}

func (reloadingStore *reloadingStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.Lock()
		reloadingStore.delegate.Finalize(sessionRoot, lists)
		reloadingStore.mux.Unlock()
	}
}

func (reloadingStore *reloadingStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.RLock()
		defer reloadingStore.mux.RUnlock()
		return reloadingStore.delegate.FindMatch(lists, domain)
	}
	return MatchNone, nil, ""
}

func (reloadingStore *reloadingStore) Close() {
	if reloadingStore.delegate != nil {
		reloadingStore.mux.Lock()
		// close handles
		for _, handle := range reloadingStore.handlers {
			if handle != nil {
				handle.Close()
			}
		}
		reloadingStore.delegate.Close()
		reloadingStore.mux.Unlock()
	}
}
