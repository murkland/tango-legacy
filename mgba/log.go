package mgba

/*
#include <mgba/core/log.h>

typedef void tango_mgba_mLogger_log_cb(struct mLogger*, int category, enum mLogLevel level, const char* format, va_list args);

void tango_cgo_log(int category, int level, char* buf);

void tango_cgo_log_trampoline(struct mLogger* logger, int category, enum mLogLevel level, const char* format, va_list args) {
	int size = vsnprintf(NULL, 0, format, args);
	char buf[size + 1];
	vsprintf(buf, format, args);
	tango_cgo_log(category, (int)level, buf);
}


void tango_mgba_mLogSetDefaultLogger_log(tango_mgba_mLogger_log_cb* log) {
	static struct mLogFilter logFilter;
	mLogFilterInit(&logFilter);
	static struct mLogger logger = {NULL, &logFilter};
	logger.log = log;
	mLogSetDefaultLogger(&logger);
}
*/
import "C"
import "unsafe"

type LogFunc func(category string, logLevel int, message string)

var logFunc LogFunc

func SetDefaultLogger(f LogFunc) {
	C.tango_mgba_mLogSetDefaultLogger_log((*C.tango_mgba_mLogger_log_cb)(unsafe.Pointer(C.tango_cgo_log_trampoline)))
	logFunc = f
}

func LogCategoryName(category int) string {
	return C.GoString(C.mLogCategoryName(C.int(category)))
}
