#!/bin/bash

# Linux音频设备测试脚本
# 用于验证APRS Agent在Linux上的音频设备检测

echo "=== Linux音频设备检测测试 ==="
echo "系统信息: $(uname -a)"
echo ""

# 检查必要的命令
echo "检查音频工具..."
for cmd in pactl amixer aplay arecord; do
    if command -v $cmd &> /dev/null; then
        echo "✓ $cmd 可用"
    else
        echo "✗ $cmd 不可用"
    fi
done
echo ""

# 检查PulseAudio
echo "=== PulseAudio设备检测 ==="
if command -v pactl &> /dev/null; then
    echo "输入设备 (sources):"
    pactl list short sources 2>/dev/null || echo "无法获取输入设备"
    echo ""
    
    echo "输出设备 (sinks):"
    pactl list short sinks 2>/dev/null || echo "无法获取输出设备"
    echo ""
    
    echo "默认设备:"
    echo "默认输入: $(pactl get-default-source 2>/dev/null || echo '未设置')"
    echo "默认输出: $(pactl get-default-sink 2>/dev/null || echo '未设置')"
else
    echo "PulseAudio未安装或不可用"
fi
echo ""

# 检查ALSA
echo "=== ALSA设备检测 ==="
if command -v amixer &> /dev/null; then
    echo "ALSA控制设备:"
    amixer scontrols 2>/dev/null || echo "无法获取ALSA控制设备"
    echo ""
fi

if command -v aplay &> /dev/null; then
    echo "ALSA播放设备:"
    aplay -l 2>/dev/null || echo "无法获取ALSA播放设备"
    echo ""
fi

if command -v arecord &> /dev/null; then
    echo "ALSA录音设备:"
    arecord -l 2>/dev/null || echo "无法获取ALSA录音设备"
    echo ""
fi

# 检查音频组权限
echo "=== 权限检查 ==="
if groups $USER | grep -q audio; then
    echo "✓ 用户在audio组中"
else
    echo "✗ 用户不在audio组中 (可能需要添加: sudo usermod -a -G audio $USER)"
fi

# 检查音频设备文件
echo "音频设备文件权限:"
ls -la /dev/snd/ 2>/dev/null || echo "无法访问 /dev/snd/"
echo ""

# 检查共享内存
echo "共享内存状态:"
ls -la /dev/shm/ | grep -E "(pulse|audio)" || echo "未找到音频相关共享内存文件"
echo ""

echo "=== 测试完成 ==="
echo "如果看到音频设备，说明系统音频配置正常"
echo "如果遇到问题，请检查上述输出并参考LINUX_DEPLOYMENT.md"
