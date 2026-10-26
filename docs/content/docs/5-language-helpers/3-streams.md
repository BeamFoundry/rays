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
(`import . "github.com/BeamFoundry/rays/pkg/util/stream"`). Streams are constructed directly via static-like constructor methods on `Stream[T]` (e.g. `Stream[T]{}.From(...)`).

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

// Fluent pipeline with Go 1.27 struct method type parameters
topActiveNames := Stream[User]{}.FromSlice(users).
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
  `Collect` (with batch accumulation and encounter-order combining), `AnyMatch`, and `FindAny` execute across goroutines.
* **Standard Collectors**: Rich collection abstractions provided via `Collectors` methods including `ToList`, `ToSet`, `ToSlice`, `ToMap`, `Joining`, `GroupingBy`, `PartitioningBy`, `Counting`, `SummingInt`, `SummarizingInt`, and downstream collectors.
* **`lang.Optional` Integration**: Operations that may produce an empty result (`FindFirst`, `FindAny`, `Min`, `Max`,
  `Reduce`) return Rays' type-safe `lang.Optional[T]`.

---

## 1. Creating Streams

Streams can be instantiated from values, slices, channels, functions, ranges, or existing iterators:

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

// Empty stream
s0 := Stream[string]{}.FromEmpty()

// From explicit values or slices
s1 := Stream[int]{}.From(1, 2, 3, 4, 5)
s2 := Stream[string]{}.FromSlice([]string{"apple", "banana", "cherry"})

// From an existing Go iterator (iter.Seq[T])
s3 := Stream[T]{}.FromSeq(myIterSeq)

// From a collection (core.List, core.Set, ArrayList, HashSet)
sCollection := Stream[T]{}.FromCollection(myList)
// Or directly via collection's Stream() method: myList.Stream()

// From a Go channel (reads until channel is closed)
s4 := Stream[Job]{}.FromChan(jobChan)

// Numeric ranges
s5 := Stream[int]{}.FromRange(0, 10)         // 0 to 9 (exclusive)
s6 := Stream[int]{}.FromRangeClosed(1, 10)   // 1 to 10 (inclusive)

// Infinite lazy generators
s7 := Stream[float64]{}.FromGenerate(rand.Float64)
s8 := Stream[int]{}.FromIterate(1, func(x int) int { return x * 2 })
s9 := Stream[int]{}.FromIterateWithPredicate(1, func(x int) bool { return x <= 100 }, func(x int) int { return x + 1 })

// Concatenation of two streams
s10 := Stream[int]{}.FromConcat(s1, s5)
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
salaries := Stream[Employee]{}.FromSlice(employees).
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
hasEven := Stream[int]{}.From(1, 2, 3, 4, 5).AnyMatch(func(x int) bool { return x%2 == 0 })    // true
allPositive := Stream[int]{}.From(1, 2, 3, 4, 5).AllMatch(func(x int) bool { return x > 0 })   // true
noneNegative := Stream[int]{}.From(1, 2, 3, 4, 5).NoneMatch(func(x int) bool { return x < 0 }) // true

first := Stream[string]{}.From("apple", "banana").FindFirst() // returns lang.OptionalOf("apple")
any := Stream[string]{}.From("apple", "banana").FindAny()     // returns lang.Optional[string]
```

### Reductions

* **`Reduce(accumulator func(a, b T) T) lang.Optional[T]`**: Folds elements into an optional result.
* **`ReduceWithIdentity(identity T, accumulator func(a, b T) T) T`**: Folds elements starting from `identity`.
* **`ReduceWithInitial[R](identity R, accumulator func(R, T) R) R`**: Folds elements of type `T` into an accumulator of type `R`.

```go
// Reducing ints into a formatted string
joined := Stream[int]{}.From(1, 2, 3).ReduceWithInitial("Values:", func(acc string, v int) string {
    return fmt.Sprintf("%s %d", acc, v)
}) // "Values: 1 2 3"
```

### Min & Max

```go
// Natural ordering
minVal := Stream[int]{}.From(5, 2, 8, 1).Min() // OptionalOf(1)
maxVal := Stream[int]{}.From(5, 2, 8, 1).Max() // OptionalOf(8)

// By key selector
youngest := Stream[Employee]{}.FromSlice(employees).MinBy(func(e Employee) int { return e.Age })
```

---

## 4. Collectors (`stream.Collect`)

Rays provides a robust `Collector[T, A, R]` abstraction modeled after `java.util.stream.Collectors`.
A collector encapsulates a mutable reduction operation that folds input elements of type `T` into a mutable
intermediate accumulator of type `A`, and finally finishes the accumulator into a result of type `R`.

```go
type Collector[T, A, R any] struct {
    Supplier    func() A        // Allocates a fresh accumulator
    Accumulator func(A, T) A    // Folds a new item into the accumulator
    Combiner    func(A, A) A    // Merges two accumulators (used in parallel streams)
    Finisher    func(A) R       // Transforms the intermediate accumulator into the final result
}
```

Factory methods are instantiated through the `Collectors` struct (e.g. `Collectors{}.ToList[T]()` or `var c Collectors; c.ToList[T]()`).

### Complete Collectors Reference

All 23 built-in collectors provided by `Collectors` are summarized below:

| Collector | Signature | Intermediate `A` | Result `R` | Description |
|---|---|---|---|---|
| `ToList` | `ToList[T]()` | `List[T]` | `core.List[T]` | Accumulates elements into a Rays `core.List[T]`. |
| `ToSlice` | `ToSlice[T]()` | `[]T` | `[]T` | Accumulates elements into a standard Go slice `[]T`. |
| `ToSet` | `ToSet[T comparable]()` | `Set[T]` | `core.Set[T]` | Accumulates unique elements into a Rays `core.Set[T]`. |
| `ToSetSlice` | `ToSetSlice[T comparable]()` | `map[T]struct{}` | `[]T` | Collects unique elements into a standard Go slice `[]T`. |
| `ToMap` | `ToMap[T, K, V](keyMapper, valMapper, mergeRule...)` | `map[K]V` | `map[K]V` | Accumulates elements into a map with optional collision resolution. |
| `Joining` | `Joining(delimiter, prefixAndSuffix...)` | `*joinAcc` | `string` | Joins string elements with a delimiter and optional prefix/suffix. |
| `Counting` | `Counting[T]()` | `*int64` | `int64` | Counts the total number of stream elements. |
| `SummingInt` | `SummingInt[T](mapper)` | `*int64` | `int64` | Sums mapped `int` values as an `int64`. |
| `SummingFloat64` | `SummingFloat64[T](mapper)` | `*float64` | `float64` | Sums mapped `float64` values. |
| `AveragingInt` | `AveragingInt[T](mapper)` | `*avgAcc` | `float64` | Computes the arithmetic average of mapped `int` values. |
| `AveragingFloat64` | `AveragingFloat64[T](mapper)` | `*avgAcc` | `float64` | Computes the arithmetic average of mapped `float64` values. |
| `SummarizingInt` | `SummarizingInt[T](mapper)` | `*IntSummaryStatistics` | `IntSummaryStatistics` | Computes count, sum, min, max, and average for integers. |
| `SummarizingFloat64` | `SummarizingFloat64[T](mapper)` | `*Float64SummaryStatistics` | `Float64SummaryStatistics` | Computes count, sum, min, max, and average for floats. |
| `MinBy` | `MinBy[T](comparator)` | `*Optional[T]` | `lang.Optional[T]` | Finds the minimum element using a three-way comparator (`func(a, b T) int`). |
| `MaxBy` | `MaxBy[T](comparator)` | `*Optional[T]` | `lang.Optional[T]` | Finds the maximum element using a three-way comparator (`func(a, b T) int`). |
| `Reducing` | `Reducing[T](identity, op)` | `*T` | `T` | Reduces elements under an associative operator and identity value. |
| `GroupingBySlice` | `GroupingBySlice[T, K](classifier)` | `map[K][]T` | `map[K][]T` | Groups elements by key into standard slices. |
| `GroupingBy` | `GroupingBy[T, K, A, D](classifier, downstream)` | `map[K]A` | `map[K]D` | Groups elements by key and applies a downstream reduction collector. |
| `PartitioningBySlice` | `PartitioningBySlice[T](predicate)` | `map[bool][]T` | `map[bool][]T` | Partitions elements into `true` and `false` slices. |
| `PartitioningBy` | `PartitioningBy[T, A, D](predicate, downstream)` | `map[bool]A` | `map[bool]D` | Partitions elements by predicate with a downstream collector. |
| `Mapping` | `Mapping[T, U, A, R](mapper, downstream)` | `A` | `R` | Adapts a collector of type `U` to accept elements of type `T`. |
| `Filtering` | `Filtering[T, A, R](predicate, downstream)` | `A` | `R` | Adapts a collector to only accept elements satisfying a predicate. |
| `FlatMapping` | `FlatMapping[T, U, A, R](mapper, downstream)` | `A` | `R` | Adapts a collector by flattening mapped slices `[]U` into downstream. |

---

### Categorized Usage Guide

```go
import . "github.com/BeamFoundry/rays/pkg/util/stream"

var c Collectors
```

#### 4.1 Collections & Slices
Collect stream items into Rays collections (`core.List[T]`, `core.Set[T]`), native slices (`[]T`), or maps:

```go
// core.List[T]
list := Stream[int]{}.From(1, 2, 3).Collect(c.ToList[int]())

// Native slice []T
slice := Stream[int]{}.From(1, 2, 3).Collect(c.ToSlice[int]())

// core.Set[T] (deduplicated)
set := Stream[string]{}.From("a", "b", "a", "c").Collect(c.ToSet[string]())

// Unique slice []T
uniqueSlice := Stream[int]{}.From(1, 2, 2, 3, 1).Collect(c.ToSetSlice[int]())

// Key-Value map with collision resolution
userMap := Stream[User]{}.FromSlice(users).Collect(
    c.ToMap(
        func(u User) string { return u.ID },
        func(u User) string { return u.Name },
        func(existing, newVal string) string { return newVal }, // keep latest on collision
    ),
)
```

#### 4.2 String Joining
Join string streams with configurable delimiters, optional prefixes, and suffixes:

```go
// Delimiter only: "apple, banana, cherry"
csv := Stream[string]{}.From("apple", "banana", "cherry").Collect(
    c.Joining(", "),
)

// Delimiter with prefix and suffix: "[A, B, C]"
bracketed := Stream[string]{}.From("A", "B", "C").Collect(
    c.Joining(", ", "[", "]"),
)
```

#### 4.3 Counting, Numeric Reductions & Averages
Efficient single-pass aggregations over numeric properties:

```go
// Counting elements
count := Stream[string]{}.From("x", "y", "z").Collect(c.Counting[string]()) // 3

// Summing ints and floats
totalSalary := Stream[Employee]{}.FromSlice(employees).Collect(
    c.SummingInt(func(e Employee) int { return e.Salary }),
)
totalWeight := Stream[Package]{}.FromSlice(packages).Collect(
    c.SummingFloat64(func(p Package) float64 { return p.Weight }),
)

// Calculating averages
avgSalary := Stream[Employee]{}.FromSlice(employees).Collect(
    c.AveragingInt(func(e Employee) int { return e.Salary }), // returns float64
)
avgWeight := Stream[Package]{}.FromSlice(packages).Collect(
    c.AveragingFloat64(func(p Package) float64 { return p.Weight }),
)
```

#### 4.4 Summary Statistics
Collect comprehensive statistical metrics in a single pass without re-traversing the stream:

```go
stats := Stream[Employee]{}.FromSlice(employees).Collect(
    c.SummarizingInt(func(e Employee) int { return e.Salary }),
)

fmt.Println("Count:  ", stats.Count)
fmt.Println("Sum:    ", stats.Sum)
fmt.Println("Min:    ", stats.Min)
fmt.Println("Max:    ", stats.Max)
fmt.Println("Average:", stats.Average()) // (float64)
```

#### 4.5 Extrema & Folding
Find minimum or maximum items using custom comparators, or fold elements with `Reducing`:

```go
// Find employee with lowest salary using three-way comparator (returns lang.Optional[Employee])
lowestPaid := Stream[Employee]{}.FromSlice(employees).Collect(
    c.MinBy(func(a, b Employee) int { return a.Salary - b.Salary }),
)

// Associative reduction
product := Stream[int]{}.From(2, 3, 4).Collect(
    c.Reducing(1, func(a, b int) int { return a * b }),
) // 24
```

#### 4.6 Grouping & Partitioning
Group elements by an extracted key, or partition into a two-value boolean map (`map[bool]...`):

```go
// Grouping into slices: map[string][]Employee
byDept := Stream[Employee]{}.FromSlice(employees).Collect(
    c.GroupingBySlice(func(e Employee) string { return e.Department }),
)

// Grouping with downstream counting: map[string]int64
headcountByDept := Stream[Employee]{}.FromSlice(employees).Collect(
    c.GroupingBy(
        func(e Employee) string { return e.Department },
        c.Counting[Employee](),
    ),
)

// Partitioning into true/false slices: map[bool][]Student
passedFailed := Stream[Student]{}.FromSlice(students).Collect(
    c.PartitioningBySlice(func(s Student) bool { return s.Score >= 60 }),
)

// Partitioning with downstream aggregation: map[bool]int64
countPassedFailed := Stream[Student]{}.FromSlice(students).Collect(
    c.PartitioningBy(
        func(s Student) bool { return s.Score >= 60 },
        c.Counting[Student](),
    ),
)
```

#### 4.7 Downstream Transforming Adapters
Compose downstream collectors by mapping, filtering, or flattening elements before they reach the target collector:

```go
// Mapping downstream: group departments by set of employee names
namesByDept := Stream[Employee]{}.FromSlice(employees).Collect(
    c.GroupingBy(
        func(e Employee) string { return e.Department },
        c.Mapping(
            func(e Employee) string { return e.Name },
            c.ToSet[string](),
        ),
    ),
)

// Filtering downstream: group departments, but only count senior staff
seniorsByDept := Stream[Employee]{}.FromSlice(employees).Collect(
    c.GroupingBy(
        func(e Employee) string { return e.Department },
        c.Filtering(
            func(e Employee) bool { return e.YearsOfService >= 5 },
            c.Counting[Employee](),
        ),
    ),
)

// FlatMapping downstream: extract all distinct skills per department
skillsByDept := Stream[Employee]{}.FromSlice(employees).Collect(
    c.GroupingBy(
        func(e Employee) string { return e.Department },
        c.FlatMapping(
            func(e Employee) []string { return e.Skills },
            c.ToSet[string](),
        ),
    ),
)
```

#### 4.8 Custom Collectors
Construct specialized collectors using `NewCollector` (with an intermediate accumulator type) or `NewSimpleCollector` (where intermediate accumulator type equals final result):

```go
// Custom collector computing a frequency map: map[string]int
freqCollector := NewCollector(
    func() map[string]int { return make(map[string]int) }, // Supplier
    func(acc map[string]int, item string) map[string]int {  // Accumulator
        acc[item]++
        return acc
    },
    func(a, b map[string]int) map[string]int {             // Combiner (parallel merge)
        for k, v := range b {
            a[k] += v
        }
        return a
    },
    func(acc map[string]int) map[string]int { return acc }, // Finisher
)

counts := Stream[string]{}.From("cat", "dog", "cat", "bird").Collect(freqCollector)
```

> [!IMPORTANT]
> To support parallel streams correctly:
> - `Supplier` must return a fresh, distinct accumulator on each call.
> - `Combiner` must be associative: `combiner(combiner(a, b), c)` must produce the same result as `combiner(a, combiner(b, c))`.

---

## 5. Parallel Processing

To execute a pipeline concurrently across multiple goroutines, simply call `.Parallel()` on the stream:

```go
totalPrimes := Stream[int]{}.FromRangeClosed(1, 1_000_000).
    Parallel().
    Filter(func(n int) bool { return isPrime(n) }).
    Collect(Collectors{}.Counting[int]())
```

### How Parallel Streams Work
1. **Goroutine Scaling**: Concurrency is automatically scaled to `runtime.GOMAXPROCS` (with a minimum of 2 workers).
2. **Sequential Upstream Pipeline**: Upstream stages such as `Map`, `Filter`, and `FlatMapSlice` run sequentially on the producing goroutine driving the stream sequence. Concurrency begins at the terminal operation; in `Collect`, only the accumulation phase runs on worker goroutines.
3. **Batch Accumulation**: `Stream.Collect` chunks elements from the producing goroutine into 64-element batches and dispatches them across worker goroutines. For each 64-element batch, a fresh accumulator is created via `Collector.Supplier`, accumulated across workers, and stored by batch index. `Collector.Supplier` must return a fresh accumulator on each call.
4. **Encounter-Order Combining**: Once all batches finish processing, partial accumulators are merged in encounter order (from batch index `0` to `n-1`) via `Collector.Combiner`, and the final result is produced via `Collector.Finisher`. `Collector.Combiner` must be associative to ensure deterministic reduction.
5. **Short-Circuiting**: In operations such as `AnyMatch`, `AllMatch`, and `FindAny`, an atomic trigger cancels remaining worker execution as soon as the outcome is determined.

---

## 6. Well-Known Usage Patterns & Recipes

### Recipe 1: Classic Map-Reduce Word Frequency Analysis

Map-Reduce is a foundational data processing paradigm consisting of three stages:
1. **Map (Tokenization & Normalization)**: Split raw text documents into words, normalize casing, and discard unwanted punctuation.
2. **Shuffle / Partition (Key Grouping)**: Group the stream of tokens by the word itself as the grouping key.
3. **Reduce (Aggregation)**: Aggregate and count occurrences per word key.

In Rays, this entire Map-Reduce workflow is expressed fluently in a single pipeline:

```go
documents := []string{
    "Hello world",
    "Hello Go streams",
    "World of streams and Go",
}

// Sequential Map-Reduce Word Count
wordCounts := Stream[string]{}.FromSlice(documents).
    FlatMapSlice(func(doc string) []string {
        return strings.Fields(strings.ToLower(doc))
    }).
    Collect(Collectors{}.GroupingBy(
        func(word string) string { return word },
        Collectors{}.Counting[string](),
    ))

// Result: map[string]int64{"hello": 2, "world": 2, "go": 2, "streams": 2, "of": 1, "and": 1}
```

#### Parallel Map-Reduce
To scale accumulation across multiple CPU cores for large datasets, simply insert `.Parallel()` into the pipeline:

```go
// Parallel Map-Reduce Word Count across all available CPU cores
parallelWordCounts := Stream[string]{}.FromSlice(largeCorpus).
    Parallel().
    FlatMapSlice(func(doc string) []string {
        return strings.Fields(strings.ToLower(doc))
    }).
    Collect(Collectors{}.GroupingBy(
        func(word string) string { return word },
        Collectors{}.Counting[string](),
    ))
```

> [!NOTE]
> Because upstream intermediate stages run sequentially on the producing goroutine, tokenization via `FlatMapSlice` is not parallel; only accumulation into the collector runs concurrently across worker goroutines.
>
> During `Stream.Collect`, the producing goroutine dispatches tokens in 64-element batches. For each batch, a fresh accumulator is created via `Collector.Supplier`, accumulated across workers, and stored by batch index. Once all batches finish processing, `Collector.Combiner` merges the partial maps in encounter order into the final result without locks or race conditions.

---

### Recipe 2: Multi-Level Grouping & Business Analytics

Real-world analytics pipelines often require multi-dimensional grouping and reduction (e.g. sales reports by product category, top transactions, and unique customer reach).

Given an e-commerce order dataset:

```go
type Order struct {
    ID       string
    Customer string
    Category string
    Amount   float64
}

orders := []Order{
    {ID: "O1", Customer: "Alice", Category: "Electronics", Amount: 299.99},
    {ID: "O2", Customer: "Bob", Category: "Books", Amount: 19.99},
    {ID: "O3", Customer: "Alice", Category: "Electronics", Amount: 49.99},
    {ID: "O4", Customer: "Charlie", Category: "Books", Amount: 35.50},
    {ID: "O5", Customer: "Bob", Category: "Electronics", Amount: 890.00},
}

var c Collectors
```

#### Total Revenue per Category
```go
revenueByCategory := Stream[Order]{}.FromSlice(orders).Collect(
    c.GroupingBy(
        func(o Order) string { return o.Category },
        c.SummingFloat64(func(o Order) float64 { return o.Amount }),
    ),
)
// map[string]float64{"Books": 55.49, "Electronics": 1239.98}
```

#### Highest Value Order per Category (`GroupingBy` + `MaxBy`)
```go
topOrderByCategory := Stream[Order]{}.FromSlice(orders).Collect(
    c.GroupingBy(
        func(o Order) string { return o.Category },
        c.MaxBy(func(a, b Order) int {
            if a.Amount < b.Amount {
                return -1
            } else if a.Amount > b.Amount {
                return 1
            }
            return 0
        }),
    ),
)
// topOrderByCategory["Electronics"].Get() -> Order O5 ($890.00)
```

#### Unique Customers per Category (`GroupingBy` + `Mapping` + `ToSet`)
```go
customersByCategory := Stream[Order]{}.FromSlice(orders).Collect(
    c.GroupingBy(
        func(o Order) string { return o.Category },
        c.Mapping(
            func(o Order) string { return o.Customer },
            c.ToSet[string](),
        ),
    ),
)
// customersByCategory["Electronics"].Size() -> 2 (Alice, Bob)
```

---

### Recipe 3: Single-Pass Two-Way Classification (Data Validation Pipeline)

When processing incoming streams (e.g., ETL batches or HTTP form submissions), separating valid records from rejected ones typically requires two filtering passes. With `PartitioningBySlice`, this is achieved in a single scan:

```go
type Record struct {
    ID    int
    Score int
}

records := []Record{
    {ID: 1, Score: 85},
    {ID: 2, Score: -5},
    {ID: 3, Score: 92},
    {ID: 4, Score: 0},
}

// Single pass validation
partitioned := Stream[Record]{}.FromSlice(records).Collect(
    Collectors{}.PartitioningBySlice(func(r Record) bool { return r.Score > 0 }),
)

validRecords   := partitioned[true]  // [{ID: 1, Score: 85}, {ID: 3, Score: 92}]
invalidRecords := partitioned[false] // [{ID: 2, Score: -5}, {ID: 4, Score: 0}]
```

---

### Recipe 4: Financial Summary Metrics Dashboard

For financial or operational dashboards, `SummarizingFloat64` computes the transaction volume, min, max, average, and transaction count in a single traversal:

```go
stats := Stream[Order]{}.FromSlice(orders).Collect(
    Collectors{}.SummarizingFloat64(func(o Order) float64 { return o.Amount }),
)

fmt.Printf("Total Volume:  $%.2f\n", stats.Sum)
fmt.Printf("Average Order: $%.2f\n", stats.Average())
fmt.Printf("Smallest:      $%.2f\n", stats.Min)
fmt.Printf("Largest:       $%.2f\n", stats.Max)
fmt.Printf("Transactions:  %d\n", stats.Count)
```

---

### Recipe 5: Building Safe Lookup Dictionaries (`ToMap`)

When indexing data into maps, key collisions can cause silent data loss or panics. `ToMap` accepts an optional merge function to resolve collisions gracefully:

```go
type UserRole struct {
    User string
    Role string
}

roles := []UserRole{
    {User: "alice", Role: "viewer"},
    {User: "bob", Role: "editor"},
    {User: "alice", Role: "admin"},
}

// Resolve collisions by keeping the latest assigned role
roleLookup := Stream[UserRole]{}.FromSlice(roles).Collect(
    Collectors{}.ToMap(
        func(ur UserRole) string { return ur.User },
        func(ur UserRole) string { return ur.Role },
        func(existing, newVal string) string { return newVal }, // collision rule
    ),
)
// roleLookup["alice"] -> "admin"
```

---

## 7. Resource Lifecycle & AutoClose

Streams support registering cleanup hooks via `.OnClose(...)` and closing resources using `.Close()`:

```go
s := Stream[Event]{}.FromChan(eventChannel).
    OnClose(func() {
        log.Println("Stream pipeline finished; releasing resources")
    })

// Consuming the stream automatically closes it upon completion
s.ForEach(processEvent)
```
