---
title: "Prototypes & Foundries"
weight: 5
---

# Prototypes & Foundries

If you need a new instance of an object every time it is requested (e.g., per HTTP request, per GRPC stream), you cannot use a Singleton. You need a **Prototype**.

## `Foundry[T]`

A Foundry is a type-safe factory automatically provided by the container for any struct marked with `Prototype`.

```go
import (
    "context"
    "net/http"
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type SessionState struct {
    Prototype
    userID string
}

// Optional: Receive context when forged (BeamPrototypeWithContext)
func (s *SessionState) WithContext(ctx context.Context) Error {
    if userID, ok := ctx.Value("userID").(string); ok {
        s.userID = userID
    }
    return nil
}

// Optional: Run post-construction validation after WithContext and embedded hooks (BeamPostConstruct)
func (s *SessionState) PostConstruct() Error {
    if s.userID == "" {
        return NewError("SessionError", "missing userID in context")
    }
    return nil
}

type AuthMiddleware struct {
    Service
    sessionFactory Foundry[*SessionState] `@:"Inject"`
    logger         Logging                `@:"Inject"`
}

func (m *AuthMiddleware) HandleHTTP(req *http.Request) {
    // 1. Forge a brand new instance
    session, err := m.sessionFactory.Forge(req.Context())
    if err != nil {
        m.logger.Error("Failed to forge session", "error", err)
        return
    }

    // 2. Use the isolated prototype
    m.logger.Info("Handle HTTP connection", "userID", session.userID)
}
```

## Prototype Initialization Lifecycle

When a `Prototype` Beam is forged via `Foundry[T].Forge(ctx)` or injected into a parent Beam, the container creates and injects a fresh instance, then invokes its initialization hooks in the following order:

1. **`BeamPrototypeWithContext.WithContext(ctx context.Context) Error`** (if implemented) — receives the `context.Context` passed to `Forge(ctx)`, enriched with `PrototypeContextParentType` and `PrototypeContextParentInstance`.
2. **`InnerBeamPostConstruct.PostConstruct(parent any) Error`** (if implemented on any anonymous embedded struct) — runs after `WithContext`, allowing embedded modules to inspect context-initialized fields on the parent.
3. **`BeamPostConstruct.PostConstruct() Error`** (if implemented) — runs last, after both `WithContext` and any embedded `InnerBeamPostConstruct` hooks have completed.

