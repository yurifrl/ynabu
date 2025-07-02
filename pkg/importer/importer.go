package importer

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/yurifrl/ynabu/pkg/config"
)

// Transaction represents a minimal interface for financial transactions.
// It matches the behaviour required by the importer without binding to a
// concrete implementation. Any struct that implements these getters can be
// imported.
type Transaction interface {
    Date() string
    Payee() string
    Memo() string
    Amount() float64
}

// Importer is responsible for bringing transactions from any source into
// the application. It is intentionally decoupled from CLI / HTTP details
// so it can be reused by both layers.
type Importer struct {
    cfg    *config.Config
    logger *log.Logger
}

// New returns a new Importer instance.
func New(cfg *config.Config, logger *log.Logger) *Importer {
    return &Importer{cfg: cfg, logger: logger}
}

// Import receives a slice of transactions and, for now, simply prints them.
// In the future this is the place where the persistence or forwarding logic
// should live.
func (i *Importer) Import(txns []Transaction) {
    for _, t := range txns {
        // For initial implementation we just print. Using fmt to avoid newlines miss.
        fmt.Printf("%s | %s | %.2f | %s\n", t.Date(), t.Payee(), t.Amount(), t.Memo())
    }
} 