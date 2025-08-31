package audio

import (
	"math"
	"sync"
)

// APRSProcessor APRS专用音频处理器
type APRSProcessor struct {
	mu sync.RWMutex

	// APRS音频参数
	noiseGateThreshold float64 // 噪声门限
	compressionRatio   float64 // 压缩比
	peakThreshold      float64 // 峰值门限

	// 音频处理状态
	isNoiseGateEnabled  bool
	isCompressorEnabled bool
	isLimiterEnabled    bool

	// 统计信息
	peakLevel     float64
	rmsLevel      float64
	clippingCount int
}

// NewAPRSProcessor 创建新的APRS音频处理器
func NewAPRSProcessor() *APRSProcessor {
	return &APRSProcessor{
		noiseGateThreshold: -40.0, // -40dB噪声门限
		compressionRatio:   4.0,   // 4:1压缩比
		peakThreshold:      -3.0,  // -3dB峰值门限

		isNoiseGateEnabled:  true,
		isCompressorEnabled: true,
		isLimiterEnabled:    true,
	}
}

// ProcessAudio 处理APRS音频数据
func (ap *APRSProcessor) ProcessAudio(input []byte, sampleRate int, channels int) []byte {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// 复制输入数据
	output := make([]byte, len(input))
	copy(output, input)

	// 计算音频电平
	ap.calculateLevels(output)

	// 应用噪声门限
	if ap.isNoiseGateEnabled {
		ap.applyNoiseGate(output)
	}

	// 应用压缩器
	if ap.isCompressorEnabled {
		ap.applyCompressor(output)
	}

	// 应用限幅器
	if ap.isLimiterEnabled {
		ap.applyLimiter(output)
	}

	return output
}

// calculateLevels 计算音频电平
func (ap *APRSProcessor) calculateLevels(data []byte) {
	if len(data) == 0 {
		ap.peakLevel = -96.0
		ap.rmsLevel = -96.0
		return
	}

	// 计算RMS值
	var sum float64
	sampleCount := len(data) / 2 // 假设16位音频

	var peak float64
	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		sampleAbs := math.Abs(float64(sample))
		sum += sampleAbs * sampleAbs

		if sampleAbs > peak {
			peak = sampleAbs
		}
	}

	rms := math.Sqrt(sum / float64(sampleCount))

	// 转换为分贝
	if rms > 0 {
		ap.rmsLevel = 20 * math.Log10(rms/32767.0)
	} else {
		ap.rmsLevel = -96.0
	}

	if peak > 0 {
		ap.peakLevel = 20 * math.Log10(peak/32767.0)
	} else {
		ap.peakLevel = -96.0
	}
}

// applyNoiseGate 应用噪声门限
func (ap *APRSProcessor) applyNoiseGate(data []byte) {
	threshold := math.Pow(10, ap.noiseGateThreshold/20.0) * 32767.0

	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		sampleAbs := math.Abs(float64(sample))

		if sampleAbs < threshold {
			// 低于门限，静音
			data[j] = 0
			data[j+1] = 0
		}
	}
}

// applyCompressor 应用压缩器
func (ap *APRSProcessor) applyCompressor(data []byte) {
	threshold := math.Pow(10, -20.0/20.0) * 32767.0 // -20dB门限
	ratio := ap.compressionRatio

	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		sampleAbs := math.Abs(float64(sample))

		if sampleAbs > threshold {
			// 计算压缩
			excess := sampleAbs - threshold
			compressedExcess := excess / ratio
			newSample := threshold + compressedExcess

			// 保持符号
			if sample < 0 {
				newSample = -newSample
			}

			// 限制在16位范围内
			if newSample > 32767 {
				newSample = 32767
			} else if newSample < -32768 {
				newSample = -32768
			}

			newSampleInt16 := int16(newSample)
			data[j] = byte(newSampleInt16 & 0xFF)
			data[j+1] = byte((newSampleInt16 >> 8) & 0xFF)
		}
	}
}

// applyLimiter 应用限幅器
func (ap *APRSProcessor) applyLimiter(data []byte) {
	threshold := math.Pow(10, ap.peakThreshold/20.0) * 32767.0

	for j := 0; j < len(data); j += 2 {
		sample := int16(data[j]) | int16(data[j+1])<<8
		sampleAbs := math.Abs(float64(sample))

		if sampleAbs > threshold {
			// 超过门限，限幅
			if sample > 0 {
				sample = int16(threshold)
			} else {
				sample = -int16(threshold)
			}

			ap.clippingCount++

			data[j] = byte(sample & 0xFF)
			data[j+1] = byte((sample >> 8) & 0xFF)
		}
	}
}

// SetNoiseGateThreshold 设置噪声门限
func (ap *APRSProcessor) SetNoiseGateThreshold(threshold float64) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.noiseGateThreshold = threshold
}

// SetCompressionRatio 设置压缩比
func (ap *APRSProcessor) SetCompressionRatio(ratio float64) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.compressionRatio = ratio
}

// SetPeakThreshold 设置峰值门限
func (ap *APRSProcessor) SetPeakThreshold(threshold float64) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.peakThreshold = threshold
}

// EnableNoiseGate 启用/禁用噪声门限
func (ap *APRSProcessor) EnableNoiseGate(enabled bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.isNoiseGateEnabled = enabled
}

// EnableCompressor 启用/禁用压缩器
func (ap *APRSProcessor) EnableCompressor(enabled bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.isCompressorEnabled = enabled
}

// EnableLimiter 启用/禁用限幅器
func (ap *APRSProcessor) EnableLimiter(enabled bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.isLimiterEnabled = enabled
}

// GetPeakLevel 获取峰值电平
func (ap *APRSProcessor) GetPeakLevel() float64 {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.peakLevel
}

// GetRMSLevel 获取RMS电平
func (ap *APRSProcessor) GetRMSLevel() float64 {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.rmsLevel
}

// GetClippingCount 获取限幅次数
func (ap *APRSProcessor) GetClippingCount() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.clippingCount
}

// ResetClippingCount 重置限幅计数
func (ap *APRSProcessor) ResetClippingCount() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.clippingCount = 0
}

// GetStatus 获取处理器状态
func (ap *APRSProcessor) GetStatus() map[string]interface{} {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	return map[string]interface{}{
		"noise_gate_enabled":   ap.isNoiseGateEnabled,
		"compressor_enabled":   ap.isCompressorEnabled,
		"limiter_enabled":      ap.isLimiterEnabled,
		"noise_gate_threshold": ap.noiseGateThreshold,
		"compression_ratio":    ap.compressionRatio,
		"peak_threshold":       ap.peakThreshold,
		"peak_level":           ap.peakLevel,
		"rms_level":            ap.rmsLevel,
		"clipping_count":       ap.clippingCount,
	}
}
