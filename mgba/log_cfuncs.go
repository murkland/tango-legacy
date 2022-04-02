package mgba

import "C"

//export tango_cgo_log
func tango_cgo_log(category C.int, level C.int, message *C.char) {
	logFunc(LogCategoryName(int(category)), int(level), C.GoString(message))
}
