package autoconfig

import (
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/tx"
)

// TransactionConfiguration provides default transaction Beams such as [TransactionManager].
type TransactionConfiguration struct {
	Configuration
}

var _, _ = any(&TransactionConfiguration{}).(Stereotype)

// TransactionManager provides the default [TransactionManager] Beam when none is explicitly registered.
func (this *TransactionConfiguration) TransactionManager(
	_ struct {
		ConditionalOnMissingBeam
	},
) TransactionManager {
	return DefaultTransactionManager{}.New()
}
