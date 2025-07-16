package gc

import (
	"unsafe"
)

/*
#include <stdlib.h>
#include <string.h>

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

// 清零内存
void clear_memory(void* ptr, size_t size) {
    if (ptr != NULL) {
        memset(ptr, 0, size);
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

// memset 清零内存
func memset(ptr unsafe.Pointer, value int, size int) {
	C.clear_memory(ptr, C.size_t(size))
}
