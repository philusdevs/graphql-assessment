package dataloader

import (
	"sync"
	"time"

	"github.com/philusdevs/graphql-assessment/graph/model"
)

// PeopleLoaderConfig captures the config to create a new PeopleLoader
type PeopleLoaderConfig struct {
	// Fetch is a method that provides the data for the loader
	Fetch func(keys []string) ([]*model.People, []error)

	// Wait is how long wait before sending a batch
	Wait time.Duration

	// MaxBatch will limit the maximum number of keys to send in one batch, 0 = not limit
	MaxBatch int
}

// NewPeopleLoader creates a new PeopleLoader given a fetch, wait, and maxBatch
func NewPeopleLoader(config PeopleLoaderConfig) *PeopleLoader {
	return &PeopleLoader{
		fetch:    config.Fetch,
		wait:     config.Wait,
		maxBatch: config.MaxBatch,
	}
}

// PeopleLoader batches and caches requests
type PeopleLoader struct {
	// this method provides the data for the loader
	fetch func(keys []string) ([]*model.People, []error)

	// how long to do before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// INTERNAL

	// lazily created cache
	cache map[string]*model.People

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *peopleLoaderBatch

	// mutex to prevent races
	mu sync.Mutex
}

type peopleLoaderBatch struct {
	keys    []string
	data    []*model.People
	error   []error
	closing bool
	done    chan struct{}
}

// Load People by key, batching and caching will be applied automatically
func (l *PeopleLoader) Load(key string) (*model.People, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a People.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *PeopleLoader) LoadThunk(key string) func() (*model.People, error) {
	l.mu.Lock()
	if it, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return func() (*model.People, error) {
			return it, nil
		}
	}
	if l.batch == nil {
		l.batch = &peopleLoaderBatch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)
	l.mu.Unlock()

	return func() (*model.People, error) {
		<-batch.done

		var data *model.People
		if pos < len(batch.data) {
			data = batch.data[pos]
		}

		var err error
		// returns a single error for everything
		if len(batch.error) == 1 {
			err = batch.error[0]
		} else if batch.error != nil {
			err = batch.error[pos]
		}

		if err == nil {
			l.mu.Lock()
			l.unsafeSet(key, data)
			l.mu.Unlock()
		}

		return data, err
	}
}

// LoadAll fetches many keys at once. broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *PeopleLoader) LoadAll(keys []string) ([]*model.People, []error) {
	results := make([]func() (*model.People, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	peoples := make([]*model.People, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		peoples[i], errors[i] = thunk()
	}
	return peoples, errors
}

// LoadAllThunk returns a function that when called will block waiting for a Peoples.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *PeopleLoader) LoadAllThunk(keys []string) func() ([]*model.People, []error) {
	results := make([]func() (*model.People, error), len(keys))
	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}
	return func() ([]*model.People, []error) {
		peoples := make([]*model.People, len(keys))
		errors := make([]error, len(keys))
		for i, thunk := range results {
			peoples[i], errors[i] = thunk()
		}
		return peoples, errors
	}
}

// Prime the cache with the provided key and value. If the key already exists, no change is made
// and false is returned.
// (To forcefully prime the cache, clear the key first with loader.clear(key).prime(key, value).)
func (l *PeopleLoader) Prime(key string, value *model.People) bool {
	l.mu.Lock()
	var found bool
	if _, found = l.cache[key]; !found {
		// make a copy when writing to the cache, its easy to pass a pointer in from a loop var
		// and end up with the whole cache pointing to the same value.
		cpy := *value
		l.unsafeSet(key, &cpy)
	}
	l.mu.Unlock()
	return !found
}

// Clears the value at key from the cache, if the value exists
func (l *PeopleLoader) Clear(key string) {
	l.mu.Lock()
	delete(l.cache, key)
	l.mu.Unlock()
}

func (l *PeopleLoader) unsafeSet(key string, value *model.People) {
	if l.cache == nil {
		l.cache = map[string]*model.People{}
	}
	l.cache[key] = value
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *peopleLoaderBatch) keyIndex(l *PeopleLoader, key string) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *peopleLoaderBatch) startTimer(l *PeopleLoader) {
	time.Sleep(l.wait)
	l.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		l.mu.Unlock()
		return
	}

	l.batch = nil
	l.mu.Unlock()

	b.end(l)
}

func (b *peopleLoaderBatch) end(l *PeopleLoader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}
