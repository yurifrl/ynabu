package csv

import (
	"bytes"
	"fmt"
)

type Record interface {
	Date() string
	Payee() string
	Memo() string
	Amount() float64
}

type FilterFunc[T Record] func(T) bool

func Create[T Record](records []T, filter FilterFunc[T]) []byte {
	var buf bytes.Buffer
	buf.WriteString("Date,Payee,Memo,Amount\n")
	for _, r := range records {
		if filter == nil || filter(r) {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%.2f\n",
				r.Date(),
				r.Payee(),
				r.Memo(),
				r.Amount()))
		}
	}
	return buf.Bytes()
}
