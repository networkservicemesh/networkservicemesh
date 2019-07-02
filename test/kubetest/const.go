package kubetest

const (
	//StartLogsOf - start header of log transaction
	StartLogsOf = "Start logs of"
	//EndLogsOf - end header of log transaction
	EndLogsOf = "End logs of"
	//MaxTransactionLineWidth - limit of header line
	MaxTransactionLineWidth = 128
	//TransactionLogUnit - charter of header line
	TransactionLogUnit      = '#'
	//WritePodLogsInFile - name of OS variable for enabling logging to file
	WritePodLogsInFile      = "WRITE_POD_LOGS_IN_FILES"
	//WritePodLogsDir - name of OS variable for custom dir for logs
	WritePodLogsDir         = "WRITE_POD_LOGS_DIR"
	//DefaultLogDir - default name of dir for logs
	DefaultLogDir = "logs"
)
