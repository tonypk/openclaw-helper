use serde_json::{json, Value};
use std::sync::OnceLock;
use tauri_plugin_shell::ShellExt;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};

#[cfg(not(windows))]
use tokio::net::UnixStream;

static PIPE_PATH: OnceLock<String> = OnceLock::new();
static RPC_ID: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);

/// Maximum retries for connecting to the Go helper.
const MAX_RETRIES: u32 = 3;
/// Delay between retries in milliseconds.
const RETRY_DELAY_MS: u64 = 500;

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

    // Wait for sidecar to be ready by polling with retries
    let pipe_clone = pipe.clone();
    std::thread::spawn(move || {
        for attempt in 1..=10 {
            std::thread::sleep(std::time::Duration::from_millis(300 * attempt));
            if try_ping_sync(&pipe_clone) {
                log::info!("Go helper sidecar ready after {} attempts", attempt);
                return;
            }
            log::info!("Waiting for Go helper sidecar (attempt {}/10)...", attempt);
        }
        log::warn!("Go helper sidecar did not respond to ping after 10 attempts");
    });

    log::info!("Go helper sidecar spawned on {}", PIPE_PATH.get().unwrap());
    Ok(())
}

/// Synchronous ping test to verify the sidecar is listening.
fn try_ping_sync(pipe_path: &str) -> bool {
    #[cfg(windows)]
    {
        use std::io::{Read, Write};
        let request = b"{\"jsonrpc\":\"2.0\",\"id\":0,\"method\":\"helper.ping\"}\n";
        match std::fs::OpenOptions::new().read(true).write(true).open(pipe_path) {
            Ok(mut pipe) => {
                if pipe.write_all(request).is_err() {
                    return false;
                }
                let mut buf = [0u8; 256];
                match pipe.read(&mut buf) {
                    Ok(n) if n > 0 => true,
                    _ => false,
                }
            }
            Err(_) => false,
        }
    }

    #[cfg(not(windows))]
    {
        use std::io::{Read, Write};
        use std::os::unix::net::UnixStream as StdUnixStream;
        let request = b"{\"jsonrpc\":\"2.0\",\"id\":0,\"method\":\"helper.ping\"}\n";
        match StdUnixStream::connect(pipe_path) {
            Ok(mut stream) => {
                stream.set_read_timeout(Some(std::time::Duration::from_secs(2))).ok();
                if stream.write_all(request).is_err() {
                    return false;
                }
                stream.shutdown(std::net::Shutdown::Write).ok();
                let mut buf = [0u8; 256];
                match stream.read(&mut buf) {
                    Ok(n) if n > 0 => true,
                    _ => false,
                }
            }
            Err(_) => false,
        }
    }
}

/// Call a JSON-RPC method on the Go helper with retry logic.
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

    let mut last_err: Option<Box<dyn std::error::Error + Send + Sync>> = None;

    for attempt in 0..MAX_RETRIES {
        if attempt > 0 {
            tokio::time::sleep(std::time::Duration::from_millis(RETRY_DELAY_MS * (attempt as u64))).await;
        }

        match send_request(&request_bytes).await {
            Ok(response) => {
                let resp: Value = serde_json::from_str(&response)?;

                if let Some(error) = resp.get("error") {
                    let msg = error.get("message")
                        .and_then(|m| m.as_str())
                        .unwrap_or("unknown error");
                    return Err(msg.to_string().into());
                }

                return Ok(resp.get("result").cloned().unwrap_or(Value::Null));
            }
            Err(e) => {
                log::warn!("[ipc] attempt {}/{} failed: {}", attempt + 1, MAX_RETRIES, e);
                last_err = Some(e);
            }
        }
    }

    Err(last_err.unwrap_or_else(|| "helper not reachable".to_string().into()))
}

/// Send a single RPC request and return the raw response string.
async fn send_request(request_bytes: &[u8]) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
    let pipe_path = PIPE_PATH.get().ok_or("helper not started")?;

    #[cfg(not(windows))]
    {
        let stream = UnixStream::connect(pipe_path).await?;
        let (reader, mut writer) = stream.into_split();
        writer.write_all(request_bytes).await?;
        writer.shutdown().await?;
        let mut buf_reader = BufReader::new(reader);
        let mut line = String::new();
        buf_reader.read_line(&mut line).await?;
        Ok(line)
    }

    #[cfg(windows)]
    {
        let stream = tokio::net::windows::named_pipe::ClientOptions::new()
            .open(pipe_path)?;
        let (reader, mut writer) = tokio::io::split(stream);
        writer.write_all(request_bytes).await?;
        writer.shutdown().await?;
        let mut buf_reader = BufReader::new(reader);
        let mut line = String::new();
        buf_reader.read_line(&mut line).await?;
        Ok(line)
    }
}
