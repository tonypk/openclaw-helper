use serde_json::{json, Value};
use std::sync::OnceLock;
use tauri_plugin_shell::ShellExt;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};

#[cfg(not(windows))]
use tokio::net::UnixStream;

static PIPE_PATH: OnceLock<String> = OnceLock::new();
static RPC_ID: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);

/// Start the Go helper sidecar process.
pub fn start_helper(app: &tauri::App) -> Result<(), Box<dyn std::error::Error>> {
    // Determine pipe/socket path
    #[cfg(windows)]
    let pipe = r"\\.\pipe\openclaw-helper".to_string();

    #[cfg(not(windows))]
    let pipe = format!("{}/openclaw-helper.sock", std::env::temp_dir().display());

    // Remove stale socket file on Unix
    #[cfg(not(windows))]
    {
        let _ = std::fs::remove_file(&pipe);
    }

    PIPE_PATH.set(pipe.clone()).ok();

    // Launch the Go helper sidecar
    let sidecar = app.shell().sidecar("och-helper")
        .map_err(|e| format!("failed to create sidecar command: {}", e))?
        .args(["--pipe", &pipe]);

    let (mut rx, _child) = sidecar.spawn()
        .map_err(|e| format!("failed to spawn och-helper sidecar: {}", e))?;

    // Log sidecar output in background
    tauri::async_runtime::spawn(async move {
        use tauri_plugin_shell::process::CommandEvent;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    log::info!("[och-helper] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Stderr(line) => {
                    log::warn!("[och-helper] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Terminated(status) => {
                    log::warn!("[och-helper] process terminated: {:?}", status);
                    break;
                }
                CommandEvent::Error(err) => {
                    log::error!("[och-helper] error: {}", err);
                    break;
                }
                _ => {}
            }
        }
    });

    // Give sidecar a moment to start listening
    std::thread::sleep(std::time::Duration::from_millis(500));

    log::info!("Go helper sidecar started on {}", PIPE_PATH.get().unwrap());
    Ok(())
}

/// Call a JSON-RPC method on the Go helper.
pub async fn call(method: &str, params: Option<Value>) -> Result<Value, Box<dyn std::error::Error + Send + Sync>> {
    let id = RPC_ID.fetch_add(1, std::sync::atomic::Ordering::SeqCst);

    let request = json!({
        "jsonrpc": "2.0",
        "id": id,
        "method": method,
        "params": params.unwrap_or(Value::Null),
    });

    let mut request_bytes = serde_json::to_vec(&request)?;
    request_bytes.push(b'\n');

    // Connect and send
    #[cfg(not(windows))]
    let response = {
        let pipe_path = PIPE_PATH.get().ok_or("helper not started")?;
        let stream = UnixStream::connect(pipe_path).await?;
        let (reader, mut writer) = stream.into_split();
        writer.write_all(&request_bytes).await?;
        writer.shutdown().await?;
        let mut buf_reader = BufReader::new(reader);
        let mut line = String::new();
        buf_reader.read_line(&mut line).await?;
        line
    };

    #[cfg(windows)]
    let response = {
        let pipe_path = PIPE_PATH.get().ok_or("helper not started")?;
        // Connect to Windows Named Pipe via tokio
        let stream = tokio::net::windows::named_pipe::ClientOptions::new()
            .open(pipe_path)?;
        let (reader, mut writer) = tokio::io::split(stream);
        writer.write_all(&request_bytes).await?;
        writer.shutdown().await?;
        let mut buf_reader = BufReader::new(reader);
        let mut line = String::new();
        buf_reader.read_line(&mut line).await?;
        line
    };

    let resp: Value = serde_json::from_str(&response)?;

    if let Some(error) = resp.get("error") {
        let msg = error.get("message")
            .and_then(|m| m.as_str())
            .unwrap_or("unknown error");
        return Err(msg.to_string().into());
    }

    Ok(resp.get("result").cloned().unwrap_or(Value::Null))
}
