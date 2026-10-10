---
title: "6. Testing Framework"
weight: 6
---

# Testing

Rays eliminates the pain of testing singletons by integrating natively with Ginkgo and Gomega.

Import the testing package and testing suites directly into your test files.

## Automatic Container Isolation
The `TestingContainer` hooks into Ginkgo's lifecycle. After every `It()` block, the container is **wiped clean**. Singletons do not bleed between tests.

## Targeted Wiring

By
Instead of scanning the whole application, you manually register exactly the Beams you need for the unit under test using `AddBeamFor()`.

```go
import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    . "github.com/BeamFoundry/rays/pkg/testing"

    // Will detect all beams in  myproject/pkg/services
    . "github.com/mygroup/myproject/pkg/services"
)

var _ = Describe("UserService", func() {

    BeforeEach(func() {
        // Use the TestingContainer to mock YAML configuration for the test
        GetContainer().LoadConfigAt(map[string]any{
            "dsn": "mock://db",
        }, "database")

        // 1. Will load all the Stereotype imported in services
        GetContainer().EnableScanning()
    })

    It("should correctly wire the mocked database", func() {

        // 2. Initialize the isolated container
        err := GetContainer().Initialize()
        Expect(err).NotTo(HaveOccurred())

        // 3. Fetch the wired instance
        svc := GetBeamDefaultInstance[*UserService]()

        // 4. Assert
        Expect(svc).NotTo(BeNil())
        Expect(svc.db.dsn).To(Equal("mock://db"))
    })
})
```

## Advanced: Mocking Prototypes (`Foundry[T]`)

When a service depends on a `Foundry[T]`, you often want to mock the forged instances during tests rather than relying on the real Prototype lifecycle.

Because `Foundry[T]` is an interface, you can implement a mock version and register it in your test to override the framework's default factory behavior.

```go
import (
    "context"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
    . "github.com/BeamFoundry/rays/pkg/testing"

    // Will detect all beams in  myproject/pkg/middleware
    . "github.com/mygroup/myproject/pkg/middleware"
)

// 1. Create a Mock Foundry implementation
type MockSessionFoundry struct {
    Prototype
    Component // Register as a standard test Component
    MockSession *SessionState
}

func (m *MockSessionFoundry) WithContext(_ context.Context) (*SessionState, Error) {
    return m.MockSession, nil
}

var _ = Describe("AuthMiddleware", func() {
    It("should use the mocked foundry", func() {
        // 2. Will load all the Stereotype imported in services including AuthMiddleware
        GetContainer().EnableScanning()

        // 3. Instantiate and inject the mock instead of the real Prototype
        mockFoundry := &MockSessionFoundry{
            MockSession: &SessionState{userID: "mock-user-123"},
        }
        //
        AddBeamFor(mockFoundry)

        err := GetContainer().Initialize()
        Expect(err).NotTo(HaveOccurred())

        middleware := GetBeamDefaultInstance[*AuthMiddleware]()

        // When middleware.HandleHTTP is called, it will use the mock-user-123 session.
    })
})
```
