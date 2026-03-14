use serde_json::{json, Value};
use std::sync::OnceLock;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};


#[cfg(windows)]
use tokio::net::windows::named_pipe::ClientOptions;
#[cfg(not(windows))]
use tokio::net::UnixStream;

static PIPE_PATH: OnceLock<String> = OnceLock::new();
static RPC_ID: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);

/// Start the Go helper sidecar process.
pub fn start_helper(app: tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    // In production, the Go binary is bundled as a sidecar
    // For now, set the default pipe path
    #[cfg(windows)]
    PIPE_PATH.set(r"\\.\pipe\openclaw-helper".to_string()).ok();

    #[cfg(not(windows))]
    PIPE_PATH.set(format!("{}/openclaw-helper.sock", std::env::temp_dir().display())).ok();

    // TODO: Launch sidecar via tauri_plugin_shell::CommandBuilder
    // let sidecar = app.shell().sidecar("och-helper").unwrap();
    // sidecar.spawn().expect("failed to spawn helper");

    let _ = app; // suppress unused warning until sidecar is wired

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
        let mut buf_reader = BufReader::new(reader);
        let mut line = String::new();
        buf_reader.read_line(&mut line).await?;
        line
    };

    #[cfg(windows)]
    let response = {
        // Windows named pipe connection
        let pipe_path = PIPE_PATH.get().ok_or("helper not started")?;
        // TODO: Implement Windows named pipe client
        String::from("{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32603,\"message\":\"Windows pipe not yet implemented\"}}")
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
