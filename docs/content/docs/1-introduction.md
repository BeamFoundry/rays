---
title: "1. Introduction"
weight: 1
---

# Introduction & Philosophy

Go is famous for its simplicity and explicit dependency passing. However, as Go applications scale to hundreds
of services, repositories, and cross-cutting concerns, manual wiring (often done in massive `init()`
functions or `main.go`) becomes fragile and difficult to maintain.

**BeamFoundry Rays solves this by introducing a robust Inversion of Control (IoC) container.**

## Core Principles

1. **Annotation-Driven, but Type-Safe:** We use struct tags (`@:"Inject"`) for declaring dependencies,
  but rely on Go's strict typing to ensure the container wires the correct pointers.
2. **Encapsulation Matters:** Unlike manual DI which forces you to expose dependencies via public fields
  or constructor arguments, Rays natively injects into **private fields**. Your internal state remains protected.
3. **Two-Phase Bootstrapping:** To safely handle circular dependencies—a common pain point in large systems—Rays
  separates instantiation/injection from initialization.
4. **Context-Aware Scoping:** Singletons are great for services, but short-lived state requires a different approach.
  Rays introduces `Foundry[T]` to bridge the gap between global services and request-scoped prototypes.
