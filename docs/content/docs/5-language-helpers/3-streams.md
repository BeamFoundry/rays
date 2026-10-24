---
title: "Streams"
weight: 3
---

# Java 8-Style Streams

Processing sequences of data in Go often requires verbose `for` loops, manual slice allocation, and custom grouping
logic. While Go introduced native range-over-function iterators (`iter.Seq`), chaining multiple transformations across
different generic types remained cumbersome.

With **Go 1.27** introducing generic type parameters on struct methods, Rays provides `pkg/util/stream`: an idiomatic,
lazy, thread-safe, and expressive Stream processing framework inspired by Java 8 Streams.

Similar to `pkg/lang` (e.g. `OptionalOf`, `OptionalEmpty`), `pkg/util/stream` is designed to be cleanly **dot-imported**
(`import . "github.com/BeamFoundry/rays/pkg/util/stream"`). Constructor functions are prefixed with `Stream` (such as `StreamOf`,
`StreamOfSlice`, `StreamEmpty`) so they can be invoked cleanly without cluttering or colliding with common identifier names.

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

// Fluent pipeline with Go 1.27 struct method type parameters
topActiveNames := StreamOfSlice(users).
    Filter(func(u User) bool { return u.Active }).
    SortedByDesc(func(u User) int { return u.Score }).
    Limit(5).
    Map(func(u User) string { return u.Name }).
    ToSlice()
```

## Key Capabilities

* **Fluent Method Type Inference**: Call `.Map(...)`, `.FlatMap(...)`, `.Collect(...)`, `.SortedBy(...)`, or
  `.ReduceWithInitial(...)` directly on the stream without standalone functions.
* **Lazy Evaluation**: Backed by Go's native `iter.Seq[T]`. Stream pipelines defer computation until a terminal
  operation is called. Short-circuiting operations (`Limit`, `TakeWhile`, `FindFirst`, `AnyMatch`) stop evaluating
  upstream elements as soon as their termination condition is met.
* **Single-Use Semantics**: Following Java 8 Stream semantics, a stream can only be consumed once. Attempting to
  traverse or reuse an already operated or closed stream raises a framework `IllegalStateError`.
* **Parallel Execution**: Switch any pipeline to concurrent mode with `.Parallel()`. Terminal operations like `ForEach`,
  `Collect` (with worker map-reduce chunking), `AnyMatch`, and `FindAny` execute across goroutines.
* **Standard Collectors**: Rich collection abstractions including `ToList`, `ToSet`, `Joining`, `GroupingBy`,
  `PartitioningBy`, `ToMap`, `Counting`, `SummingInt`, `SummarizingInt`, and downstream collectors.
* **`lang.Optional` Integration**: Operations that may produce an empty result (`FindFirst`, `FindAny`, `Min`, `Max`,
  `Reduce`) return Rays' type-safe `lang.Optional[T]`.

---

## 1. Creating Streams

Streams can be instantiated from values, slices, channels, functions, ranges, or existing iterators:

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

// Empty stream
s0 := StreamEmpty[string]()

// From explicit values or slices
s1 := StreamOf(1, 2, 3, 4, 5)
s2 := StreamOfSlice([]string{"apple", "banana", "cherry"})

// From an existing Go iterator (iter.Seq[T])
s3 := StreamOfSeq(myIterSeq)

// From a collection (core.List, core.Set, ArrayList, HashSet)
sCollection := StreamOfCollection(myList)
// Or directly via collection's Stream() method: myList.Stream()

// From a Go channel (reads until channel is closed)
s4 := StreamOfChan(jobChan)

// Numeric ranges
s5 := StreamRange(0, 10)         // 0 to 9 (exclusive)
s6 := StreamRangeClosed(1, 10)   // 1 to 10 (inclusive)

// Infinite lazy generators
s7 := StreamGenerate(rand.Float64)
s8 := StreamIterate(1, func(x int) int { return x * 2 })
s9 := StreamIterateWithPredicate(1, func(x int) bool { return x <= 100 }, func(x int) int { return x + 1 })

// Concatenation of two streams
s10 := StreamConcat(s1, s5)
```

---

## 2. Intermediate Operations

Intermediate operations return a new `Stream` and are **lazy**. Elements are pulled through the pipeline only when a
terminal operation triggers evaluation.

| Operation | Description |
|---|---|
| `Filter(predicate func(T) bool)` | Retains only elements that satisfy the predicate. |
| `Map[R](mapper func(T) R)` | Transforms elements from type `T` to type `R`. |
| `FlatMap[R](mapper func(T) Stream[R])` | Flattens mapped sub-streams into a single continuous stream. |
| `FlatMapSlice[R](mapper func(T) []R)` | Flattens mapped slices into a single stream. |
| `FlatMapSeq[R](mapper func(T) iter.Seq[R])` | Flattens mapped `iter.Seq` iterators. |
| `Distinct()` | Deduplicates elements based on equality. |
| `DistinctBy[K comparable](keySelector func(T) K)` | Deduplicates elements using an extracted key. |
| `Sorted(comparator...)` | Sorts elements using natural ordering or a custom comparator (`func(a, b T) int`). |
| `SortedFunc(less func(a, b T) bool)` | Sorts elements using a custom `less` function. |
| `SortedBy[K Ordered](keySelector func(T) K)` | Sorts elements in ascending order by an extracted ordered key. |
| `SortedByDesc[K Ordered](keySelector func(T) K)` | Sorts elements in descending order by an extracted ordered key. |
| `Limit(maxSize int64)` | Truncates the stream to at most `maxSize` elements (short-circuits upstream). |
| `Skip(n int64)` | Discards the first `n` elements. |
| `TakeWhile(predicate func(T) bool)` | Takes elements as long as the predicate returns true, then terminates. |
| `DropWhile(predicate func(T) bool)` | Drops elements until the predicate first returns false, then yields the rest. |
| `Peek(action func(T))` | Performs an action on each element as it passes through (useful for debugging). |

### Example: Mapping and Sorting

```go
type Employee struct {
    Name       string
    Department string
    Salary     int
}

// Fluent transformation from Employee structs to formatted strings
salaries := StreamOfSlice(employees).
    Filter(func(e Employee) bool { return e.Department == "Engineering" }).
    SortedByDesc(func(e Employee) int { return e.Salary }).
    Map(func(e Employee) string {
        return fmt.Sprintf("%s: $%d", e.Name, e.Salary)
    }).
    ToSlice()
```

---

## 3. Terminal Operations

Terminal operations trigger the stream pipeline and consume the stream.

### Slice & Iterator Conversion

* **`ToSlice() []T`**: Consumes the stream and returns all elements as a slice in encounter order.
* **`Seq() iter.Seq[T]`**: Yields elements as a Go 1.23+ iterator sequence, enabling direct `for ... := range stream.Seq()` loops.
* **`ForEach(action func(T))`**: Invokes the action on each element (concurrently if parallel).
* **`ForEachOrdered(action func(T))`**: Invokes the action in strict encounter order.

### Matching & Search

All match operations short-circuit as soon as the outcome is determined:

```go
hasEven := StreamOf(1, 2, 3, 4, 5).AnyMatch(func(x int) bool { return x%2 == 0 })    // true
allPositive := StreamOf(1, 2, 3, 4, 5).AllMatch(func(x int) bool { return x > 0 })   // true
noneNegative := StreamOf(1, 2, 3, 4, 5).NoneMatch(func(x int) bool { return x < 0 }) // true

first := StreamOf("apple", "banana").FindFirst() // returns lang.OptionalOf("apple")
any := StreamOf("apple", "banana").FindAny()     // returns lang.Optional[string]
```

### Reductions

* **`Reduce(accumulator func(a, b T) T) lang.Optional[T]`**: Folds elements into an optional result.
* **`ReduceWithIdentity(identity T, accumulator func(a, b T) T) T`**: Folds elements starting from `identity`.
* **`ReduceWithInitial[R](identity R, accumulator func(R, T) R) R`**: Folds elements of type `T` into an accumulator of type `R`.

```go
// Reducing ints into a formatted string
joined := StreamOf(1, 2, 3).ReduceWithInitial("Values:", func(acc string, v int) string {
    return fmt.Sprintf("%s %d", acc, v)
}) // "Values: 1 2 3"
```

### Min & Max

```go
// Natural ordering
minVal := StreamOf(5, 2, 8, 1).Min() // OptionalOf(1)
maxVal := StreamOf(5, 2, 8, 1).Max() // OptionalOf(8)

// By key selector
youngest := StreamOfSlice(employees).MinBy(func(e Employee) int { return e.Age })
```

---

## 4. Collectors (`stream.Collect`)

Rays provides a robust `Collector[T, A, R]` abstraction modeled after `java.util.stream.Collectors`.
When dot-importing `pkg/util/stream`, collectors like `ToList`, `ToSet`, and `GroupingBy` can be passed directly:

### Standard Collectors

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

// Collect to core.List[T], core.Set[T], or native slice
list  := StreamOf(1, 2, 3).Collect(ToList[int]())         // returns core.List[int]
set   := StreamOf("a", "b", "c").Collect(ToSet[string]()) // returns core.Set[string]
slice := StreamOf(1, 2, 3).Collect(ToSlice[int]())        // returns native []int

// Joining strings
joined := StreamOf("A", "B", "C").Collect(
    Joining(", ", "[", "]"),
) // "[A, B, C]"

// Grouping by key into slices
byDept := StreamOfSlice(employees).Collect(
    GroupingBySlice(func(e Employee) string { return e.Department }),
) // map[string][]Employee

// Grouping with downstream reduction
countByDept := StreamOfSlice(employees).Collect(
    GroupingBy(
        func(e Employee) string { return e.Department },
        Counting[Employee](),
    ),
) // map[string]int64

// Partitioning by boolean predicate
passedFailed := StreamOfSlice(students).Collect(
    PartitioningBySlice(func(s Student) bool { return s.Grade >= 60 }),
) // map[bool][]Student

// Summary Statistics (Count, Sum, Min, Max, Average)
stats := StreamOfSlice(employees).Collect(
    SummarizingInt(func(e Employee) int { return e.Salary }),
)
fmt.Printf("Average salary: %.2f (Min: %d, Max: %d)\n",
    stats.Average(), stats.Min, stats.Max)

// Building maps with collision resolution
empMap := StreamOfSlice(employees).Collect(
    ToMap(
        func(e Employee) int { return e.ID },
        func(e Employee) string { return e.Name },
        func(existing, newVal string) string { return existing }, // optional merge rule
    ),
)
```

---

## 5. Parallel Processing

To execute a pipeline concurrently across multiple goroutines, simply call `.Parallel()` on the stream:

```go
total := StreamRangeClosed(1, 1_000_000).
    Parallel().
    Filter(func(n int) bool { return isPrime(n) }).
    Collect(Counting[int]())
```

### How Parallel Streams Work
1. **Goroutine Scaling**: Concurrency is automatically scaled to `runtime.GOMAXPROCS`.
2. **Chunked Map-Reduce**: During `Collect`, stream chunks are distributed across worker goroutines. Each worker maintains its own isolated accumulator created with `Collector.Supplier`.
3. **Combining Partial Results**: Once workers finish their batches, partial accumulators are merged using `Collector.Combiner`, and the final result is produced via `Collector.Finisher` (encounter order is not preserved).
4. **Short-Circuiting**: In operations such as `AnyMatch`, `AllMatch`, and `FindAny`, an atomic trigger stops all workers as soon as the result is decided.

---

## 6. Resource Lifecycle & AutoClose

Streams support registering cleanup hooks via `.OnClose(...)` and closing resources using `.Close()`:

```go
s := StreamOfChan(eventChannel).
    OnClose(func() {
        log.Println("Stream pipeline finished; releasing resources")
    })

// Consuming the stream automatically closes it upon completion
s.ForEach(processEvent)
```
