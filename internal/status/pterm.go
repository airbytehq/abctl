package status

import "github.com/pterm/pterm"

func NewPTerm() *Status {
	return &Status{
		Info:    pTermPrinter{pp: pterm.Info},
		Warn:    pTermPrinter{pp: pterm.Warning},
		Error:   pTermPrinter{pp: pterm.Error},
		Debug:   pTermPrinter{pp: pterm.Debug},
		Spinner: pTermSpinner{sp: &pterm.DefaultSpinner},
	}
}

type pTermPrinter struct {
	pp pterm.PrefixPrinter
}

func (p pTermPrinter) Println(s string) {
	p.pp.Println(s)
}

type pTermSpinner struct {
	sp *pterm.SpinnerPrinter
}

func (p pTermSpinner) Start(s string) {
	p.sp, _ = p.sp.Start(s)
}

func (p pTermSpinner) Fail(s string) {
	p.sp.Fail(s)
}

func (p pTermSpinner) Success(s string) {
	p.sp.Success(s)
}

func (p pTermSpinner) Text(s string) {
	p.sp.UpdateText(s)
}
