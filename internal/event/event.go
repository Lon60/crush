package event

func SetNonInteractive(bool)       {}
func SetContinueBySessionID(bool)  {}
func SetContinueLastSession(bool)  {}
func Init()                        {}
func GetID() string                { return "" }
func Alias(string)                 {}
func Error(any, ...any)            {}
func Flush()                       {}
func send(string, ...any)          {}
func pairsToProps(...any) struct{} { return struct{}{} }
