package dataloadgen_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/graph-gophers/dataloader"
	"github.com/mshaeon/dataloadgen"
	"github.com/vektah/dataloaden/example"
)

// Benchmarks copied from https://github.com/vektah/dataloaden

type benchmarkUser struct {
	Name string
	ID   string
}

func BenchmarkDataloader(b *testing.B) {
	ctx := context.Background()
	dl := dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		users := make([]*dataloader.Result, len(keys))

		for i, key := range keys {
			if rand.Int()%100 == 1 {
				users[i] = &dataloader.Result{Error: fmt.Errorf("user not found")}
			} else if rand.Int()%100 == 1 {
				users[i] = &dataloader.Result{}
			} else {
				users[i] = &dataloader.Result{Data: &benchmarkUser{ID: key.String(), Name: "user " + key.String()}}
			}
		}
		return users
	},
		dataloader.WithBatchCapacity(100),
		dataloader.WithWait(500*time.Nanosecond),
	)

	b.Run("caches", func(b *testing.B) {
		queries := []IntKey{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, IntKey(rand.Int()%300))
		}
		b.ResetTimer()
		thunks := make([]func() (interface{}, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.Load(ctx, queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("random spread", func(b *testing.B) {
		queries := []IntKey{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, IntKey(rand.Int()))
		}
		b.ResetTimer()
		thunks := make([]func() (interface{}, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.Load(ctx, queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("concurently", func(b *testing.B) {
		queries := []IntKey{}
		for n := 0; n < b.N*10; n++ {
			queries = append(queries, IntKey(rand.Int()))
		}
		b.ResetTimer()
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				for j := 0; j < b.N; j++ {
					dl.Load(ctx, queries[i*b.N+j])()
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}

// IntKey implements the Key interface for an int
type IntKey int

// String is an identity method. Used to implement String interface
func (k IntKey) String() string { return strconv.Itoa(int(k)) }

// String is an identity method. Used to implement Key Raw
func (k IntKey) Raw() interface{} { return k }

func BenchmarkDataloadgen(b *testing.B) {
	dl := dataloadgen.NewLoader(func(keys []int) (map[int]benchmarkUser, error) {
		users := make(map[int]benchmarkUser, len(keys))
		errors := make(dataloadgen.ErrorMap[int], len(keys))

		for i, key := range keys {
			if key%100 == 1 {
				errors[i] = fmt.Errorf("user not found")
			} else if key%100 == 1 {
				users[i] = benchmarkUser{}
			} else {
				users[i] = benchmarkUser{ID: strconv.Itoa(key), Name: "user " + strconv.Itoa(key)}
			}
		}
		return users, errors
	},
		dataloadgen.WithBatchCapacity(100),
		dataloadgen.WithWait(500*time.Nanosecond),
	)

	b.Run("caches", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, rand.Int()%300)
		}
		b.ResetTimer()
		thunks := make([]func() (benchmarkUser, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.LoadThunk(queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("random spread", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, rand.Int())
		}
		b.ResetTimer()
		thunks := make([]func() (benchmarkUser, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.LoadThunk(queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("concurently", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < 10*b.N; n++ {
			queries = append(queries, rand.Int())
		}
		b.ResetTimer()
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				for j := 0; j < b.N; j++ {
					dl.Load(queries[j+i*b.N])
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}
func BenchmarkDataloaden(b *testing.B) {
	dl := NewUserLoader(UserLoaderConfig{
		Wait:     500 * time.Nanosecond,
		MaxBatch: 100,
		Fetch: func(keys []int) ([]*example.User, []error) {
			users := make([]*example.User, len(keys))
			errors := make([]error, len(keys))

			for i, key := range keys {
				if rand.Int()%100 == 1 {
					errors[i] = fmt.Errorf("user not found")
				} else if rand.Int()%100 == 1 {
					users[i] = nil
				} else {
					users[i] = &example.User{ID: strconv.Itoa(key), Name: "user " + strconv.Itoa(key)}
				}
			}
			return users, errors
		},
	})

	b.Run("caches", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, rand.Int()%300)
		}
		b.ResetTimer()
		thunks := make([]func() (*example.User, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.LoadThunk(queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("random spread", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < b.N; n++ {
			queries = append(queries, rand.Int())
		}
		b.ResetTimer()
		thunks := make([]func() (*example.User, error), b.N)
		for i := 0; i < b.N; i++ {
			thunks[i] = dl.LoadThunk(queries[i])
		}

		for i := 0; i < b.N; i++ {
			thunks[i]()
		}
	})

	b.Run("concurently", func(b *testing.B) {
		queries := []int{}
		for n := 0; n < b.N*10; n++ {
			queries = append(queries, rand.Int())
		}
		b.ResetTimer()
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				for j := 0; j < b.N; j++ {
					dl.Load(queries[i*b.N+j])
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}
