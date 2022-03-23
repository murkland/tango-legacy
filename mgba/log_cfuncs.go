package mgba

import "C"

//export bbn6_cgo_log
func bbn6_cgo_log(category C.int, level C.int, message *C.char) {
	logFunc(LogCategoryName(int(category)), int(level), C.GoString(message))
}
