package main

import (
  "time"
  "runtime"
)

// global vars
type pData struct {
  oldSize       uint64
  prevNum       uint64
  curNum        uint64
  decay         uint64
  decayUpperLim uint64
  decayLowerLim uint64
  delay         time.Duration
  profChan      chan uint64
}

// create new pData object
func newPData(pc chan uint64) pData {
  p := pData {
    oldSize:    0,
    prevNum:    0,
    curNum:      0,
    decay:      17,
    decayUpperLim:  20,
    decayLowerLim:  15,
    delay:      time.Duration(25) * time.Second,
    profChan:    pc,
  }

  return p
}

// TIMER
func timer(p *pData) {
  begin := time.Now()
  second := time.Second
  counter := time.Duration(0)
  rCounter := 0

  for {
    time.Sleep(second)

    // increment profiler timer to ceiling
    if time.Since(begin) >= p.delay {
      if p.delay < (time.Duration(50) * second) {
        p.delay += (time.Duration(10) * second)
      }
      p.profChan <-2    // start profile routine
      counter = time.Duration(0)
      begin = time.Now()
    }
    counter += second
    rCounter += 1
  }
}

// PROFILER
func profiler(s *Sync, pc chan uint64) {
  // make new pData object
  p := newPData(pc)

  // kick off timer thread
  go timer(&p)

  // main profiler loop
  for {
    // stall until we get signal to profile
    for {
      select {
        case start := <-p.profChan:
          if start == 2 {
            goto Profile
          }
      }
    }

Profile:  // goto tag
    second := time.Second
    size := uint64(0)
    //hits := make(map[int]int)    // part of the debug profiling routine

    // iterate through hashmap, lock child maps, and
    // profile data and delete old entries
    for _, store := range s.data {
      store.mu.Lock()
      for key, node := range store.dict {
        if uint64(node.ts / second) > p.decay {
          if key > 5 {
            delete(store.dict, key)
          }
        }
        size++
        //hits[int(key)] = int(node.hits)
      }
      store.mu.Unlock()
    }
    runtime.Gosched()

    // perf section
    pf("\r                                                          ")
    pf("\r==Profiling== ::\tdelay: %d\tdecay: %d\tsz: %d\n", (p.delay / second), p.decay, size)

    if size - p.oldSize > 0 {
      if p.decay > p.decayLowerLim {
        p.decay -= 1
      }
    } else if p.decay < p.decayUpperLim {
      p.decay += 5
    }

    p.oldSize = size
  }
}
