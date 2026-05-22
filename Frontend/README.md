# 前端使用指南

## 功能介绍

前端页面提供以下能力：

- 普通对话：传统请求-响应模式。
- 流式对话：通过 SSE 实时展示模型响应。
- 文件上传：支持点击上传和拖拽上传，将文件写入知识库目录并触发索引构建。

## 启动完整环境

在项目根目录双击运行：

```powershell
start-all.cmd
```

或在 PowerShell 中运行：

```powershell
.\start-all.ps1
```

默认地址：

- 前端：`http://localhost:8080`
- 后端：`http://localhost:6872/api`
- Attu：`http://localhost:8000`

## 单独启动前端

```bash
cd Frontend
chmod +x start.sh
./start.sh
```

也可以在该目录下直接执行：

```bash
python -m http.server 8080
```

## 上传文件到知识库

支持的文件格式包括 PDF、TXT、MD、CSV、DOC、DOCX。上传成功后，后端会将文件保存到配置项 `file_dir` 指定的目录，并重建对应知识库索引。

## 后端 API

### 上传文件

- URL：`/api/upload`
- 方法：`POST`
- Content-Type：`multipart/form-data`
- 参数：`file`

响应示例：

```json
{
  "message": "OK",
  "data": {
    "fileName": "example.pdf",
    "filePath": "<configured-file-dir>/example.pdf",
    "fileSize": 1024000
  }
}
```

curl 示例：

```bash
curl -X POST http://localhost:6872/api/upload \
  -F "file=@/path/to/your/file.csv"
```

## 技术栈

- 原生 JavaScript
- Fetch API
- FormData API
- Drag and Drop API
- Server-Sent Events

## 注意事项

- 确保后端服务已启动，默认端口为 `6872`。
- 上传大文件时请耐心等待。
- 如果上传失败，请检查文件大小、文件格式、后端服务状态和保存目录权限。
