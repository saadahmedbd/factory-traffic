package ports

import (
	"context"
	"factory-traffic/internal/domain"
)

// ControllerPort abstracts the communication with physical traffic hardware or simulated controllers.
type ControllerPort interface {
	// SendCommand transmits an instruction to the signal controller.
	SendCommand(ctx context.Context, cmd domain.ControllerCommand) error
	// HealthCheck returns true if hardware controller is responding.
	HealthCheck(ctx context.Context) error
}
