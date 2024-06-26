package status

type Status struct {
	Info    Printer
	Warn    Printer
	Error   Printer
	Debug   Printer
	Spinner Spinner
}

type Printer interface {
	Println(string)
}

type Spinner interface {
	Start(string)
	Fail(string)
	Success(string)
	Text(string)
}
