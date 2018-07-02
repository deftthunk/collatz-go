package main

import (
  "runtime"
  "fmt"
  "sync"
  "time"
)

var pl = fmt.Println
var pf = fmt.Printf

// STRUCT DECLARATIONS
type Sync struct {
  data    map[uint64]*Store
  record    uint64
  mu      *sync.Mutex
  ch      chan uint64
  shards    uint64
  begin    time.Time
}

type Store struct {
  dict    map[uint64]*Node
  mu      *sync.Mutex
}

type Node struct {
  value    uint64
  ts      time.Duration
  hits    uint64
}

type Hash struct {
  key      uint64
  node    *Node
}

type Ret struct {
  index    int
  rec      uint64
}

// Sync Hash Read
func (s *Sync) hashRead(key uint64) (*Node, bool) {
  store := s.data[key % s.shards]

  store.mu.Lock()
  node, ok := store.dict[key]
  store.mu.Unlock()

  return node, ok
}

// HashWrite Worker
func hashWriteWorker(s *Sync, ch chan *Hash) {
  for {
    select {
      case hash := <-ch:
        store := s.data[hash.key % s.shards]

        store.mu.Lock()
        store.dict[hash.key] = hash.node
        store.mu.Unlock()
    }
  }
}

// Sync Hash Write
func hashWrite(s *Sync, ch chan *Hash) {
  poolSize := 12
  goArr := make([]chan *Hash, poolSize)

  // spin up workers
  for i:=0; i < poolSize; i++ {
    goArr[i] = make(chan *Hash, 1000)
    go hashWriteWorker(s, goArr[i])
  }

  index := 0
  for {
    select {
      case hash := <-ch:
        goArr[index] <-hash
    }

    if index < (poolSize - 1) {
      index++
    } else {
      index = 0
    }
  }
}

// Sync Record Check
func (s *Sync) recCheck() (rec uint64) {
  s.mu.Lock()
  rec = s.record
  s.mu.Unlock()

  return
}

// Sync Record Put
func (s *Sync) recPut(rec uint64) {
  s.mu.Lock()
  s.record = rec
  s.mu.Unlock()
}

// create new Sync object
func newSync(shards uint64, threads int) Sync {
  s := Sync {
    data:    make(map[uint64]*Store, shards),
    ch:      make(chan uint64, threads),
    record:    1,
    shards:    shards,
    begin:    time.Now(),
    mu:      new(sync.Mutex),
  }

  // populate Sync.data
  for i:=uint64(0); i<shards; i++ {
    s.data[i] = &Store {
      dict:  make(map[uint64]*Node, 15000000 / shards),
      mu:    new(sync.Mutex),
    }
  }

  // seed first values
  store := s.data[1]
  store.dict[1] = &Node {
    value:    0,
    ts:      time.Since(s.begin),
    hits:    0,
  }

  return s
}

// COLLATZ/CONTROL
func control(start uint64, s *Sync, ch chan uint64, ret uint64, hwChan chan *Hash, lastProf time.Duration) {
  var length uint64 = 0
  start = start
  orig := start
  last := uint64(0)

  // start collatz
  // handle first number, to avoid unecessary lookup
  length++
  last = start
  if start & 1 == 0 {    // check for even-ness
    start /= 2
  } else {
    start = start * 3 + 1
  }

  // now start looping on result
  for {
    if start <= orig {
      myStruct, ok := s.hashRead(start)
      if ok {
        length += myStruct.value
        myStruct.ts = lastProf
        myStruct.hits++

        // update hit, then add new node for the number before the hit
        hwChan <-&Hash{start, myStruct}
        hwChan <-&Hash{last, &Node{myStruct.value + 1, lastProf, uint64(0)}}
        break
      }
    }

    length++
    last = start
    if start & 1 == 0 {    // check for even-ness
      start /= 2
      continue
    } else {
      start = start * 3 + 1
      continue
    }
  }
  // end collatz

  if length > ret {
    if length > s.recCheck() {
      pf("\r                                  ")
      pl("\rrecord:", length, "\t", orig)
      s.recPut(length)
      ret = length
    }
  }
  ch<-ret
}

// MAIN
func main() {
  var start uint64 =  3
  var end uint64 =  1000000000
  var shardCount uint64 = 2000
  var threads int = 10

  // more setups and assignments
  _ = runtime.GOMAXPROCS(threads)
  ch := make(chan uint64, threads)
  hwChan := make(chan *Hash, 100)
  profChan := make(chan uint64, 5)
  s := newSync(shardCount, threads)
  lastProf := time.Since(s.begin)    // used to update hash entries

  // kick off profiler and hashWrite routines
  go profiler(&s, profChan)
  go hashWrite(&s, hwChan)

  // spin up worker threads
  for i:=0; i<threads; i++ {
    go control(start, &s, ch, uint64(1), hwChan, lastProf)
    start++
  }

  // create thread to send 'start' value to profiler
  status := func() {
    for {
      time.Sleep(300 * time.Millisecond)
      pf("\r                                     ")
      pf("\rCur: %d                 ", start)
      lastProf = time.Since(s.begin)
    }
  }
  go status()

  // spawn control threads to compute collatz sequences
  tmpRec := uint64(1)
  for start < end {
    select {
      case ret := <-ch:
        // if returned record is larger than last (tmpRec) record
        // set tmpRec to new record value
        if ret > tmpRec {
          tmpRec = ret
        }
        start++
        go control(start, &s, ch, tmpRec, hwChan, lastProf)
        continue
    }
  }
}

