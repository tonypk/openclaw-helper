use serde_json::Value;
use crate::helper_ipc;

/// Tauri command that forwards JSON-RPC calls to the Go helper process.
#[tauri::command]
pub async fn helper_rpc(method: String, params: Option<Value>) -> Result<Value, String> {
    helper_ipc::call(&method, params)
        .await
        .map_err(|e| e.to_string())
}
