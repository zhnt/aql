package gc

import (
	"unsafe"
)

/*
#include <stdlib.h>

// 分配对齐内存
void* allocate_aligned_memory(size_t size) {
    void* ptr;
    int result = posix_memalign(&ptr, 16, size); // 16字节对齐
    if (result != 0) {
        return NULL;
    }
    return ptr;
}

// 释放对齐内存
void free_aligned_memory(void* ptr) {
    if (ptr != NULL) {
        free(ptr);
    }
}
*/
import "C"

// allocateAlignedMemory 分配对齐内存
func allocateAlignedMemory(size uint32) unsafe.Pointer {
	ptr := C.allocate_aligned_memory(C.size_t(size))
	return unsafe.Pointer(ptr)
}

// freeAlignedMemory 释放对齐内存
func freeAlignedMemory(ptr unsafe.Pointer, size uint32) {
	C.free_aligned_memory(ptr)
}
