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

// Optional: Initialize state when forged
func (s *SessionState) WithContext(ctx context.Context) Error {
    s.userID = ctx.Value("userID").(string)
    return nil
}

type AuthMiddleware struct {
    Service
    sessionFactory Foundry[*SessionState] `@:"Inject"`
    logger         Logging                `@:"Inject"`
}

func (m *AuthMiddleware) HandleHTTP(req *http.Request) {
    // 1. Forge a brand new instance
    session, _ := m.sessionFactory.Forge(req.Context())

    // 2. Use the isolated prototype
    m.logger.Info("Handle HTTP connection", "userID", session.userID)
}
```
