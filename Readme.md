# Velox — High-Performance In-Memory Database

[![Go](https://github.com/sharpsalt/Velox-In-Memory-Database/actions/workflows/go.yml/badge.svg)](https://github.com/sharpsalt/Velox-In-Memory-Database/actions)

Velox is a Redis-compatible, in-memory key-value store built entirely from scratch in Go. It speaks the Redis wire protocol (RESP), supports the same data types Redis does, and achieves multi-million requests-per-second throughput on a single thread by exploiting Linux's `epoll` event loop, `sync.Pool`-based zero-allocation buffers, and hand-written RESP encoding.

```
<img width="923" height="119" alt="image" src="https://github.com/user-attachments/assets/41b479d0-7205-4d5f-b090-f23d3bcf3113" />

```

---

## Table of Contents

1. [Why Velox?](#why-velox)
2. [Architecture Overview](#architecture-overview)
3. [Repository Layout](#repository-layout)
4. [Getting Started](#getting-started)
5. [Configuration](#configuration)
6. [Supported Commands](#supported-commands)
7. [Data Types & Internal Encodings](#data-types--internal-encodings)
8. [Core Subsystems — Deep Dive](#core-subsystems--deep-dive)
   - [Event Loop (epoll)](#1-event-loop-async_tcpgo)
   - [RESP Protocol](#2-resp-protocol-respgo)
   - [Key-Value Store](#3-key-value-store-storego)
   - [Object Model](#4-object-model-objectgo)
   - [Type Encoding](#5-type-encoding-typeencodingg)
   - [Key Expiry](#6-key-expiry-expiredgo)
   - [Eviction — Approximated LRU](#7-eviction--approximated-lru)
   - [AOF Persistence](#8-aof-persistence-aofgo)
   - [Pub/Sub](#9-pubsub-pubsubgo)
   - [Transactions](#10-transactions-commgo)
   - [Data Structures](#11-data-structures)
   - [Metrics](#12-metrics-metricsgo)
9. [Performance Engineering](#performance-engineering)
10. [Benchmark Tool](#benchmark-tool-velox-benchmark)
11. [CLI Client](#cli-client-velox-cli)
12. [Running Tests](#running-tests)
13. [Known Limitations & TODOs](#known-limitations--todos)

---

## Why Velox?

Velox was built as a ground-up learning exercise in high-performance systems programming. Every design decision — from the choice of `epoll` over Go's standard `net` package, to swapping `fmt.Sprintf` for `strconv.AppendInt`, to pooling 64 KB read buffers — is intentional and documented inside the code. Reading this codebase explains *why* databases like Redis are fast, not just *that* they are fast.

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│                     Client Connections                    │
│          (up to 20,000; idle clients cost ~0 RAM)        │
└────────────────────────┬─────────────────────────────────┘
                         │  TCP on :7379  (RESP wire protocol)
┌────────────────────────▼─────────────────────────────────┐
│                  Event Loop  (epoll)                      │
│   async_tcp.go — single goroutine, non-blocking sockets  │
│                                                          │
│  ┌───────────┐   ┌──────────────┐   ┌─────────────────┐ │
│  │ Accept FD │   │ Read command │   │ Write response  │ │
│  │ Register  │   │ (ReadBufPool)│   │ (WriteBufPool)  │ │
│  │ in epoll  │   │ DecodeRESP   │   │  Encode RESP    │ │
│  └───────────┘   └──────┬───────┘   └────────▲────────┘ │
│                         │                    │           │
│                  ┌──────▼────────────────────┘           │
│                  │    Command Dispatch  (eval.go)        │
│                  │    executeCommand() switch            │
│                  └──────┬────────────────────────────────┘
│                         │
│          ┌──────────────┼──────────────┬─────────────┐
│          ▼              ▼              ▼             ▼
│     String         Hash            List         Set / ZSet
│     eval.go   eval_hash.go   eval_list.go   eval_set/zset.go
│          │              │              │             │
│          └──────────────┴──────────────┴──────────── ┘
│                                 │
│                         store.go  (RWMutex-protected map)
│                         expire.go (separate expires map)
│                         eviction.go + evictionpool.go
│                         aof.go    (persistence)
│                         pubsub.go (channels)
│                         metrics.go
└──────────────────────────────────────────────────────────┘
```

**Single-threaded, event-driven.** Like Redis, Velox processes commands on a single goroutine. Concurrency safety on the store comes from a `sync.RWMutex`, not from multi-threading. A separate goroutine handles OS signals (SIGTERM/SIGINT).

---

## Repository Layout

```
Velox-In-Memory-Database/
│
├── main.go                     # Entry point: flag parsing, signal handling, goroutine wiring
│
├── server/
│   ├── async_tcp.go            # epoll event loop, client accept/read/write lifecycle
│   └── sync_tcp.go             # legacy synchronous server (reference implementation)
│
├── core/
│   ├── eval.go                 # String commands + command dispatcher (executeCommand)
│   ├── eval_hash.go            # HSET/HGET/HGETALL/HDEL/HLEN/HEXISTS/HKEYS/HVALS/HINCRBY/HSETNX
│   ├── eval_list.go            # LPUSH/RPUSH/LPOP/RPOP/LRANGE/LLEN/LINDEX
│   ├── eval_set.go             # SADD/SREM/SMEMBERS/SISMEMBER/SCARD/SINTER/SUNION
│   ├── eval_zset.go            # ZADD/ZSCORE/ZRANK/ZRANGE/ZCARD/ZREM
│   ├── resp.go                 # RESP encoder + zero-allocation decoder (DecodeCommands)
│   ├── store.go                # Core hash table (map[string]*Obj) with RWMutex
│   ├── object.go               # Obj struct: TypeEncoding, Value, LastAccessedAt
│   ├── typeencoding.go         # Bit-packed type+encoding helpers, TypeName, EncodingName
│   ├── expire.go               # hasExpired, getExpiry, active deletion (DeleteExpiredKey)
│   ├── eviction.go             # evict(), evictFirst/Random/LRU, populateEvictionPool
│   ├── evictionpool.go         # Max-heap of eviction candidates (EvictionPool)
│   ├── aof.go                  # Append-Only File: DumpAllAOF, ReplayAOF
│   ├── pubsub.go               # SUBSCRIBE/UNSUBSCRIBE/PUBLISH channel routing
│   ├── comm.go                 # Client struct, TxnBegin/Exec/Discard, ReadBufPool/WriteBufPool
│   ├── cmd.go                  # RedisCmd struct and RedisCmds type
│   ├── events.go               # GlobalCachedTime, InitStore, Shutdown
│   ├── metrics.go              # ServerMetrics struct, TrackCommand/Connection, OpsPerSecond
│   ├── stats.go                # KeyspaceStat (4-database keyspace counters)
│   ├── ziplist.go              # Compact flat-slice storage (hash + quicklist nodes)
│   ├── quicklist.go            # Doubly-linked list of ZipList nodes (Redis list encoding)
│   ├── skiplist.go             # Probabilistic skip list (sorted sets / ZSet)
│   ├── intset.go               # Sorted integer array (small integer-only sets)
│   └── type_string.go          # deduceTypeEncoding helper for string values
│
├── config/
│   └── config.go               # All server tuning knobs (port, eviction, limits, AOF path)
│
├── velox-benchmark/
│   └── main.go                 # Concurrent benchmark tool with pipelining
│
├── velox-cli/
│   └── main.go                 # Interactive REPL client
│
├── integration_test.go         # End-to-end command tests
├── main_test.go                # Startup/shutdown tests
└── go.mod
```

---

## Getting Started

### Prerequisites

- Go 1.21 or later
- Linux (the event loop uses `syscall.EpollCreate1` — Linux-only)

### Build & Run the Server

```bash
# Clone the repository
git clone https://github.com/sharpsalt/Velox-In-Memory-Database.git
cd Velox-In-Memory-Database

# Build the server binary
go build -o bin/velox .

# Start the server (defaults: host=0.0.0.0, port=7379)
./bin/velox

# Or specify host/port explicitly
./bin/velox -host 127.0.0.1 -port 7379
```

You should see:

```
2024/01/01 00:00:00 hello!! is it really running
2024/01/01 00:00:00 Initializing store — replaying AOF if available...
2024/01/01 00:00:00 no AOF file found at ./velox.aof — starting with empty database
2024/01/01 00:00:00 Store initialized, total keys: 0
2024/01/01 00:00:00 Starting an asynchronous TCP Server on 0.0.0.0 7379
```

### Connect with the Built-in CLI

```bash
go run ./velox-cli

# Output:
# Connected to Velox Database!
# Type your commands below (e.g., SET name srijan, GET name). Type 'exit' to quit.
velox> SET name srijan
+OK
velox> GET name
$6
srijan
velox> TTL name
:-1
velox> exit
```

### Connect with redis-cli

Because Velox speaks RESP, any Redis client works out of the box:

```bash
redis-cli -p 7379
127.0.0.1:7379> PING
PONG
127.0.0.1:7379> SET counter 0
OK
127.0.0.1:7379> INCR counter
(integer) 1
```

---

## Configuration

All knobs live in `config/config.go` and can be overridden via source before building. Future work could add environment variable or config-file support.

| Variable | Default | Description |
|---|---|---|
| `Host` | `"0.0.0.0"` | Listening interface |
| `Port` | `7379` | Listening port |
| `KeysLimit` | `100` | Max number of keys before eviction triggers |
| `EvictionStrategy` | `"allkeys-lru"` | One of `simple-first`, `allkeys-random`, `allkeys-lru` |
| `EvictionRatio` | `0.40` | Fraction of keys to evict per eviction run (40%) |
| `LRUSampleSize` | `5` | Random keys sampled per eviction round (Redis default) |
| `AOFFile` | `"./velox.aof"` | Path for Append-Only File persistence |
| `HashMaxZiplistEntries` | `128` | Hash ziplist → hashtable promotion threshold (entries) |
| `HashMaxZiplistValue` | `64` | Hash ziplist → hashtable promotion threshold (value size in bytes) |
| `ListMaxZiplistSize` | `128` | Max elements per QuickList node |
| `SetMaxIntsetEntries` | `512` | IntSet → hashtable promotion threshold |

**Flag overrides at runtime:**

```bash
./bin/velox -host 127.0.0.1 -port 6380
```

---

## Supported Commands

### String

| Command | Syntax | Description |
|---|---|---|
| `PING` | `PING [message]` | Returns PONG or the given message |
| `SET` | `SET key value [EX seconds]` | Set a key with optional expiry |
| `GET` | `GET key` | Get the value of a key |
| `DEL` | `DEL key [key ...]` | Delete one or more keys |
| `EXISTS` | `EXISTS key [key ...]` | Count how many keys exist |
| `INCR` | `INCR key` | Increment integer by 1 |
| `DECR` | `DECR key` | Decrement integer by 1 |
| `INCRBY` | `INCRBY key increment` | Increment by a given amount |
| `DECRBY` | `DECRBY key decrement` | Decrement by a given amount |
| `MSET` | `MSET k1 v1 k2 v2 ...` | Set multiple keys at once |
| `MGET` | `MGET k1 k2 ...` | Get multiple values at once |
| `TTL` | `TTL key` | Returns TTL in seconds (-1 = no expiry, -2 = missing) |
| `EXPIRE` | `EXPIRE key seconds` | Set a TTL on an existing key |
| `TYPE` | `TYPE key` | Returns the type of the value (`string`, `hash`, etc.) |
| `KEYS` | `KEYS pattern` | Returns all keys matching a glob pattern |
| `DBSIZE` | `DBSIZE` | Returns the number of keys in the store |
| `FLUSHDB` | `FLUSHDB` | Deletes all keys |
| `SLEEP` | `SLEEP seconds` | Sleeps for N seconds (for testing) |

### Hash

| Command | Syntax | Description |
|---|---|---|
| `HSET` | `HSET key field value [field value ...]` | Set one or more fields |
| `HGET` | `HGET key field` | Get a single field's value |
| `HGETALL` | `HGETALL key` | Get all fields and values |
| `HDEL` | `HDEL key field [field ...]` | Delete one or more fields |
| `HLEN` | `HLEN key` | Number of fields in the hash |
| `HEXISTS` | `HEXISTS key field` | Check if a field exists |
| `HKEYS` | `HKEYS key` | Return all field names |
| `HVALS` | `HVALS key` | Return all field values |
| `HINCRBY` | `HINCRBY key field increment` | Increment a field's integer value |
| `HSETNX` | `HSETNX key field value` | Set a field only if it does not exist |

### List

| Command | Syntax | Description |
|---|---|---|
| `LPUSH` | `LPUSH key element [element ...]` | Prepend elements to the list head |
| `RPUSH` | `RPUSH key element [element ...]` | Append elements to the list tail |
| `LPOP` | `LPOP key` | Remove and return the leftmost element |
| `RPOP` | `RPOP key` | Remove and return the rightmost element |
| `LRANGE` | `LRANGE key start stop` | Return elements in the index range |
| `LLEN` | `LLEN key` | Return the list length |
| `LINDEX` | `LINDEX key index` | Return the element at index (supports negative indices) |

### Set

| Command | Syntax | Description |
|---|---|---|
| `SADD` | `SADD key member [member ...]` | Add members to a set |
| `SREM` | `SREM key member [member ...]` | Remove members from a set |
| `SMEMBERS` | `SMEMBERS key` | Return all members |
| `SISMEMBER` | `SISMEMBER key member` | Check membership |
| `SCARD` | `SCARD key` | Return the number of members |
| `SINTER` | `SINTER key [key ...]` | Return the intersection |
| `SUNION` | `SUNION key [key ...]` | Return the union |

### Sorted Set (ZSet)

| Command | Syntax | Description |
|---|---|---|
| `ZADD` | `ZADD key score member [score member ...]` | Add or update members with scores |
| `ZSCORE` | `ZSCORE key member` | Return the score of a member |
| `ZRANK` | `ZRANK key member` | Return the rank (0-based, ascending) |
| `ZRANGE` | `ZRANGE key start stop` | Return members by rank range |
| `ZCARD` | `ZCARD key` | Return the number of members |
| `ZREM` | `ZREM key member [member ...]` | Remove members |

### Pub/Sub

| Command | Syntax | Description |
|---|---|---|
| `SUBSCRIBE` | `SUBSCRIBE channel [channel ...]` | Subscribe to channels |
| `UNSUBSCRIBE` | `UNSUBSCRIBE [channel ...]` | Unsubscribe from channels (all if none given) |
| `PUBLISH` | `PUBLISH channel message` | Publish a message to a channel |

### Transactions

| Command | Syntax | Description |
|---|---|---|
| `MULTI` | `MULTI` | Begin a transaction block |
| `EXEC` | `EXEC` | Execute all queued commands |
| `DISCARD` | `DISCARD` | Discard the queued commands |

### Server & Diagnostics

| Command | Syntax | Description |
|---|---|---|
| `INFO` | `INFO [section]` | Returns server info (`server`, `clients`, `memory`, `stats`, `keyspace`) |
| `CLIENT` | `CLIENT LIST\|SETNAME\|GETNAME` | Client metadata commands |
| `OBJECT` | `OBJECT IDLETIME key \| ENCODING key \| HELP` | Internal object inspection |
| `LRU` | `LRU` | Returns current LRU eviction pool diagnostics |
| `LATENCY` | `LATENCY LATEST \| RESET` | Latency stub (for client compatibility) |
| `BGREWRITEAOF` | `BGREWRITEAOF` | Trigger an AOF dump immediately |
| `COMMAND` | `COMMAND` | Returns OK (for client handshake compatibility) |

---

## Data Types & Internal Encodings

Velox mirrors Redis's dual-encoding strategy: objects start life in a compact encoding and are promoted to a more general one as they grow, balancing memory efficiency with speed.

```
Type          Small encoding         Large encoding
────────────────────────────────────────────────────────
String        int (integer strings)  raw / embstr
Hash          ziplist                hashtable (map[string]string)
List          quicklist of ziplists  (same — quicklist grows dynamically)
Set           intset                 hashtable (map[string]struct{})
ZSet          —                      skiplist + hashtable
```

The type and encoding are packed into a single `uint8` field (`TypeEncoding`) using bit-shifting: the upper 4 bits store the type, and the lower 4 bits store the encoding. Extracting either is a single bit mask operation (`getType`, `getEncoding` in `typeencoding.go`), making every type-check allocation-free.

---

## Core Subsystems — Deep Dive

### 1. Event Loop (`async_tcp.go`)

Velox uses **Linux `epoll`** directly via Go's `syscall` package rather than Go's standard `net` library. This gives us precise control over blocking behavior.

**Startup sequence:**

```
syscall.Socket(AF_INET, SOCK_STREAM|O_NONBLOCK, 0)
syscall.SetNonblock(serverFD, true)
syscall.Bind(serverFD, addr)
syscall.Listen(serverFD, max_clients)
syscall.EpollCreate1(0)          → epollFD
syscall.EpollCtl(EPOLL_CTL_ADD, serverFD, EPOLLIN)
```

**Main loop tick:**

```
UpdateCachedTime()               → update GlobalCachedTime (avoids syscall in hot path)
if cron interval passed → DeleteExpiredKey()
EpollWait(epollFD, events, 100ms)
  ├── if event.Fd == serverFD → Accept() new client, register with epoll
  └── else → readCommands(client) → EvalAndRespond() → respondAsync()
atomic.StoreInt32(&eStatus, WAITING)
```

**Engine state machine** (via `atomic int32`):
- `WAITING` → `BUSY` → `WAITING` (normal operation)
- `WAITING/BUSY` → `SHUTTING_DOWN` (on SIGTERM/SIGINT, only from `WaitForSignal`)

The `CompareAndSwap` from `WAITING` to `BUSY` ensures that once `SHUTTING_DOWN` is set by the signal handler goroutine, the event loop will not re-enter the processing path — a classic lock-free shutdown pattern.

**C10M buffer pool architecture:**

Previously each client pre-allocated a 64 KB read buffer and a 1 KB write buffer inside its struct. At one million idle clients that would require ~65 GB of RAM. Velox solves this by moving the buffers to global `sync.Pool`s:

```go
var ReadBufPool  = sync.Pool{New: func() interface{} { b := make([]byte, 64*1024); return &b }}
var WriteBufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 1024)) }}
```

A buffer is borrowed from the pool only for the instant a client's socket is ready to be read, then immediately returned. An idle client is just a file descriptor integer — effectively zero memory overhead.

---

### 2. RESP Protocol (`resp.go`)

RESP (Redis Serialization Protocol) is the wire format. Every client message and server response is encoded in RESP.

**Wire format quick reference:**

```
Simple String:  +OK\r\n
Error:          -ERR message\r\n
Integer:        :42\r\n
Bulk String:    $6\r\nfoobar\r\n
Nil Bulk:       $-1\r\n
Array:          *3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
```

**Zero-allocation decoder (`DecodeCommands`):**

The original decode path read into `[]interface{}` and re-typed everything at command dispatch time. The new path parses directly into pooled `*RedisCmd` structs, completely bypassing the intermediate interface boxing:

```go
cmd := cmdPool.Get().(*RedisCmd)  // pooled RedisCmd
cmd.Cmd = strings.ToUpper(token)  // first token
cmd.Args = append(cmd.Args, ...)  // subsequent tokens
```

After processing, `FreeCmds()` returns each `*RedisCmd` to `cmdPool`.

**Zero-allocation encoder (`Encode`):**

All encoding uses `append()` into pre-sized byte slices and `strconv.AppendInt` for integers, replacing the former `fmt.Sprintf` calls. For example, encoding an integer response went from ~180 ns (2 allocs) to ~25 ns (0 allocs) — a 7x speedup.

Pre-computed byte constants avoid repeated string-to-`[]byte` conversions:

```go
var RESP_OK     = []byte("+OK\r\n")
var RESP_NIL    = []byte("$-1\r\n")
var RESP_ZERO   = []byte(":0\r\n")
var RESP_ONE    = []byte(":1\r\n")
var RESP_QUEUED = []byte("+QUEUED\r\n")
```

---

### 3. Key-Value Store (`store.go`)

The backing store is a Go `map[string]*Obj` protected by a `sync.RWMutex`:

```go
var store   map[string]*Obj
var expires map[*Obj]uint64    // absolute expiry timestamp in ms, keyed by object pointer
var storeMu sync.RWMutex
```

Expiry is stored separately from the object — the same pattern Redis uses — so scanning for expired keys doesn't require touching every object's fields.

**Object pool (`sync.Pool`):**

`Obj` structs are never heap-allocated on the fast path. `NewObj` obtains one from `objPool`, initializes it, and returns it. When a key is deleted (`delLocked`), the `Obj` is cleared and returned to the pool:

```go
var objPool = sync.Pool{New: func() interface{} { return &Obj{} }}
```

**`Get` — lazy expiry check:**

```go
func Get(k string) *Obj {
    storeMu.RLock()
    v := store[k]
    if v != nil && hasExpired(v) {
        storeMu.RUnlock()
        storeMu.Lock()
        // re-check after acquiring write lock (another goroutine may have beaten us)
        if _, exists := store[k]; exists {
            delLocked(k)
        }
        storeMu.Unlock()
        return nil
    }
    if v != nil { v.LastAccessedAt = getCurrentClock() }
    storeMu.RUnlock()
    return v
}
```

The read-lock-to-write-lock upgrade with re-check is the correct Go pattern: you cannot upgrade a read lock directly, so you release it, acquire the write lock, and re-validate the condition.

**`Put` — eviction on overflow:**

```go
func Put(k string, obj *Obj) {
    storeMu.Lock()
    defer storeMu.Unlock()
    if len(store) >= config.KeysLimit {
        evict()   // runs inside the write lock
    }
    obj.LastAccessedAt = getCurrentClock()
    store[k] = obj
    KeyspaceStat[0]["keys"]++
}
```

---

### 4. Object Model (`object.go`)

Every value in the store is wrapped in an `Obj`:

```go
type Obj struct {
    TypeEncoding   uint8       // upper 4 bits = type, lower 4 bits = encoding
    Value          interface{} // actual data (string, map, *QuickList, *SkipList, etc.)
    LastAccessedAt uint32      // 20-bit LRU clock (seconds since epoch, masked)
}
```

The `LastAccessedAt` field is a **20-bit LRU clock** (`time.Now().Unix() & 0x00FFFFF`). At second resolution this wraps every ~12 days, and wraparound is handled correctly in `getIdleTime`. Redis uses 24 bits (194-day window) but uses C bitfields to pack it alongside type information. Velox uses a dedicated `uint32` field since Go does not have bitfields.

---

### 5. Type Encoding (`typeencoding.go`)

Types and encodings are encoded into a single `uint8`:

```
Bit 7  Bit 6  Bit 5  Bit 4  │  Bit 3  Bit 2  Bit 1  Bit 0
───────── TYPE ──────────────│────────── ENCODING ──────────
0 0 0 0  → STRING            │  0 0 0 0 → RAW
0 0 0 1  → HASH (<<4 = 16)  │  0 0 0 1 → INT
0 0 1 0  → LIST (<<4 = 32)  │  0 0 1 0 → ZIPLIST
0 0 1 1  → SET  (<<4 = 48)  │  0 0 1 1 → HT
0 1 0 0  → ZSET (<<4 = 64)  │  0 1 0 0 → QUICKLIST
                              │  0 1 0 1 → INTSET
                              │  0 1 1 0 → SKIPLIST
                              │  1 0 0 0 → EMBSTR
```

Type is extracted with `(te >> 4) << 4`, encoding with `te & 0b00001111`.

---

### 6. Key Expiry (`expire.go`)

Velox implements both expiry strategies Redis uses:

**Passive expiry (lazy deletion):** checked in `Get()` when the key is accessed. If expired, the key is deleted and `nil` is returned. This is free from a CPU perspective — you pay only when you touch an expired key.

**Active expiry (cron-based):** triggered once per second from the event loop by `DeleteExpiredKey()`. It calls `expireSample()` in a loop:

```go
func expireSample() float32 {
    limit := 20
    expiresCount := 0
    // Go map iteration is randomized — effectively a random sample
    for key, obj := range store {
        if hasExpired(obj) {
            delLocked(key)
            expiresCount++
            limit--
        }
        if limit == 0 { break }
    }
    return float32(expiresCount) / float32(20.0)
}

func DeleteExpiredKey() {
    for {
        frac := expireSample()
        if frac < 0.25 { break }  // stop if <25% of sample was expired
    }
}
```

This is the exact algorithm documented in Redis's `EXPIRE` documentation. The threshold of 25% prevents looping when only a few keys are expired, and the loop continues when mass expiration is happening.

**`hasExpired` uses `GlobalCachedTime`** instead of `time.Now()`, eliminating a syscall on every expiry check:

```go
func hasExpired(obj *Obj) bool {
    exp, ok := expires[obj]
    if !ok { return false }
    return exp <= uint64(GlobalCachedTime)
}
```

---

### 7. Eviction — Approximated LRU

When `len(store) >= config.KeysLimit`, the `Put()` function calls `evict()`, which dispatches to one of three strategies:

#### `simple-first`
Evicts the first key found by iterating the map. O(1) but makes no effort to pick a good victim.

#### `allkeys-random`
Evicts `EvictionRatio × KeysLimit` random keys. Go's map iteration is randomized, so this is effectively random without any extra bookkeeping.

#### `allkeys-lru` (default)

This is Velox's approximated LRU, modeled precisely after Redis's implementation.

**The eviction pool (`evictionpool.go`):**

A max-heap of up to 16 `PoolItem` structs, ordered by idle time (highest idle = least recently used = best candidate to evict). The pool *persists across eviction rounds*, which is the key insight: each sampling round adds more candidates, and the pool converges to holding the genuinely idlest keys across the whole store.

```
Round 1: sample 5 keys → pool = [K3(20s), K1(15s), K7(12s), K2(8s), K9(3s)]
Round 2: sample 5 more → K12 has been idle 25s → K9(3s) kicked out
          pool = [K12(25s), K3(20s), K1(15s), K7(12s), K2(8s)]
Evict: pop K12 → actual eviction
```

**`PushItem` logic:**

- If key is already in pool → skip (O(1) keyset lookup)
- If pool has room → `heap.Push` (O(log n))
- If pool is full → scan the leaves (bottom half of the heap array, which holds the minimum in a max-heap) to find the item with the least idle time; if the new key has more idle time, replace it and call `heap.Fix` (O(log n))

This gives >90% of true LRU accuracy with only 5 random samples per round — the same accuracy Redis achieves.

---

### 8. AOF Persistence (`aof.go`)

Velox persists data using an **Append-Only File** in a "dump on shutdown / replay on startup" model (equivalent to Redis's `BGREWRITEAOF` with no continuous append).

**`DumpAllAOF()`** (called on `BGREWRITEAOF` and on graceful shutdown):

Iterates the store under a read lock, serializes each key using its type-appropriate command (`SET`, `HSET`, `RPUSH`, `SADD`), encodes the command in RESP format, and writes it to `./velox.aof` via a buffered writer. The file is truncated and rewritten from scratch.

**`ReplayAOF()`** (called on startup via `InitStore()`):

Reads the AOF file, passes the entire content through `DecodeCommands`, and replays each command by calling the corresponding `eval*` function directly (bypassing the network layer). This restores the full state from the last dump.

**Supported data types in AOF:**

| Type | Dump format |
|---|---|
| String | `SET key value` |
| Hash (ziplist) | `HSET key f1 v1 f2 v2 ...` |
| Hash (hashtable) | `HSET key f1 v1 f2 v2 ...` |
| List | `RPUSH key e1 e2 e3 ...` |
| Set (intset) | `SADD key m1 m2 ...` |
| Set (hashtable) | `SADD key m1 m2 ...` |

> **Note:** Expiry information is not persisted in the current AOF implementation. Keys restored on startup have no TTL.

---

### 9. Pub/Sub (`pubsub.go`)

The pub/sub system uses a single map:

```go
var channels = make(map[string]map[*Client]struct{})
// channel name → set of subscribed client pointers
```

Because the event loop is strictly single-threaded, this map needs no mutex.

**SUBSCRIBE:** Adds `*Client` to the channel's subscriber set and writes a three-element RESP array back (`["subscribe", channelName, count]`).

**PUBLISH:** Iterates the channel's subscriber set and calls `client.Write(encodedPayload)` for each, counting successful writes. Returns the receiver count.

**UNSUBSCRIBE:** Removes the client from listed channels (or all channels if none given). Cleans up empty channels.

**Disconnect cleanup:** `RemoveClientFromPubSub(c)` is called in the event loop whenever a client's socket read returns an error, ensuring dangling pointers are never left in channel maps.

---

### 10. Transactions (`comm.go`)

Velox supports Redis-style optimistic transactions via MULTI/EXEC/DISCARD:

```go
type Client struct {
    Fd      int
    enqueue RedisCmds  // queued commands
    isTxn   bool       // transaction in progress?
    ...
}
```

**`MULTI`:** Sets `c.isTxn = true`. Returns `+OK`.

**While `isTxn == true`:** Every incoming command is checked in `EvalAndRespond`. If the command is not `EXEC` or `DISCARD`, it is appended to `c.enqueue` and `+QUEUED` is returned to the client.

**`EXEC`:** Calls `TxnExec()`, which writes the RESP array header, then executes all queued commands in order by calling `executeCommand()` for each. After execution, `isTxn` is reset and `enqueue` is cleared.

**`DISCARD`:** Clears `enqueue` and resets `isTxn`. Returns `+OK`.

---

### 11. Data Structures

#### ZipList (`ziplist.go`)

A flat `[]string` slice used as a compact sequential store. For hashes, entries are stored as interleaved field-value pairs: `[field1, value1, field2, value2, ...]`. All operations are O(n) scan, which is acceptable for small collections (< 128 entries) due to cache locality.

Operations: `HashGet`, `HashSet`, `HashDel`, `HashExists`, `HashEntries`, `HashKeys`, `HashValues`, `HashGetAll`, `LPush`, `RPush`, `LPop`, `RPop`, `Index`, `Range`, `Entries`.

#### QuickList (`quicklist.go`)

A doubly-linked list of `QuickListNode` structs, where each node contains a `*ZipList`. This is the exact encoding Redis uses for all List values.

- **LPUSH/RPUSH:** If the head/tail node's ziplist is at capacity (`ListMaxZiplistSize`), a new node is prepended/appended. Otherwise the value goes into the existing ziplist. O(1) amortized.
- **LPOP/RPOP:** Delegates to the head/tail ziplist. If the node becomes empty, it's unlinked and the node count is decremented. O(1).
- **LINDEX:** Walks the node chain skipping whole nodes, then indexes into the ziplist. O(n) worst case but skips whole ziplists.
- **LRANGE:** Coalesces ranges across multiple nodes efficiently.

#### SkipList (`skiplist.go`)

A probabilistic multi-level linked list used as the primary storage for Sorted Sets (ZSets). Each node has up to 32 levels, with each level having a `forward` pointer and a `span` (the number of level-0 nodes the forward pointer skips over).

- **Insert:** O(log N) expected, traces the insertion point at each level and updates forward pointers and spans.
- **Delete:** O(log N) expected.
- **GetRank:** O(log N) — accumulates span values to compute the 1-based rank.
- **GetNodeByRank:** O(log N) — traverses using span counters.
- **Range:** O(log N + M) where M is the number of returned elements.

A companion `map[string]float64` (hashtable) is maintained alongside the skiplist in `eval_zset.go` to allow O(1) score lookups (`ZSCORE`) without traversing the list.

Node ordering: ascending by score, then lexicographically by member when scores are equal — matching Redis's ZSet semantics exactly.

#### IntSet (`intset.go`)

A sorted `[]int64` slice. Used when all members of a Set are parseable as integers and the count is below `SetMaxIntsetEntries`.

- **Add:** Binary search to find insertion point, then `copy` to shift elements. O(log n) search + O(n) insert.
- **Remove:** Binary search then `copy` to close the gap. O(log n) + O(n).
- **Contains:** Binary search. O(log n).

When a non-integer is added or the count limit is exceeded, `eval_set.go` promotes the encoding from IntSet to a Go `map[string]struct{}`.

---

### 12. Metrics (`metrics.go`)

```go
type ServerMetrics struct {
    StartTime           time.Time
    CommandsProcessed   uint64   // atomic
    ConnectionsReceived uint64   // atomic
    NetworkBytesIn      uint64   // atomic (reserved for future use)
    NetworkBytesOut     uint64   // atomic (reserved for future use)
}
```

All counters are incremented via `atomic.AddUint64` so they can be read safely by background goroutines (e.g., a hypothetical metrics exporter) without holding any lock.

Exposed via `INFO stats`:
- `total_connections_received`
- `total_commands_processed`
- `instantaneous_ops_per_sec` (lifetime average)

---

## Performance Engineering

Every hot-path decision in Velox has a documented rationale. The table below summarises the most impactful changes:

| Change | Before | After | Gain |
|---|---|---|---|
| Read buffer | `make([]byte, 64KB)` per request | `sync.Pool` borrow/return | ~15–25% throughput |
| Write buffer | `bytes.NewBuffer()` per request | `sync.Pool` borrow/return | ~0 allocs/request at steady state |
| RESP integer encode | `fmt.Sprintf(":%d\r\n", v)` | `strconv.AppendInt` | ~7x faster, 0 allocs |
| RESP string encode | `fmt.Sprintf("$%d\r\n%s\r\n", len, v)` | `append` + `strconv.AppendInt` | ~7x faster |
| Command decode | `[]interface{}` + re-type | Direct `*RedisCmd` from pool | ~3x fewer allocations |
| Client struct | `ReadBuf [64KB]` per idle client | No buffers in struct; pool only | 0 RAM per idle client |
| Time syscalls | `time.Now()` every expiry check | `GlobalCachedTime` (1/tick) | Eliminates N syscalls per tick |
| TxnExec header | `fmt.Sprintf("*%d\r\n", n)` | `buf.WriteByte + strconv.AppendInt` | 0 allocs |
| Obj allocation | `&Obj{...}` on every `NewObj` | `sync.Pool` with field reset | 0 allocs on fast path |

---

## Benchmark Tool (`velox-benchmark`)

A standalone concurrent benchmark tool that simulates N parallel clients sending pre-formatted RESP commands as fast as possible.

### Build & Run

```bash
go build -o bin/velox-benchmark ./velox-benchmark

./bin/velox-benchmark \
  -h 127.0.0.1 \   # server host
  -p 7379 \        # server port
  -c 50 \          # number of parallel client goroutines
  -n 1000000 \     # total requests per test
  -d 3 \           # payload size in bytes (for SET)
  -q 50 \          # pipeline depth (send 50 commands per write)
  -t PING,SET,GET  # comma-separated list of tests
```

### Supported tests

`PING`, `SET`, `GET`, `INCR`

### How it works

1. Commands are pre-encoded in RESP format once before the test loop begins — no format overhead during measurement.
2. For pipelining (`-q N`), the pipelined payload (N copies of the command) is built once and reused.
3. Each client goroutine opens its own TCP connection, sends batches, and reads responses using a `bufio.Reader`.
4. Bulk string responses (`$...`) require reading two lines; the tool handles this correctly.
5. Completed requests and errors are counted via `atomic.AddUint64` from all goroutines.
6. Throughput = `completedReqs / elapsed_seconds`.

### Example output

```
====== Velox Benchmark ======
Server: 127.0.0.1:7379
Clients: 50
Requests per test: 1000000
Payload size: 3 bytes

Running PING test...
  1000000 requests completed in 0.26 seconds
  50 parallel clients
  3797336.57 requests per second

Running SET test...
  1000000 requests completed in 0.58 seconds
  50 parallel clients
  1734256.37 requests per second

Running GET test...
  1000000 requests completed in 0.37 seconds
  50 parallel clients
  2675978.75 requests per second
```

---

## CLI Client (`velox-cli`)

An interactive REPL that converts plain-text user input into RESP and prints the raw server response.

```bash
go run ./velox-cli
```

```
velox> SET city Mumbai
+OK
velox> GET city
$6
Mumbai
velox> EXPIRE city 10
:1
velox> TTL city
:9
velox> DEL city
:1
velox> GET city
$-1
velox> SUBSCRIBE news
...waiting for messages...
```

> The CLI does minimal response parsing — it prints the raw RESP bytes. For a richer interactive experience, use `redis-cli -p 7379`.

---

## Running Tests

```bash
# Unit tests (RESP encode/decode, config, server lifecycle)
go test ./...

# Integration tests (starts the server internally, sends real commands)
go test -v -run TestIntegration

# Benchmark sub-package test
go test ./velox-benchmark/...

# Race detector (recommended during development)
go test -race ./...
```

---

## Known Limitations & TODOs

| Area | Status / Note |
|---|---|
| Persistence | AOF dumps only on shutdown / `BGREWRITEAOF`. No continuous append. TTLs not persisted. |
| Eviction `KeysLimit` | Default is 100 (good for testing, too low for production). Raise in `config.go`. |
| AOF ZSet | ZSet data (`ZADD`) is not currently dumped to the AOF file. |
| Pub/Sub thread safety | Safe because the event loop is single-threaded; multi-threading would require a mutex on `channels`. |
| `OBJECT IDLETIME` | Uses a 20-bit clock (wraps ~12 days). Redis uses 24 bits. |
| `INFO memory` | Returns an estimate based on key count × 128 bytes. A real implementation would use `runtime.MemStats`. |
| `CLIENT LIST` | Returns a static placeholder string. |
| Inline RESP | `DecodeCommands` requires RESP array format (`*N\r\n...`). Plain inline commands (`PING\r\n`) are not supported on the fast path. |
| Windows / macOS | The async server uses Linux-only `epoll`. Run on Linux or inside WSL2. The sync server (`sync_tcp.go`) works on all platforms. |
| `BGREWRITEAOF` | Currently synchronous. The TODO comment notes it should fork a child process. |
| `SetMaxIntsetEntries` | Default 512. Promotion to hashtable not yet wired for the size limit (only non-integer addition triggers promotion today). |

---

## License
MIT — see `LICENSE` for details.
