package notifier

import "github.com/ridopark/solo-adeventure/backend/internal/ports"

type Noop struct{}

func NewNoop() *Noop                       { return &Noop{} }
func (Noop) Notify(_ ports.NotifyEvent) {}
