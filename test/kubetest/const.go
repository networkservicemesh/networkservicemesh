package kubetest

const (
	//StartLogsOf - start header of log transaction
	StartLogsOf = "Start logs of"
	//EndLogsOf - end header of log transaction
	EndLogsOf = "End logs of"
	//MaxTransactionLineWidth - limit of header line
	MaxTransactionLineWidth = 128
	//TransactionLogUnit - charter of header line
	TransactionLogUnit = '#'
	//StoreLogsInAnyCases means that logs should be stored\displayed in any case.
	StoreLogsInAnyCases = "STORE_LOGS_IN_ANY_CASES"
	//StorePodLogsInFiles - name of OS variable for enabling logging to file
	StorePodLogsInFiles = "STORE_POD_LOGS_IN_FILES"
	//StorePodLogsDir - name of OS variable for custom dir for logs
	StorePodLogsDir = "STORE_POD_LOGS_DIR"
	//DefaultLogDir - default name of dir for logs
	DefaultLogDir = "logs"
)
