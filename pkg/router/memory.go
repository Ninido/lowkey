package router

import (
	"os"
)

// EstimateModelMemory estimates the memory needed for a model based on its file size
// GGUF file size is a reasonable proxy; actual usage varies by context and batch settings
func EstimateModelMemory(modelPath string) (int64, error) {
	info, err := os.Stat(modelPath)
	if err != nil {
		return 0, err
	}

	// Add 20% overhead for context, KV cache, etc.
	return info.Size() * 120 / 100, nil
}

// CanSpawnNewInstance checks if there's enough memory to spawn a new instance
func (r *Router) CanSpawnNewInstance(modelPath string) bool {
	needed, err := EstimateModelMemory(modelPath)
	if err != nil {
		return false
	}

	available, err := GetAvailableMemoryBytes()
	if err != nil {
		return false
	}

	// Keep reserve_pct free
	reserve := int64(r.cfg.MemoryReservePct) * available / 100
	return needed <= available - reserve
}
