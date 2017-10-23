package state

type RecoveryLogWriter interface {
	WriteRecoveryLog([]byte) error
	WriteLostResourceLog([]byte) error
	DeleteRecoveryLog() error
}
type RecoveryLogReader interface {
	ReadRecoveryLog() (map[string]Instance, error)
}
