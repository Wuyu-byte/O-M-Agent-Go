package tools

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/gogf/gf/v2/frame/g"
)

type QueryGPUStatusInput struct{}

type GPUInfo struct {
	Index           string `json:"index"`
	Name            string `json:"name"`
	MemoryUsedMB    string `json:"memory_used_mb"`
	MemoryTotalMB   string `json:"memory_total_mb"`
	UtilizationGPU  string `json:"utilization_gpu_percent"`
	TemperatureC    string `json:"temperature_c"`
	AvailableMemory string `json:"available_memory_mb,omitempty"`
}

type QueryGPUStatusOutput struct {
	Success bool      `json:"success"`
	GPUs    []GPUInfo `json:"gpus,omitempty"`
	Message string    `json:"message"`
	Raw     string    `json:"raw,omitempty"`
}

func NewQueryGPUStatusTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_gpu_status",
		"Query local lab GPU status with nvidia-smi. Use it to inspect GPU model, memory usage, utilization, temperature, and recommend an idle GPU for experiments. This is read-only.",
		func(ctx context.Context, input *QueryGPUStatusInput, opts ...tool.Option) (string, error) {
			cmdCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			out, err := exec.CommandContext(cmdCtx, "nvidia-smi", "--query-gpu=index,name,memory.used,memory.total,utilization.gpu,temperature.gpu", "--format=csv,noheader,nounits").CombinedOutput()
			if err != nil {
				return marshalJSON(QueryGPUStatusOutput{
					Success: false,
					Message: "nvidia-smi is unavailable or no NVIDIA GPU is visible on this machine",
					Raw:     strings.TrimSpace(string(out)),
				})
			}
			gpus, parseErr := parseGPUCSV(string(out))
			if parseErr != nil {
				return marshalJSON(QueryGPUStatusOutput{
					Success: false,
					Message: fmt.Sprintf("failed to parse nvidia-smi output: %v", parseErr),
					Raw:     strings.TrimSpace(string(out)),
				})
			}
			return marshalJSON(QueryGPUStatusOutput{
				Success: true,
				GPUs:    gpus,
				Message: fmt.Sprintf("found %d GPU(s)", len(gpus)),
			})
		})
	if err != nil {
		panic(err)
	}
	return t
}

func parseGPUCSV(out string) ([]GPUInfo, error) {
	reader := csv.NewReader(strings.NewReader(out))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	gpus := make([]GPUInfo, 0, len(records))
	for _, record := range records {
		if len(record) < 6 {
			continue
		}
		used, _ := strconv.Atoi(strings.TrimSpace(record[2]))
		total, _ := strconv.Atoi(strings.TrimSpace(record[3]))
		available := ""
		if total > 0 {
			available = strconv.Itoa(total - used)
		}
		gpus = append(gpus, GPUInfo{
			Index:           strings.TrimSpace(record[0]),
			Name:            strings.TrimSpace(record[1]),
			MemoryUsedMB:    strings.TrimSpace(record[2]),
			MemoryTotalMB:   strings.TrimSpace(record[3]),
			UtilizationGPU:  strings.TrimSpace(record[4]),
			TemperatureC:    strings.TrimSpace(record[5]),
			AvailableMemory: available,
		})
	}
	return gpus, nil
}

type QueryPythonEnvInput struct{}

type QueryPythonEnvOutput struct {
	Success       bool   `json:"success"`
	PythonCommand string `json:"python_command,omitempty"`
	PythonVersion string `json:"python_version,omitempty"`
	TorchVersion  string `json:"torch_version,omitempty"`
	TorchCUDA     string `json:"torch_cuda,omitempty"`
	CUDAAvailable string `json:"cuda_available,omitempty"`
	Message       string `json:"message"`
	Raw           string `json:"raw,omitempty"`
}

func NewQueryPythonEnvTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_python_env",
		"Inspect local Python experiment environment, including Python version, PyTorch version, CUDA version, and whether torch.cuda is available. This is read-only.",
		func(ctx context.Context, input *QueryPythonEnvInput, opts ...tool.Option) (string, error) {
			pythonCmd, err := findPythonCommand(ctx)
			if err != nil {
				return marshalJSON(QueryPythonEnvOutput{
					Success: false,
					Message: err.Error(),
				})
			}
			versionOut, _ := runCommand(ctx, 8*time.Second, pythonCmd, "--version")
			probe := `import json
result={}
try:
    import torch
    result["torch_version"]=torch.__version__
    result["torch_cuda"]=str(torch.version.cuda)
    result["cuda_available"]=str(torch.cuda.is_available())
except Exception as e:
    result["torch_error"]=str(e)
print(json.dumps(result, ensure_ascii=False))`
			torchOut, _ := runCommand(ctx, 12*time.Second, pythonCmd, "-c", probe)
			result := QueryPythonEnvOutput{
				Success:       true,
				PythonCommand: pythonCmd,
				PythonVersion: strings.TrimSpace(versionOut),
				Message:       "python environment inspected",
				Raw:           strings.TrimSpace(torchOut),
			}
			var torch map[string]string
			if err := json.Unmarshal([]byte(strings.TrimSpace(torchOut)), &torch); err == nil {
				result.TorchVersion = torch["torch_version"]
				result.TorchCUDA = torch["torch_cuda"]
				result.CUDAAvailable = torch["cuda_available"]
				if torch["torch_error"] != "" {
					result.Message = "python is available, but PyTorch could not be imported"
				}
			}
			return marshalJSON(result)
		})
	if err != nil {
		panic(err)
	}
	return t
}

func findPythonCommand(ctx context.Context) (string, error) {
	candidates := []string{"python", "python3"}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "py", "python3"}
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate); err != nil {
			continue
		}
		if out, err := runCommand(ctx, 5*time.Second, candidate, "--version"); err == nil && strings.TrimSpace(out) != "" {
			return candidate, nil
		}
	}
	return "", errors.New("python executable was not found in PATH")
}

type ReadLabLogInput struct {
	Path      string `json:"path" jsonschema:"description=Log file path. Relative paths are resolved under lab_log_dir. Absolute paths must stay inside lab_log_dir."`
	TailLines int    `json:"tail_lines" jsonschema:"description=Number of trailing lines to read. Defaults to 200 and is capped at 1000."`
}

type ReadLabLogOutput struct {
	Success          bool     `json:"success"`
	Path             string   `json:"path,omitempty"`
	TailLines        int      `json:"tail_lines,omitempty"`
	Content          string   `json:"content,omitempty"`
	DetectedProblems []string `json:"detected_problems,omitempty"`
	Message          string   `json:"message"`
}

func NewReadLabLogTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"read_lab_log",
		"Read recent lines from an experiment or training log under the configured lab_log_dir and detect common errors such as CUDA OOM, missing packages, missing files, permission denied, disk full, or CUDA mismatch. This is read-only.",
		func(ctx context.Context, input *ReadLabLogInput, opts ...tool.Option) (string, error) {
			if input == nil || strings.TrimSpace(input.Path) == "" {
				return marshalJSON(ReadLabLogOutput{Success: false, Message: "path is required"})
			}
			logPath, err := resolveLabLogPath(ctx, input.Path)
			if err != nil {
				return marshalJSON(ReadLabLogOutput{Success: false, Message: err.Error()})
			}
			tailLines := input.TailLines
			if tailLines <= 0 {
				tailLines = 200
			}
			if tailLines > 1000 {
				tailLines = 1000
			}
			content, err := tailFile(logPath, tailLines)
			if err != nil {
				return marshalJSON(ReadLabLogOutput{Success: false, Path: logPath, Message: err.Error()})
			}
			return marshalJSON(ReadLabLogOutput{
				Success:          true,
				Path:             logPath,
				TailLines:        tailLines,
				Content:          content,
				DetectedProblems: detectLogProblems(content),
				Message:          "log file read successfully",
			})
		})
	if err != nil {
		panic(err)
	}
	return t
}

func resolveLabLogPath(ctx context.Context, inputPath string) (string, error) {
	baseVar, err := g.Cfg().Get(ctx, "lab_log_dir", "./logs")
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(baseVar.String())
	if err != nil {
		return "", err
	}
	path := inputPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("log path must stay inside lab_log_dir: %s", base)
	}
	return absPath, nil
}

func tailFile(path string, tailLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	lines := make([]string, 0, tailLines)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > tailLines {
			copy(lines, lines[1:])
			lines = lines[:tailLines]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func detectLogProblems(content string) []string {
	lower := strings.ToLower(content)
	checks := []struct {
		needle string
		label  string
	}{
		{"cuda out of memory", "CUDA_OUT_OF_MEMORY"},
		{"outofmemoryerror", "OUT_OF_MEMORY"},
		{"modulenotfounderror", "MISSING_PYTHON_PACKAGE"},
		{"importerror", "IMPORT_ERROR"},
		{"filenotfounderror", "FILE_NOT_FOUND"},
		{"no such file or directory", "FILE_NOT_FOUND"},
		{"permission denied", "PERMISSION_DENIED"},
		{"no space left on device", "DISK_FULL"},
		{"cuda error", "CUDA_ERROR"},
		{"driver version is insufficient", "CUDA_DRIVER_MISMATCH"},
		{"address already in use", "PORT_IN_USE"},
	}
	var problems []string
	seen := map[string]bool{}
	for _, check := range checks {
		if strings.Contains(lower, check.needle) && !seen[check.label] {
			seen[check.label] = true
			problems = append(problems, check.label)
		}
	}
	return problems
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, name, args...).CombinedOutput()
	return string(out), err
}

func marshalJSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
